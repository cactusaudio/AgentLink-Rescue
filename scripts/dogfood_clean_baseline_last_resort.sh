#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/agentlink"
# v0.5.1 brain-aware rescue dogfood: clean-baseline is a last resort, not Brain auto-execution.
if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
fi

"$BIN" rescue --level clean-baseline --dry-run --json > /tmp/agentlink-clean-baseline-dryrun.json
/usr/bin/python3 - /tmp/agentlink-clean-baseline-dryrun.json <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
assert doc["level"] == "clean-baseline", doc
assert doc["dryRun"] is True, doc
text=json.dumps(doc).lower()
for forbidden in ["route.flush", "deep.quarantine", "enable tun", "enable proxy"]:
    assert forbidden not in text, (forbidden, doc)
assert "clean.route.rebuild.safe_dhcp" in text, doc
assert "clean.location.automatic" in text, doc
PY

set +e
"$BIN" rescue --level clean-baseline --json >/tmp/agentlink-clean-baseline-no-yes.out 2>&1
code=$?
set -e
if [ "$code" -eq 0 ]; then
  echo "clean-baseline mutation did not require --yes" >&2
  cat /tmp/agentlink-clean-baseline-no-yes.out >&2
  exit 1
fi
grep -q 'clean-baseline rescue requires --yes' /tmp/agentlink-clean-baseline-no-yes.out

echo "dogfood_clean_baseline_last_resort OK"
