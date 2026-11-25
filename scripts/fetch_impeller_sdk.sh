#!/usr/bin/env bash
# Downloads the prebuilt Impeller Standalone SDK pinned in IMPELLER_SHA.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SHA="$(cat "$ROOT/IMPELLER_SHA")"
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) PLATFORM=darwin-arm64 ;;
  Darwin-x86_64) PLATFORM=darwin-x64 ;;
  *) echo "Only macOS is supported for now." >&2; exit 1 ;;
esac
DEST="$ROOT/third_party/impeller_sdk"
rm -rf "$DEST" && mkdir -p "$DEST"
curl -sSfL -o "$DEST.zip" "https://storage.googleapis.com/flutter_infra_release/flutter/$SHA/$PLATFORM/impeller_sdk.zip"
unzip -q "$DEST.zip" -d "$DEST" && rm "$DEST.zip"
# The static lib is ~1 GB of debug info and unused; we link the dylib.
rm -f "$DEST/lib/libimpeller.a"
# Ship with an @rpath install name so Go binaries can find it.
install_name_tool -id @rpath/libimpeller.dylib "$DEST/lib/libimpeller.dylib"
echo "Impeller SDK ($SHA, $PLATFORM) ready in $DEST"
