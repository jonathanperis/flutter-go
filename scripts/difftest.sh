#!/usr/bin/env bash
# Regenerates reference layouts with real Flutter, then compares flutter-go.
# FLUTTER defaults to `flutter` on PATH.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FLUTTER="${FLUTTER:-flutter}"
(cd "$ROOT/difftest/flutter" && "$FLUTTER" test --reporter compact test/)
cd "$ROOT" && go test ./difftest -v -run "MatchFlutter" "$@"
