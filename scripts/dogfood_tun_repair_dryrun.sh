#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi
"$BIN" rescue --level tun --dry-run --json >/tmp/agentlink-tun-dryrun.json
/usr/bin/python3 -m json.tool /tmp/agentlink-tun-dryrun.json >/dev/null
/usr/bin/python3 - /tmp/agentlink-tun-dryrun.json <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
actions=doc.get("actions", [])
stages=[a.get("stage","") for a in actions]
for stage in ["1_stop_runtime_processes", "4_kick_network_extension_daemons", "7_dns_airdrop_awdl_repair"]:
    assert stage in stages, (stage, stages)
order=["1_stop_runtime_processes","2_bootout_launch_items","3_quarantine_residue","4_kick_network_extension_daemons","5_down_stale_utun","6_active_wifi_repair","7_dns_airdrop_awdl_repair"]
last=-1
for stage in order:
    if stage in stages:
        idx=stages.index(stage)
        assert idx >= last, (stage, stages)
        last=idx
text=json.dumps(doc)
assert "route add default 198.18.0.1" not in text, text
assert "standard.location" not in text, text
assert "route.flush" not in text, text
PY
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
