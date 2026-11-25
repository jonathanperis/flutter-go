#!/usr/bin/env bash
# The repository root is flutter/flutter at the pinned release tag
# (flutter_go/FLUTTER_TAG) with every .dart file left out, byte for byte, at
# upstream's paths. flutter-go's own files are only:
#   - go.mod, go.sum
#   - *.go files, placed beside the Dart files they replace
#   - flutter_go/ (tools, tests, docs, the runner: things upstream has no
#     counterpart for)
#   - .github/README.md (what GitHub shows on the repository page)
#
#   FLUTTER_SRC=~/Projetos/jonathanperis/flutter flutter_go/scripts/sync_upstream.sh          # re-sync
#   FLUTTER_SRC=~/Projetos/jonathanperis/flutter flutter_go/scripts/sync_upstream.sh --check  # verify only
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)" # flutter_go/
REPO="$(cd "$ROOT/.." && pwd)"
SRC="${FLUTTER_SRC:-$HOME/Projetos/jonathanperis/flutter}"
TAG="$(cat "$ROOT/FLUTTER_TAG")"
OURS='^(go\.mod|go\.sum|flutter_go/.*|.*\.go|\.github/README\.md)$'

PIN="$(git -C "$SRC" rev-parse -q --verify "refs/tags/$TAG^{commit}")" || {
  echo "Tag $TAG is not in $SRC; run: git -C $SRC fetch origin tag $TAG" >&2; exit 1; }
# The prebuilt engine (Impeller SDK) must be the one this release ships.
ENGINE="$(git -C "$SRC" show "$PIN:bin/internal/engine.version" | tr -d '[:space:]')"
if [[ "$ENGINE" != "$(cat "$ROOT/IMPELLER_SHA")" ]]; then
  echo "flutter_go/IMPELLER_SHA must be $ENGINE, the engine of Flutter $TAG" >&2; exit 1
fi

theirs() { git -C "$SRC" ls-tree -r "$PIN" | awk '$4 !~ /\.dart$/ && $1 != "160000" {print $1, $3, $4}' | sort -k3; }

if theirs | awk '{print $3}' | grep -Eq "$OURS"; then
  echo "upstream now has files in flutter-go's own paths:" >&2
  theirs | awk '{print $3}' | grep -E "$OURS" >&2; exit 1
fi

if [[ "${1:-}" == "--check" ]]; then
  # Committed blob hashes, so line-ending filters can't hide a change...
  ours_view="$(git -C "$REPO" ls-files -s | awk '{print $1, $2, $4}' | awk -v re="$OURS" '$3 !~ re' | sort -k3)"
  # ...and nothing modified or untracked outside flutter-go's own paths.
  dirty="$( { git -C "$REPO" diff --name-only; git -C "$REPO" ls-files --others --exclude-standard; } | grep -Ev "$OURS" || true)"
  if diff <(echo "$ours_view") <(theirs) >/dev/null && [[ -z "$dirty" ]]; then
    echo "repository = flutter/flutter@$TAG without .dart files, plus flutter-go's own files"
  else
    diff <(echo "$ours_view") <(theirs) | head -20 >&2 || true
    [[ -n "$dirty" ]] && { echo "modified or untracked upstream paths:" >&2; echo "$dirty" | head -20 >&2; }
    echo "differs from flutter/flutter@$TAG; re-run without --check" >&2; exit 1
  fi
  exit 0
fi

cd "$REPO"
# Remove the old upstream files (everything that isn't ours), then extract
# the tag without its Dart files.
git ls-files -z | tr '\0' '\n' | grep -Ev "$OURS" | tr '\n' '\0' | xargs -0 git rm -q --cached --ignore-unmatch --
git -C "$SRC" ls-tree -r --name-only "$PIN" | grep -v '\.dart$' | while IFS= read -r f; do rm -f -- "$f"; done
git -C "$SRC" archive --format=tar "$PIN" | tar -x -C "$REPO" --exclude='*.dart'
git -C "$SRC" ls-tree -r --name-only -z "$PIN" | tr '\0' '\n' | grep -v '\.dart$' | tr '\n' '\0' | xargs -0 git add -f --

# Files committed upstream with CRLF must be staged as upstream's exact blobs
# (git add normalizes them under upstream's own `* text=auto`).
theirs | while read -r m h f; do
  if git -C "$SRC" cat-file blob "$h" | LC_ALL=C grep -q $'\r'; then
    nh="$(git -C "$SRC" cat-file blob "$h" | git hash-object -w --stdin)"
    git update-index --add --cacheinfo "$m,$nh,$f"
  fi
done
echo "root = flutter/flutter@$TAG without .dart files ($(theirs | wc -l | tr -d ' ') files)"
