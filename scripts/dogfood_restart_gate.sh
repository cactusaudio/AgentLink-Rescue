#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi
"$BIN" restart-gate prepare --json >/tmp/agentlink-restart-gate.json || true
/usr/bin/python3 -m json.tool /tmp/agentlink-restart-gate.json >/dev/null
"$BIN" restart-gate verify --json >/tmp/agentlink-restart-gate-verify.json || true
/usr/bin/python3 -m json.tool /tmp/agentlink-restart-gate-verify.json >/dev/null
echo "dogfood_restart_gate OK"
