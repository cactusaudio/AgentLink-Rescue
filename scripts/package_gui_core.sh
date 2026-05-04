#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/dist/Cactus AgentLink Rescue.app"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.0-core-gui.zip"
INFO="$ROOT/gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Resources/Info.plist"

cd "$ROOT"
scripts/package.sh
scripts/build_gui.sh

rm -rf "$APP" "$ZIP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$INFO" "$APP/Contents/Info.plist"
cp "$ROOT/gui/build/CactusAgentLinkRescue" "$APP/Contents/MacOS/CactusAgentLinkRescue"
cp -R "$ROOT/dist/Cactus-AgentLink-Rescue" "$APP/Contents/Resources/agentlink"
chmod +x "$APP/Contents/MacOS/CactusAgentLinkRescue" "$APP/Contents/Resources/agentlink/bin/agentlink" "$APP/Contents/Resources/agentlink/rescue.sh" "$APP/Contents/Resources/agentlink/agentlink.command"
xattr -cr "$APP" 2>/dev/null || true

if find "$APP/Contents/Resources/agentlink/assets/models" -name '*.gguf' -print | grep .; then
  echo "core GUI must not contain GGUF model files" >&2
  exit 1
fi
if find "$APP/Contents/Resources/agentlink/assets/runtimes" -type f -name 'llama-cli' -print | grep .; then
  echo "core GUI must not contain llama.cpp runtime binaries" >&2
  exit 1
fi
find "$APP" -depth \( -name .DS_Store -o -name __MACOSX -o -name '._*' -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +

(
  cd "$ROOT/dist"
  COPYFILE_DISABLE=1 zip -r -X "$(basename "$ZIP")" "Cactus AgentLink Rescue.app" >/dev/null
)

if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
  echo "core GUI zip contains Finder metadata" >&2
  exit 1
fi

echo "core GUI app: $APP"
echo "core GUI zip: $ZIP"
/usr/bin/shasum -a 256 "$ZIP"
