#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
TMPHOME="$(mktemp -d)"
trap 'rm -rf "$TMPHOME"' EXIT
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi
HOME="$TMPHOME" "$BIN" ticket create --type clash-tun-fix --json >/tmp/agentlink-ticket.json
/usr/bin/python3 -m json.tool /tmp/agentlink-ticket.json >/dev/null
TICKET_DIR="$(/usr/bin/python3 - <<'PY'
import json
print(json.load(open('/tmp/agentlink-ticket.json'))['directory'])
PY
)"
test -x "$TICKET_DIR/Run-Clash-TUN-Fix.command"
grep 'sudo ./bin/agentlink rescue --level tun --yes' "$TICKET_DIR/Run-Clash-TUN-Fix.command" >/dev/null
if grep -E 'sudo -S|PASSWORD|password=' "$TICKET_DIR/Run-Clash-TUN-Fix.command" >/dev/null; then
  echo "ticket contains unsafe password handling" >&2
  exit 1
fi
echo "dogfood_terminal_ticket OK"
