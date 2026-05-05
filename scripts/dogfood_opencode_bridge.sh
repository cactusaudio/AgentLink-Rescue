#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
TMPHOME="$(mktemp -d)"
trap 'rm -rf "$TMPHOME"' EXIT
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi
HOME="$TMPHOME" "$BIN" opencode doctor --json >/tmp/agentlink-opencode-doctor.json
/usr/bin/python3 -m json.tool /tmp/agentlink-opencode-doctor.json >/dev/null
HOME="$TMPHOME" "$BIN" opencode install --dry-run --json >/tmp/agentlink-opencode-install.json
/usr/bin/python3 -m json.tool /tmp/agentlink-opencode-install.json >/dev/null
HOME="$TMPHOME" "$BIN" opencode configure-local-gemma --dry-run --json >/tmp/agentlink-opencode-configure.json
/usr/bin/python3 -m json.tool /tmp/agentlink-opencode-configure.json >/dev/null
HOME="$TMPHOME" "$BIN" opencode install-plugin --json >/tmp/agentlink-opencode-plugin.json
/usr/bin/python3 -m json.tool /tmp/agentlink-opencode-plugin.json >/dev/null
/usr/bin/python3 - <<'PY'
import json
for path in ["/tmp/agentlink-opencode-doctor.json", "/tmp/agentlink-opencode-configure.json", "/tmp/agentlink-opencode-plugin.json"]:
    doc=json.load(open(path))
    status=doc.get("status")
    assert status not in {"installed_verified", "configured", "verified", "plugin_scaffold_available"}, (path, status, doc)
    assert status in {"experimental", "scaffold_available", "template_generated", "manual_merge_required", "not_verified"}, (path, status, doc)
plugin=json.load(open("/tmp/agentlink-opencode-plugin.json"))
assert plugin["status"] == "scaffold_available", plugin
PY
if [ -e "$TMPHOME/.config/opencode/agentlink-gemma.json" ]; then
  echo "opencode dry-run mutated config" >&2
  exit 1
fi
echo "dogfood_opencode_bridge OK"
