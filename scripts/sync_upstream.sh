#!/usr/bin/env bash
# Copies every non-Dart file of flutter/flutter at the pinned release tag
# (FLUTTER_TAG) into upstream/, byte for byte, at its upstream path.
#
# upstream/ is never edited by hand: flutter-go's real difference from Flutter
# is the Dart that is left out here and the Go that replaces it.
#
#   FLUTTER_SRC=~/Projetos/jonathanperis/flutter ./scripts/sync_upstream.sh          # re-sync
#   FLUTTER_SRC=~/Projetos/jonathanperis/flutter ./scripts/sync_upstream.sh --check  # verify only
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="${FLUTTER_SRC:-$HOME/Projetos/jonathanperis/flutter}"
TAG="$(cat "$ROOT/FLUTTER_TAG")"
DEST="$ROOT/upstream"

PIN="$(git -C "$SRC" rev-parse -q --verify "refs/tags/$TAG^{commit}")" || {
  echo "Tag $TAG is not in $SRC; run: git -C $SRC fetch origin tag $TAG" >&2; exit 1; }
# The prebuilt engine (Impeller SDK) must be the one this release ships.
ENGINE="$(git -C "$SRC" show "$PIN:bin/internal/engine.version" | tr -d '[:space:]')"
if [[ "$ENGINE" != "$(cat "$ROOT/IMPELLER_SHA")" ]]; then
  echo "IMPELLER_SHA must be $ENGINE, the engine of Flutter $TAG" >&2; exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
git -C "$SRC" archive --format=tar "$PIN" | tar -x -C "$TMP" --exclude='*.dart'
# Keep the module boundary: nothing in upstream/ belongs to flutter-go's
# Go module, so `go test ./...` never descends into it.
printf 'module upstream.flutter.invalid\n\ngo 1.24\n' > "$TMP/go.mod"
printf '%s\n%s\n' "$TAG" "$PIN" > "$TMP/UPSTREAM_COMMIT"

if [[ "${1:-}" == "--check" ]]; then
  # Compare committed blob hashes, so line-ending filters can't hide changes.
  ours="$(git -C "$ROOT" ls-files -s upstream | awk '{sub("^upstream/","",$4); print $1, $2, $4}' \
    | grep -v -e ' go.mod$' -e ' UPSTREAM_COMMIT$' | sort -k3)"
  theirs="$(git -C "$SRC" ls-tree -r "$PIN" | awk '$4 !~ /\.dart$/ && $1 != "160000" {print $1, $3, $4}' | sort -k3)"
  if diff <(echo "$ours") <(echo "$theirs") >/dev/null && diff -r -q --no-dereference "$TMP" "$DEST" >/dev/null; then
    echo "upstream/ matches flutter/flutter@$TAG (non-Dart files)"
  else
    echo "upstream/ differs from flutter/flutter@$TAG; re-run without --check" >&2; exit 1
  fi
  exit 0
fi

# Files committed upstream with CRLF must be staged as upstream's exact blobs
# (git add would normalize them under upstream's own `* text=auto`).
stage_exact() {
  git -C "$SRC" ls-tree -r "$PIN" | awk '$4 !~ /\.dart$/ && $1 != "160000"' | while read -r m _ h f; do
    if git -C "$SRC" cat-file blob "$h" | LC_ALL=C grep -q $'\r'; then
      nh="$(git -C "$SRC" cat-file blob "$h" | git -C "$ROOT" hash-object -w --stdin)"
      git -C "$ROOT" update-index --add --cacheinfo "$m,$nh,upstream/$f"
    fi
  done
}

rm -rf "$DEST"
mv "$TMP" "$DEST"
trap - EXIT
git -C "$ROOT" add -A -f upstream
stage_exact
echo "upstream/ = flutter/flutter@$TAG without .dart files ($(find "$DEST" -type f -o -type l | wc -l | tr -d ' ') files)"
