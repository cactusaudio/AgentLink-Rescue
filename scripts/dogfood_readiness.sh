#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/dist/Cactus-AgentLink-Rescue/bin/agentlink"
VERSION="0.4.4"
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/package.sh" >/dev/null
fi
"$BIN" version | grep "$VERSION" >/dev/null

TMPHOME="$(mktemp -d)"
trap 'rm -rf "$TMPHOME"' EXIT
mkdir -p "$TMPHOME/.codex"
printf 'model = "test"\n' > "$TMPHOME/.codex/config.toml"

HOME="$TMPHOME" "$BIN" readiness doctor --json >/tmp/agentlink-readiness.json
/usr/bin/python3 -m json.tool /tmp/agentlink-readiness.json >/dev/null
grep -q '"status"' /tmp/agentlink-readiness.json

HOME="$TMPHOME" "$BIN" dev doctor --json >/tmp/agentlink-dev.json
/usr/bin/python3 -m json.tool /tmp/agentlink-dev.json >/dev/null

HOME="$TMPHOME" "$BIN" last-good save --json >/tmp/agentlink-lastgood-save.json
/usr/bin/python3 -m json.tool /tmp/agentlink-lastgood-save.json >/dev/null
grep -q '"status": "saved"' /tmp/agentlink-lastgood-save.json

printf 'model = "broken"\n' > "$TMPHOME/.codex/config.toml"
HOME="$TMPHOME" "$BIN" last-good restore --last --yes --json >/tmp/agentlink-lastgood-restore.json
/usr/bin/python3 -m json.tool /tmp/agentlink-lastgood-restore.json >/dev/null
grep -q 'model = "test"' "$TMPHOME/.codex/config.toml"

HOME="$TMPHOME" "$BIN" support bundle --json >/tmp/agentlink-support-bundle.json
/usr/bin/python3 -m json.tool /tmp/agentlink-support-bundle.json >/dev/null
BUNDLE="$(/usr/bin/python3 - <<'PY'
import json
print(json.load(open('/tmp/agentlink-support-bundle.json')).get('bundlePath',''))
PY
)"
test -f "$BUNDLE"
unzip -l "$BUNDLE" | grep -q 'offline-readiness.json'

echo "readiness/last-good/support dogfood OK"
