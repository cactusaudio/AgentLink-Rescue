#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-$ROOT/gpt-pro-results/v052-gui-field-beta-$(date -u +%Y%m%dT%H%M%SZ)}"
FULL="${FULL:-1}"
cd "$ROOT"
mkdir -p "$OUT"

GO_BIN="${GO_BIN:-${GO:-$(command -v go || true)}}"
if [ -z "$GO_BIN" ]; then
  echo "go is required" >&2
  exit 2
fi
if [ -d vendor ]; then
  export GOFLAGS="${GOFLAGS:-} -mod=vendor"
fi
PYTHON="${PYTHON:-$(command -v python3 || true)}"
if [ -z "$PYTHON" ]; then
  echo "python3 is required" >&2
  exit 2
fi

host_os="$("$GO_BIN" env GOOS)"
host_arch="$("$GO_BIN" env GOARCH)"
BUILD_DIR="$ROOT/.gpt-pro-build"
BIN_AGENTLINK="${AGENTLINK_BIN:-$BUILD_DIR/agentlink-${host_os}-${host_arch}}"
BIN_RESCUE="${AGENTLINK_RESCUE_BIN:-$BUILD_DIR/agentlink-rescue-${host_os}-${host_arch}}"
mkdir -p "$BUILD_DIR"

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

if [ -f CHECKSUMS.txt ]; then
  run_log 00-checksums shasum -a 256 -c CHECKSUMS.txt
fi

run_log 01-go-version "$GO_BIN" version
run_log 02-build-agentlink "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$BIN_AGENTLINK" ./cmd/agentlink
run_log 03-build-agentlink-rescue "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$BIN_RESCUE" ./cmd/agentlink-rescue
run_log 04-version "$BIN_AGENTLINK" version
run_json 05-doctor "$BIN_AGENTLINK" doctor --json
run_json 06-guided-dry-run "$BIN_AGENTLINK" guided rescue --target auto --dry-run --json
run_json 07-support-bundle "$BIN_AGENTLINK" support bundle --json

"$PYTHON" - "$ROOT" "$OUT" <<'PY'
import json, pathlib, re, sys
root = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])

def read(rel):
    return (root / rel).read_text(errors="ignore")

status = read("gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Views/StatusDashboardView.swift")
app = read("gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/AppState.swift")
client = read("gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/AgentlinkClient.swift")
runner = read("gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/CommandRunner.swift")
dogfood = read("scripts/dogfood_gui_core.sh")
report = read("governor/reports/V0521_GUI_FIELD_BETA_READINESS.md")

checks = {
    "dashboard_has_main_action": "Check & Plan Rescue" in status,
    "dashboard_has_beta_check": "Run Field Beta Check" in status,
    "dashboard_has_support_bundle": "Export Support Bundle" in status,
    "dashboard_says_no_sudo": "does not run sudo in the GUI" in status,
    "appstate_has_beta_check": "func runBetaReadinessCheck()" in app,
    "appstate_guided_dry_run": '"guided", "rescue"' in app and '"--dry-run"' in app,
    "appstate_no_guided_yes_in_beta": not re.search(r"runBetaReadinessCheck[\\s\\S]*guided[^\\n]+--yes", app),
    "appstate_runs_support_bundle": '"support", "bundle"' in app,
    "appstate_serializes_commands": "if isRunning { return }" in app and "defer { isRunning = false }" in app,
    "appstate_support_path_filename_only": "supportBundleDisplayPath" in app and ".lastPathComponent" in app,
    "client_emits_beta_readiness": "betaReadiness" in client and "mainActionDryRunOnly" in client and "noSudoInGUI" in client,
    "runner_redacts_home_paths": r'"/Users/[^/"\\s]+"' in runner or "/Users/" in runner and "<home>" in runner,
    "dogfood_asserts_beta": "betaReadiness" in dogfood and "noSudoInGUI" in dogfood and "guidedDryRunReady" in dogfood,
    "report_documents_beta": "controlled field beta" in report and "not public release" in report,
}
failed = {k: v for k, v in checks.items() if not v}
if failed:
    raise SystemExit(f"static GUI beta contract failed: {failed}")

guided = json.loads((out / "06-guided-dry-run.json").read_text())
support = json.loads((out / "07-support-bundle.json").read_text())
assert guided.get("mode") == "dry-run", guided
assert guided.get("status") in {"healthy", "planned", "dry_run_complete", "no_safe_action", "manual_action_required", "ticket_created", "restart_required"}, guided
assert support.get("status") != "failed", support

screenshot = root / "docs/gui-dogfood/v0.5.2/after/beta-readiness-dashboard.jpg"
assert screenshot.exists() and screenshot.stat().st_size > 0, screenshot
blob = screenshot.read_bytes()
assert b"/Users/jack" not in blob and b"/Users/bowei" not in blob, "screenshot contains literal home path bytes"

(out / "static-gui-contract.json").write_text(json.dumps({"ok": True, "checks": checks}, indent=2, sort_keys=True) + "\n")
PY

run_log 20-targeted-go-tests "$GO_BIN" test ./internal/genome ./internal/cli ./internal/guided ./internal/supportbundle -count=1
run_log 21-go-vet "$GO_BIN" vet ./...

gui_runtime_status="skipped_non_darwin"
if [ "$host_os" = "darwin" ]; then
  run_log 30-swift-build swift build -c release --package-path gui/CactusAgentLinkRescue
  run_log 31-gui-core-dogfood bash scripts/dogfood_gui_core.sh
  gui_runtime_status="passed"
fi

v0501_status="not_run"
if [ "$FULL" = "1" ]; then
  V0501_OUT="$OUT/v0501-regression"
  FULL=1 OUT="$V0501_OUT" bash scripts/gpt_pro_v0501_audit.sh >"$OUT/40-v0501-audit.log" 2>&1
  v0501_status="passed"
  run_log 41-full-go-test "$GO_BIN" test ./... -count=1 -timeout 35m
else
  run_log 40-fast-go-test "$GO_BIN" test ./internal/genome ./internal/cli ./internal/guided ./internal/supportbundle ./scripts -count=1
fi

"$PYTHON" - "$OUT" "$FULL" "$gui_runtime_status" "$v0501_status" <<'PY'
import json, pathlib, sys, time
out = pathlib.Path(sys.argv[1])
full = sys.argv[2] == "1"
summary = {
    "schemaVersion": 1,
    "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "status": "completed",
    "full": full,
    "guiRuntimeStatus": sys.argv[3],
    "v0501RegressionStatus": sys.argv[4],
    "note": "v0.5.2 GUI field beta audit. Linux cloud cannot prove macOS SwiftUI runtime; Darwin runs dogfood_gui_core.",
    "artifacts": sorted(p.name for p in out.iterdir() if p.is_file()),
}
(out / "SUMMARY.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n")
print(json.dumps(summary, indent=2, sort_keys=True))
PY

echo "v0.5.2 GUI beta audit artifacts: $OUT"
