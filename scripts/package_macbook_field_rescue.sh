#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="0.4.5"
FIELD_ROOT="$ROOT/dist/Cactus MacBook Network Rescue"
FIELD_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.5-macbook-field-gui-proxykit.zip"
APP_NAME="Cactus AgentLink Rescue.app"
APP="$FIELD_ROOT/$APP_NAME"
PROXYKIT_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v${VERSION}-brain-gui-gemma4-e4b-q4km-proxykit.zip"

cd "$ROOT"
scripts/package_gui_brain.sh

rm -rf "$FIELD_ROOT" "$FIELD_ZIP" "$ROOT/dist/field-staging"
mkdir -p "$FIELD_ROOT"

if [ -f "$PROXYKIT_ZIP" ]; then
  STAGE="$ROOT/dist/field-staging"
  mkdir -p "$STAGE"
  unzip -q "$PROXYKIT_ZIP" -d "$STAGE"
  cp -R "$STAGE/$APP_NAME" "$APP"
else
  cp -R "$ROOT/dist/$APP_NAME" "$APP"
fi

AGENTLINK_ROOT="$APP/Contents/Resources/agentlink"
cp "$ROOT/packaging/field-mode.macbook-network-rescue.json" "$AGENTLINK_ROOT/field-mode.json"

cat > "$FIELD_ROOT/RUN-FIRST.command" <<'SH'
#!/bin/bash
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
APP="$DIR/Cactus AgentLink Rescue.app"
AGENTLINK="$APP/Contents/Resources/agentlink"
xattr -cr "$DIR" 2>/dev/null || true
chmod +x "$APP/Contents/MacOS/CactusAgentLinkRescue" 2>/dev/null || true
chmod +x "$AGENTLINK/bin/agentlink" "$AGENTLINK/rescue.sh" "$AGENTLINK/agentlink.command" 2>/dev/null || true
find "$AGENTLINK/assets/runtimes/llama.cpp" -type f \( -name 'llama-*' -o -name '*.dylib' \) -exec chmod +x {} \; 2>/dev/null || true
open "$APP" || {
  echo "If the app is blocked, run:"
  echo "xattr -cr \"$DIR\""
  echo "cd \"$AGENTLINK\""
  echo "sudo ./bin/agentlink rescue --level safe"
}
SH

cat > "$FIELD_ROOT/README-MACBOOK-NETWORK-RESCUE.txt" <<'TXT'
Cactus MacBook Network Rescue

Use this copy when Wi-Fi/Ethernet connects but internet breaks after Clash Verge TUN mode.

1. Double-click RUN-FIRST.command, or double-click Cactus AgentLink Rescue.app.
2. Click Analyze Network.
3. If Safe Repair is recommended, copy/run the Terminal command:
   cd ".../Cactus AgentLink Rescue.app/Contents/Resources/agentlink"
   sudo ./bin/agentlink rescue --level safe
4. If not fixed, run Standard:
   sudo ./bin/agentlink rescue --level standard --yes
5. If still not fixed, export Support Bundle.
6. Deep repair only if instructed:
   sudo ./bin/agentlink rescue --level deep --yes
7. Do not re-enable Clash TUN until network is confirmed working.
8. AgentLink will not enable proxy/TUN automatically.
9. If the app cannot open:
   xattr -cr "Cactus MacBook Network Rescue"
   then open the app again.
TXT

cat > "$FIELD_ROOT/emergency-terminal-commands.txt" <<'TXT'
cd "<path-to-this-folder>/Cactus AgentLink Rescue.app/Contents/Resources/agentlink"

./bin/agentlink version
./bin/agentlink diagnose
./bin/agentlink field macbook-network-rescue
sudo ./bin/agentlink rescue --level safe
sudo ./bin/agentlink rescue --level standard --yes
sudo ./bin/agentlink rescue --level deep --yes
./bin/agentlink support bundle

Fallback:
sudo /bin/bash ./rescue.sh safe
sudo /bin/bash ./rescue.sh standard
sudo /bin/bash ./rescue.sh deep
TXT

chmod +x "$FIELD_ROOT/RUN-FIRST.command" "$AGENTLINK_ROOT/bin/agentlink" "$AGENTLINK_ROOT/rescue.sh" "$AGENTLINK_ROOT/agentlink.command"
find "$FIELD_ROOT" -depth \( -name .DS_Store -o -name __MACOSX -o -name '._*' -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +

if [ ! -f "$AGENTLINK_ROOT/field-mode.json" ]; then
  echo "field package missing field-mode.json" >&2
  exit 1
fi
if ! find "$AGENTLINK_ROOT/assets/models" -name 'gemma-4-E4B-it-Q4_K_M.gguf' -print | grep . >/dev/null; then
  echo "field package missing Gemma model" >&2
  exit 1
fi
if ! find "$AGENTLINK_ROOT/assets/runtimes" -type f -name 'llama-cli' -print | grep . >/dev/null; then
  echo "field package missing llama-cli" >&2
  exit 1
fi

(
  cd "$ROOT/dist"
  COPYFILE_DISABLE=1 zip -r -X "$(basename "$FIELD_ZIP")" "Cactus MacBook Network Rescue" >/dev/null
)

if unzip -l "$FIELD_ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
  echo "field rescue zip contains Finder metadata" >&2
  exit 1
fi

echo "field rescue folder: $FIELD_ROOT"
echo "field rescue zip: $FIELD_ZIP"
/usr/bin/shasum -a 256 "$FIELD_ZIP"
