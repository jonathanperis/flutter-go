#!/usr/bin/env bash
# Regenerates reference layouts with real Flutter, then compares flutter-go.
# FLUTTER defaults to `flutter` on PATH.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FLUTTER="${FLUTTER:-flutter}"
(cd "$ROOT/difftest/flutter" && "$FLUTTER" test --reporter compact test/)
# References that need the Impeller backend (pixels): test/impeller/ writes
# only when run with --enable-impeller (Impeller on Vulkan/SwiftShader).
(cd "$ROOT/difftest/flutter" && "$FLUTTER" test --reporter compact --enable-impeller test/impeller/)
cd "$ROOT" && go test ./difftest -v -run "MatchFlutter" "$@"
