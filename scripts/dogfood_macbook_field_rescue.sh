#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.0-macbook-field-gui-proxykit.zip"
if [ ! -f "$ZIP" ]; then
  "$ROOT/scripts/package_macbook_field_rescue.sh" >/dev/null
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
unzip -q "$ZIP" -d "$TMP"

FOLDER="$TMP/Cactus MacBook Network Rescue"
APP="$FOLDER/Cactus AgentLink Rescue.app"
AGENTLINK="$APP/Contents/Resources/agentlink"
BIN="$AGENTLINK/bin/agentlink"

xattr -cr "$FOLDER" 2>/dev/null || true
test -d "$APP"
test -x "$BIN"
test -x "$FOLDER/RUN-FIRST.command"
test -f "$FOLDER/README-MACBOOK-NETWORK-RESCUE.txt"
test -f "$FOLDER/emergency-terminal-commands.txt"
test -f "$AGENTLINK/field-mode.json"

"$BIN" version | grep '0.5.0'
"$BIN" field macbook-network-rescue --json > "$TMP/field.json"
/usr/bin/python3 -m json.tool "$TMP/field.json" >/dev/null
/usr/bin/python3 - "$TMP/field.json" <<'PY'
import json, sys
data=json.load(open(sys.argv[1]))
assert data["fieldMode"] == "macbook-network-rescue", data
assert data["commands"]["tun"] == "sudo ./bin/agentlink rescue --level tun --yes", data
assert data["commands"]["standardSystemReset"] == "sudo ./bin/agentlink rescue --level standard-system-reset --yes", data
assert "supportBundle" in data["commands"], data
PY

"$BIN" guided rescue --target network --dry-run --json > "$TMP/guided-network.json"
/usr/bin/python3 -m json.tool "$TMP/guided-network.json" >/dev/null
"$BIN" brain doctor --json > "$TMP/brain-doctor.json"
/usr/bin/python3 -m json.tool "$TMP/brain-doctor.json" >/dev/null
"$BIN" installer inspect clash-verge-rev --json > "$TMP/clash-inspect.json"
/usr/bin/python3 -m json.tool "$TMP/clash-inspect.json" >/dev/null

grep -q 'sudo ./bin/agentlink rescue --level safe' "$FOLDER/emergency-terminal-commands.txt"
grep -q 'sudo ./bin/agentlink rescue --level tun --yes' "$FOLDER/emergency-terminal-commands.txt"
grep -q 'sudo ./bin/agentlink rescue --level standard-system-reset --yes' "$FOLDER/emergency-terminal-commands.txt"
grep -q 'AgentLink will not enable proxy/TUN automatically' "$FOLDER/README-MACBOOK-NETWORK-RESCUE.txt"

if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
  echo "field rescue zip contains Finder metadata" >&2
  exit 1
fi

echo "dogfood MacBook field rescue OK: $TMP"
