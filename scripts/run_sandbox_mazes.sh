#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FIXTURE_ROOT="${FIXTURE_ROOT:-$ROOT/testdata/chaos/network/mazes}"
BIN="${AGENTLINK_BIN:-$ROOT/bin/agentlink}"
REPORT_PATH="${REPORT_PATH:-/tmp/agentlink-sandbox-mazes-report.json}"
SKIP_BUILD="${SKIP_BUILD:-0}"
PYTHON="${PYTHON:-$(command -v python3 || true)}"

if [ -z "$PYTHON" ]; then
  echo "python3 is required for sandbox maze JSON validation" >&2
  exit 2
fi

if [ "$SKIP_BUILD" != "1" ] || [ ! -x "$BIN" ]; then
  if ! command -v go >/dev/null 2>&1 && [ -x "/Users/jack/CactusLocalAgent/.asset-cache/Cactus-Local-Agent-Pro/runtimes/go/bin/go" ]; then
    export PATH="/Users/jack/CactusLocalAgent/.asset-cache/Cactus-Local-Agent-Pro/runtimes/go/bin:$PATH"
  fi
  (cd "$ROOT" && bash scripts/build.sh >/dev/null)
fi

if [ ! -d "$FIXTURE_ROOT" ]; then
  echo "sandbox maze fixture root not found: $FIXTURE_ROOT" >&2
  exit 2
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

count=0
passed=0
failed=0
printf '[' >"$tmp/report.json"
first=1

while IFS= read -r fixture; do
  [ -n "$fixture" ] || continue
  count=$((count + 1))
  out="$tmp/$(basename "$fixture").out.json"
  "$BIN" chaos run --fixture "$fixture" --json >"$out"
  "$PYTHON" -m json.tool "$out" >/dev/null
  status="$("$PYTHON" - "$out" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    print(json.load(f).get("status", ""))
PY
)"
  if [ "$status" = "passed" ]; then
    passed=$((passed + 1))
  else
    failed=$((failed + 1))
  fi
  if [ "$first" -eq 0 ]; then
    printf ',' >>"$tmp/report.json"
  fi
  first=0
  cat "$out" >>"$tmp/report.json"
done < <(find "$FIXTURE_ROOT" -name '*.json' | sort)

printf ']' >>"$tmp/report.json"

"$PYTHON" - "$tmp/report.json" "$REPORT_PATH" "$count" "$passed" "$failed" <<'PY'
import json, sys
items = json.load(open(sys.argv[1]))
summary = {
    "schemaVersion": 1,
    "pack": "agentlink-common-network-sandbox-mazes",
    "fixtureRoot": "testdata/chaos/network/mazes",
    "count": int(sys.argv[3]),
    "passed": int(sys.argv[4]),
    "failed": int(sys.argv[5]),
    "results": items,
}
with open(sys.argv[2], "w") as f:
    json.dump(summary, f, indent=2, sort_keys=True)
print(json.dumps({k: summary[k] for k in ["pack", "count", "passed", "failed"]}, sort_keys=True))
PY

if [ "$failed" -ne 0 ]; then
  echo "sandbox maze failures recorded at $REPORT_PATH" >&2
  exit 1
fi

echo "sandbox maze report: $REPORT_PATH"
