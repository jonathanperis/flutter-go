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
# Upstream files deliberately left out: Flutter's AI-agent configuration
# and skills (they would configure agents working on flutter-go as if it
# were flutter/flutter).
EXCLUDED='^(\.agents/|\.claude/|\.gemini/|agent-artifacts/|CONTRIBUTING\.ai\.md$)|(^|/)(AGENTS|GEMINI|CLAUDE)\.md$'

PIN="$(git -C "$SRC" rev-parse -q --verify "refs/tags/$TAG^{commit}")" || {
  echo "Tag $TAG is not in $SRC; run: git -C $SRC fetch origin tag $TAG" >&2; exit 1; }
# The prebuilt engine (Impeller SDK) must be the one this release ships.
ENGINE="$(git -C "$SRC" show "$PIN:bin/internal/engine.version" | tr -d '[:space:]')"
if [[ "$ENGINE" != "$(cat "$ROOT/IMPELLER_SHA")" ]]; then
  echo "flutter_go/IMPELLER_SHA must be $ENGINE, the engine of Flutter $TAG" >&2; exit 1
fi

# "mode hash<TAB>path" for every non-Dart file of the tag (paths may contain
# spaces, so the path is only ever split off at the tab).
theirs() {
  git -C "$SRC" ls-tree -r "$PIN" | awk -F'\t' '{split($1, m, " "); if (m[1] != "160000" && $2 !~ /\.dart$/) print m[1] " " m[3] "\t" $2}' | awk -F'\t' -v re="$EXCLUDED" '$2 !~ re' | LC_ALL=C sort -t$'\t' -k2
}
# The same for what this repository's index holds outside flutter-go's paths.
ours_view() {
  git -C "$REPO" ls-files -s | awk -F'\t' -v re="$OURS" '{split($1, m, " "); if ($2 !~ re) print m[1] " " m[2] "\t" $2}' | LC_ALL=C sort -t$'\t' -k2
}

if theirs | cut -f2 | grep -Eq "$OURS"; then
  echo "upstream now has files in flutter-go's own paths:" >&2
  theirs | cut -f2 | grep -E "$OURS" >&2; exit 1
fi

if [[ "${1:-}" == "--check" ]]; then
  # Committed blob hashes, so line-ending filters can't hide a change...
  # ...and nothing modified or untracked outside flutter-go's own paths.
  dirty="$( { git -C "$REPO" diff --name-only; git -C "$REPO" ls-files --others --exclude-standard; } | grep -Ev "$OURS" || true)"
  if diff <(ours_view) <(theirs) >/dev/null && [[ -z "$dirty" ]]; then
    echo "repository = flutter/flutter@$TAG without .dart files, plus flutter-go's own files"
  else
    diff <(ours_view) <(theirs) | head -20 >&2 || true
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
# Drop the deliberately excluded files again (tracked or not, they stay out).
git -C "$SRC" ls-tree -r --name-only "$PIN" | grep -E "$EXCLUDED" | while IFS= read -r f; do rm -f -- "$f"; done
for d in .agents .claude .gemini agent-artifacts; do find "$d" -type l -delete 2>/dev/null; find "$d" -type d -empty -delete 2>/dev/null; done
theirs | cut -f2 | tr '\n' '\0' | xargs -0 git add -f --

# git add normalizes line endings under upstream's own `* text=auto`; a few
# upstream files are committed with CRLF anyway. Stage upstream's exact blob
# wherever the staged hash differs.
{ diff <(ours_view) <(theirs) || true; } | sed -n 's/^> //p' | while IFS=$'\t' read -r mh f; do
  m="${mh%% *}"; h="${mh##* }"
  git -C "$SRC" cat-file blob "$h" | git hash-object -w --stdin >/dev/null
  git update-index --add --cacheinfo "$m,$h,$f"
done
diff <(ours_view) <(theirs) >/dev/null || { echo "staging failed to match upstream" >&2; exit 1; }
echo "root = flutter/flutter@$TAG without .dart files ($(theirs | wc -l | tr -d ' ') files)"
