#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go not found; set PATH or install Go for chaos checks" >&2
  exit 1
fi

go test ./internal/chaos ./internal/packagehealth ./internal/journal ./internal/command

DIST_BIN="$ROOT/dist/Cactus-AgentLink-Rescue/bin/agentlink"
LOCAL_BIN="$ROOT/bin/agentlink"
if [ -x "$DIST_BIN" ]; then
  BIN="$DIST_BIN"
elif [ -x "$LOCAL_BIN" ]; then
  BIN="$LOCAL_BIN"
else
  scripts/build.sh
  BIN="$LOCAL_BIN"
fi

if [ ! -x "$BIN" ]; then
  echo "agentlink binary not found after build: $BIN" >&2
  exit 1
fi

"$BIN" chaos list --json >/tmp/agentlink-chaos-list.json
/usr/bin/python3 -m json.tool /tmp/agentlink-chaos-list.json >/dev/null

while IFS= read -r fixture; do
  [ -n "$fixture" ] || continue
  "$BIN" chaos run --fixture "$fixture" --json >/tmp/agentlink-chaos-fixture.json
  /usr/bin/python3 -m json.tool /tmp/agentlink-chaos-fixture.json >/dev/null
  grep '"status": "passed"' /tmp/agentlink-chaos-fixture.json >/dev/null
done < <(find testdata/chaos -name '*.json' | sort)

echo "chaos release check OK"
