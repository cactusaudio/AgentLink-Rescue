#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi
"$BIN" diagnose tun --json >/tmp/agentlink-tun-diagnose.json
/usr/bin/python3 -m json.tool /tmp/agentlink-tun-diagnose.json >/dev/null
"$BIN" field macbook-network-rescue --json >/tmp/agentlink-field.json
/usr/bin/python3 -m json.tool /tmp/agentlink-field.json >/dev/null
echo "dogfood_tun_detection OK"
