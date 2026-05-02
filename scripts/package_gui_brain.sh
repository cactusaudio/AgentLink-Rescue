#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/dist/Cactus AgentLink Rescue.app"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.0-brain-gui-qwen3-4b-q4km.zip"
INFO="$ROOT/gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Resources/Info.plist"

cd "$ROOT"
scripts/package_brain.sh
scripts/build_gui.sh

rm -rf "$APP" "$ZIP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$INFO" "$APP/Contents/Info.plist"
cp "$ROOT/gui/build/CactusAgentLinkRescue" "$APP/Contents/MacOS/CactusAgentLinkRescue"
cp -R "$ROOT/dist/Cactus-AgentLink-Rescue" "$APP/Contents/Resources/agentlink"
chmod +x "$APP/Contents/MacOS/CactusAgentLinkRescue" "$APP/Contents/Resources/agentlink/bin/agentlink" "$APP/Contents/Resources/agentlink/rescue.sh" "$APP/Contents/Resources/agentlink/agentlink.command"
chmod +x "$APP"/Contents/Resources/agentlink/assets/runtimes/llama.cpp/*/llama-* "$APP"/Contents/Resources/agentlink/assets/runtimes/llama.cpp/*/*.dylib 2>/dev/null || true
xattr -cr "$APP" 2>/dev/null || true

if ! find "$APP/Contents/Resources/agentlink/assets/models" -name 'Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf' -print | grep .; then
  echo "brain GUI missing Qwen GGUF" >&2
  exit 1
fi
if ! find "$APP/Contents/Resources/agentlink/assets/runtimes" -type f -name 'llama-cli' -print | grep .; then
  echo "brain GUI missing llama-cli" >&2
  exit 1
fi
if [ ! -f "$APP/Contents/Resources/agentlink/assets/manifests/manifest.lock.json" ]; then
  echo "brain GUI missing manifest.lock.json" >&2
  exit 1
fi
find "$APP" -depth \( -name .DS_Store -o -name __MACOSX -o -name '._*' -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +

(
  cd "$ROOT/dist"
  COPYFILE_DISABLE=1 zip -r -X "$(basename "$ZIP")" "Cactus AgentLink Rescue.app" >/dev/null
)

if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
  echo "brain GUI zip contains Finder metadata" >&2
  exit 1
fi

echo "brain GUI app: $APP"
echo "brain GUI zip: $ZIP"
/usr/bin/shasum -a 256 "$ZIP"
