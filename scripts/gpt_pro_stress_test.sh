#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-$ROOT/gpt-pro-results/$(date -u +%Y%m%dT%H%M%SZ)}"
FULL="${FULL:-1}"
cd "$ROOT"
mkdir -p "$OUT"

if ! command -v go >/dev/null 2>&1 && [ -x "/Users/jack/CactusLocalAgent/.asset-cache/Cactus-Local-Agent-Pro/runtimes/go/bin/go" ]; then
  export PATH="/Users/jack/CactusLocalAgent/.asset-cache/Cactus-Local-Agent-Pro/runtimes/go/bin:$PATH"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required for the full GPT Pro stress gate" >&2
  exit 2
fi
PYTHON="${PYTHON:-$(command -v python3 || true)}"
if [ -z "$PYTHON" ]; then
  echo "python3 is required for GPT Pro stress JSON validation" >&2
  exit 2
fi

host_os="$(go env GOOS)"
host_arch="$(go env GOARCH)"
BIN="${AGENTLINK_BIN:-$ROOT/.gpt-pro-build/agentlink-${host_os}-${host_arch}}"

mkdir -p "$(dirname "$BIN")"
if [ ! -x "$BIN" ]; then
  echo "building host agentlink: ${host_os}/${host_arch}"
  (cd "$ROOT" && GOOS="$host_os" GOARCH="$host_arch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$BIN" ./cmd/agentlink)
fi
chmod +x "$BIN"

run_log() {
  local name="$1"
  shift
  echo "== $name =="
  {
    echo "$ $*"
    "$@"
  } >"$OUT/$name.log" 2>&1
}

run_json() {
  local name="$1"
  shift
  echo "== $name =="
  "$@" >"$OUT/$name.json" 2>"$OUT/$name.stderr"
  "$PYTHON" -m json.tool "$OUT/$name.json" >/dev/null
}

export AGENTLINK_BIN="$BIN"
export AGENTLINK_RECIPES_DIR="$ROOT/recipes"

run_log 00-version "$BIN" version
run_json 01-chaos-list "$BIN" chaos list --root "$ROOT/testdata/chaos" --json

REPORT_PATH="$OUT/02-sandbox-mazes.json" SKIP_BUILD=1 bash "$ROOT/scripts/run_sandbox_mazes.sh" >"$OUT/02-sandbox-mazes.log" 2>&1
"$PYTHON" -m json.tool "$OUT/02-sandbox-mazes.json" >/dev/null

run_log 03-chaos-fixtures go test ./internal/chaos -run TestChaosFixturesPass -count=1
run_log 04-core-units go test ./internal/packagehealth ./internal/journal ./internal/command ./internal/supportbundle -count=1
run_log 05-v0300-v0320-chaos env AGENTLINK_BIN="$BIN" go test ./internal/chaos -run 'TestV0300|TestV0301|TestV0320' -count=1 -timeout 20m
run_log 06-v0330-rc-chaos env AGENTLINK_BIN="$BIN" go test ./internal/chaos -run 'TestV0330' -count=1 -timeout 30m

if [ "$FULL" = "1" ]; then
  run_log 07-full-go-test go test ./... -count=1 -timeout 30m
  run_log 08-go-vet go vet ./...
fi

"$PYTHON" - "$OUT" <<'PY'
import json, pathlib, sys, time
out = pathlib.Path(sys.argv[1])
summary = {
    "schemaVersion": 1,
    "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "status": "completed",
    "note": "Cloud sandbox gate only. It does not prove real macOS host repair or GUI behavior.",
    "artifacts": sorted(p.name for p in out.iterdir() if p.is_file()),
}
maze = out / "02-sandbox-mazes.json"
if maze.exists():
    data = json.loads(maze.read_text())
    summary["sandboxMazes"] = {
        "count": data.get("count"),
        "passed": data.get("passed"),
        "failed": data.get("failed"),
    }
(out / "SUMMARY.json").write_text(json.dumps(summary, indent=2, sort_keys=True))
print(json.dumps(summary, indent=2, sort_keys=True))
PY

echo "GPT Pro stress artifacts: $OUT"
