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
if [ -e "$TMPHOME/.config/opencode/agentlink-gemma.json" ]; then
  echo "opencode dry-run mutated config" >&2
  exit 1
fi
echo "dogfood_opencode_bridge OK"
