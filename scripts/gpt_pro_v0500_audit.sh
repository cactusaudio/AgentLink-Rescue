#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-$ROOT/gpt-pro-results/v0500-audit-$(date -u +%Y%m%dT%H%M%SZ)}"
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
WORK="$OUT/work"
SNAPSHOT="$WORK/snapshot"
REPORT_MD="$WORK/report.md"
REPORT_JSON="$WORK/report.json"
INDEX_DIR="$WORK/index"
FAKE_HOME="$WORK/fake-home"

mkdir -p "$BUILD_DIR" "$WORK" "$FAKE_HOME/.docker"
cat >"$FAKE_HOME/.docker/config.json" <<'JSON'
{
  "auths": {
    "registry.example.com": {
      "auth": "dXNlcjpzdXBlcnNlY3JldA==",
      "identitytoken": "dockertoken-SECRET-1234567890"
    }
  },
  "credsStore": "osxkeychain",
  "credHelpers": {
    "registry.example.com": "desktop"
  },
  "proxies": {
    "default": {
      "httpProxy": "http://127.0.0.1:7890"
    }
  }
}
JSON

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
run_log 02-build-agentlink env CGO_ENABLED=0 "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$BIN_AGENTLINK" ./cmd/agentlink
run_log 03-build-agentlink-rescue env CGO_ENABLED=0 "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$BIN_RESCUE" ./cmd/agentlink-rescue
run_log 04-version "$BIN_RESCUE" version

export AGENTLINK_GENOME_INDEX_DIR="$INDEX_DIR"

run_json 10-index-rebuild "$BIN_RESCUE" index rebuild --json
run_json 11-index-stats "$BIN_RESCUE" index stats --json
run_json 12-explain-mac-proxy "$BIN_RESCUE" explain --card MAC-PROXY-002 --json
run_json 13-diagnose-browser-codex "$BIN_RESCUE" diagnose --symptom "browser works but codex fails" --no-snapshot --json
run_json 14-diagnose-clash-tun "$BIN_RESCUE" diagnose --symptom "clash tun broke dns" --no-snapshot --json
run_json 15-diagnose-vpn-lan "$BIN_RESCUE" diagnose --symptom "vpn loses lan" --no-snapshot --json
run_json 16-diagnose-dante "$BIN_RESCUE" diagnose --symptom "dante devices invisible" --no-snapshot --json
run_json 20-collect-strict env HOME="$FAKE_HOME" "$BIN_RESCUE" collect --out "$SNAPSHOT" --privacy strict --json
run_json 21-features "$BIN_RESCUE" features --snapshot "$SNAPSHOT" --json
run_json 22-diagnose-snapshot "$BIN_RESCUE" diagnose --symptom "browser works but codex fails" --snapshot "$SNAPSHOT" --json
run_json 30-gate-dante "$BIN_RESCUE" gate --card DANTE-CLOCK-001 --json
run_json 31-gate-proxy "$BIN_RESCUE" gate --card MAC-PROXY-002 --json
run_json 32-report-markdown "$BIN_RESCUE" report --symptom "browser works but codex fails" --snapshot "$SNAPSHOT" --format markdown --out "$REPORT_MD" --json
run_json 33-report-json "$BIN_RESCUE" report --symptom "browser works but codex fails" --snapshot "$SNAPSHOT" --format json --out "$REPORT_JSON" --json

"$PYTHON" - "$OUT" "$SNAPSHOT" "$REPORT_MD" "$REPORT_JSON" <<'PY'
import json, pathlib, sqlite3, sys
out = pathlib.Path(sys.argv[1])
snapshot = pathlib.Path(sys.argv[2])
report_md = pathlib.Path(sys.argv[3])
report_json = pathlib.Path(sys.argv[4])

def load(name):
    return json.loads((out / f"{name}.json").read_text())

stats = load("11-index-stats")
rebuild = load("10-index-rebuild")
diag = load("22-diagnose-snapshot")
diag_no_snapshot = load("13-diagnose-browser-codex")
gate_dante = load("30-gate-dante")
gate_proxy = load("31-gate-proxy")

