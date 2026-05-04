#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="$ROOT/dist/Cactus AgentLink Rescue.app"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.0-brain-gui-gemma4-e4b-q4km.zip"
PROXYKIT_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.0-brain-gui-gemma4-e4b-q4km-proxykit.zip"
INFO="$ROOT/gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Resources/Info.plist"
. "$ROOT/scripts/lib/asset_cache.sh"

cd "$ROOT"
scripts/package_brain.sh
scripts/build_gui.sh

rm -rf "$APP" "$ZIP" "$PROXYKIT_ZIP" "$ROOT/dist/proxykit-staging"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$INFO" "$APP/Contents/Info.plist"
cp "$ROOT/gui/build/CactusAgentLinkRescue" "$APP/Contents/MacOS/CactusAgentLinkRescue"
cp -R "$ROOT/dist/Cactus-AgentLink-Rescue" "$APP/Contents/Resources/agentlink"
chmod +x "$APP/Contents/MacOS/CactusAgentLinkRescue" "$APP/Contents/Resources/agentlink/bin/agentlink" "$APP/Contents/Resources/agentlink/rescue.sh" "$APP/Contents/Resources/agentlink/agentlink.command"
chmod +x "$APP"/Contents/Resources/agentlink/assets/runtimes/llama.cpp/*/llama-* "$APP"/Contents/Resources/agentlink/assets/runtimes/llama.cpp/*/*.dylib 2>/dev/null || true
xattr -cr "$APP" 2>/dev/null || true

if ! find "$APP/Contents/Resources/agentlink/assets/models" -name 'gemma-4-E4B-it-Q4_K_M.gguf' -print | grep .; then
  echo "brain GUI missing Gemma GGUF" >&2
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

case "$(uname -m)" in
  arm64) INSTALLER_ARCH="macos-arm64" ;;
  x86_64) INSTALLER_ARCH="macos-amd64" ;;
  *) INSTALLER_ARCH="" ;;
esac
CLASH_DMG_DIR=""
if [ -n "$INSTALLER_ARCH" ]; then
  CLASH_DMG_DIR="$(resolve_clash_dmg_dir "$INSTALLER_ARCH" 2>/dev/null || true)"
fi
if [ -n "$CLASH_DMG_DIR" ]; then
  STAGE="$ROOT/dist/proxykit-staging"
  mkdir -p "$STAGE"
  cp -R "$APP" "$STAGE/Cactus AgentLink Rescue.app"
  mkdir -p "$STAGE/Cactus AgentLink Rescue.app/Contents/Resources/agentlink/assets/installers/clash-verge-rev/$INSTALLER_ARCH"
  cp "$CLASH_DMG_DIR"/*.dmg "$STAGE/Cactus AgentLink Rescue.app/Contents/Resources/agentlink/assets/installers/clash-verge-rev/$INSTALLER_ARCH/"
  if [ -f "$(dirname "$CLASH_DMG_DIR")/manifest.lock.json" ]; then
    cp "$(dirname "$CLASH_DMG_DIR")/manifest.lock.json" "$STAGE/Cactus AgentLink Rescue.app/Contents/Resources/agentlink/assets/installers/clash-verge-rev/"
  fi
  find "$STAGE" -depth \( -name .DS_Store -o -name __MACOSX -o -name '._*' -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +
  (
    cd "$STAGE"
    COPYFILE_DISABLE=1 zip -r -X "$(basename "$PROXYKIT_ZIP")" "Cactus AgentLink Rescue.app" >/dev/null
  )
  mv "$STAGE/$(basename "$PROXYKIT_ZIP")" "$PROXYKIT_ZIP"
  if unzip -l "$PROXYKIT_ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
    echo "proxykit GUI zip contains Finder metadata" >&2
    exit 1
  fi
  echo "proxykit GUI zip: $PROXYKIT_ZIP"
  /usr/bin/shasum -a 256 "$PROXYKIT_ZIP"
fi
