#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi
"$BIN" rescue --level tun --dry-run --json >/tmp/agentlink-tun-dryrun.json
/usr/bin/python3 -m json.tool /tmp/agentlink-tun-dryrun.json >/dev/null
grep -E '"level": "tun"|"DryRun": true|"dryRun": true' /tmp/agentlink-tun-dryrun.json >/dev/null || {
  echo "TUN dry-run did not identify level/dry-run" >&2
  cat /tmp/agentlink-tun-dryrun.json >&2
  exit 1
}
if grep -E 'standard.location|route.flush' /tmp/agentlink-tun-dryrun.json >/dev/null; then
  echo "TUN dry-run contains broad standard reset actions" >&2
  exit 1
fi
echo "dogfood_tun_repair_dryrun OK"
