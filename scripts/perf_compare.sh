#!/usr/bin/env bash
# Runs the same 100×100 grid benchmark in flutter-go and in real Flutter
# (profile mode, macOS). Requires full Xcode for the Flutter side.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FLUTTER="${FLUTTER:-flutter}"
echo "== flutter-go"
(cd "$ROOT" && PERF=1 go test ./perf -count=1 -v -run Budget | grep -E "elements|p50")
cd "$ROOT/perf/flutter"
for mode in false true; do
  echo "== flutter (profile), layout=$mode"
  "$FLUTTER" build macos --profile --dart-define=LAYOUT=$mode >/dev/null
  ./build/macos/Build/Products/Profile/grid_bench.app/Contents/MacOS/grid_bench
done