assert (stats.get("cards") or stats.get("Cards")) == 300, stats
assert (stats.get("layers") or stats.get("Layers")) == 15, stats
assert (stats.get("ftsRows") or stats.get("FTSRows")) == 300, stats
assert (stats.get("ftsIdsUsable") or stats.get("FTSIDsUsable")) is True, stats
assert (stats.get("ftsNullIds") or stats.get("FTSNullIDs") or 0) == 0, stats
sqlite_path = pathlib.Path(rebuild.get("sqlitePath") or rebuild.get("SQLitePath"))
assert sqlite_path.exists(), sqlite_path
con = sqlite3.connect(sqlite_path)
tables = {r[0] for r in con.execute("select name from sqlite_master where type='table'")}
assert "cards" in tables and "cards_fts" in tables, tables
assert con.execute("select count(*) from cards").fetchone()[0] == 300
assert con.execute("select count(*) from cards_fts").fetchone()[0] == 300
fts_rows = con.execute("select id,title from cards_fts where cards_fts match ? order by bm25(cards_fts) limit 50", ("browser OR codex OR fails",)).fetchall()
assert fts_rows and all(r[0] and r[1] for r in fts_rows), fts_rows
assert any(r[0] in {"CODEX-API-001", "MAC-PROXY-002"} or str(r[0]).startswith("DEV-CODEX-") for r in fts_rows), fts_rows

hypotheses_no_snapshot = diag_no_snapshot.get("hypotheses") or diag_no_snapshot.get("Hypotheses") or []
hypothesis_ids_no_snapshot = [h.get("cardId") or h.get("CardID") for h in hypotheses_no_snapshot[:3]]
assert "MAC-PROXY-002" in hypothesis_ids_no_snapshot, hypothesis_ids_no_snapshot
hypotheses = diag.get("hypotheses") or diag.get("Hypotheses") or []
card_ids = [h.get("cardId") or h.get("CardID") for h in hypotheses[:3]]
assert "CODEX-API-001" in card_ids, card_ids
fts_ids = {r[0] for r in fts_rows}
for h in hypotheses:
    why = " ".join(h.get("whyMatched") or h.get("WhyMatched") or [])
    cid = h.get("cardId") or h.get("CardID")
    if "SQLite/FTS candidate retrieval selected this card" in why:
        assert cid in fts_ids, (cid, fts_ids, why)
assert (gate_dante.get("result") or gate_dante.get("Result")) == "manual_only", gate_dante
assert (gate_proxy.get("result") or gate_proxy.get("Result")) == "human_confirmed_allowed", gate_proxy
assert report_md.exists() and "AgentLink Rescue Report" in report_md.read_text(errors="ignore")
assert report_json.exists()
json.loads(report_json.read_text())

raw_dir = snapshot / "raw"
raw_text = "\n".join(p.read_text(errors="ignore") for p in raw_dir.glob("*") if p.is_file()) if raw_dir.exists() else ""
assert not ("@" in raw_text and ":" in raw_text), "raw appears to contain user:pass-like data"
combined = raw_text
redacted_dir = snapshot / "redacted"
if redacted_dir.exists():
    combined += "\n".join(p.read_text(errors="ignore") for p in redacted_dir.glob("*") if p.is_file())
combined += report_md.read_text(errors="ignore")
combined += report_json.read_text(errors="ignore")
for leak in ["dXNlcjpzdXBlcnNlY3JldA==", "dockertoken-SECRET-1234567890", "osxkeychain", "registry.example.com", "desktop"]:
    assert leak not in combined, f"privacy leak found: {leak}"
features = json.loads((snapshot / "features" / "snapshot_features.json").read_text())
assert features.get("privacyMode") == "strict", features
assert features.get("features", {}).get("docker_proxy_config_present") == "true", features
PY

run_log 40-targeted-go-tests "$GO_BIN" test ./internal/genome ./internal/cli -count=1
run_log 41-go-vet "$GO_BIN" vet ./...

if [ "$FULL" = "1" ]; then
  run_log 42-full-go-test "$GO_BIN" test ./... -count=1 -timeout 35m
fi

"$PYTHON" - "$OUT" "$FULL" "$REPORT_MD" "$REPORT_JSON" <<'PY'
import json, pathlib, time, sys
out = pathlib.Path(sys.argv[1])
full = sys.argv[2] == "1"
summary = {
    "schemaVersion": 1,
    "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "status": "completed",
    "full": full,
    "note": "v0.5 Codex phases audit only; no real macOS mutation, GUI, signing, or model behavior proven.",
    "reports": {
        "markdown": str(pathlib.Path(sys.argv[3])),
        "json": str(pathlib.Path(sys.argv[4])),
    },
    "artifacts": sorted(p.name for p in out.iterdir() if p.is_file()),
}
(out / "SUMMARY.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n")
print(json.dumps(summary, indent=2, sort_keys=True))
PY

echo "v0.5 audit artifacts: $OUT"
