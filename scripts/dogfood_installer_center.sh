#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.4-core.zip"

if [ ! -f "$ZIP" ]; then
  "$ROOT/scripts/package.sh"
fi

TMP="$(mktemp -d)"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
    rm -rf "$TMP"
  else
    echo "kept temp installer center dir: $TMP"
  fi
}
trap cleanup EXIT

/usr/bin/unzip -q "$ZIP" -d "$TMP"
PKG="$TMP/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
xattr -cr "$PKG" 2>/dev/null || true
chmod +x "$BIN" 2>/dev/null || true

"$BIN" installer list --json > "$TMP/list.json"
"$BIN" installer doctor --json > "$TMP/doctor.json"
"$BIN" installer dry-run codex-cli --json > "$TMP/codex-dry.json"
"$BIN" installer dry-run claude-code-cli --json > "$TMP/claude-dry.json"
"$BIN" installer dry-run gemini-cli --json > "$TMP/gemini-dry.json"
"$BIN" installer inspect clash-verge-rev --json > "$TMP/clash-inspect.json"
"$BIN" installer dry-run codex-app --json > "$TMP/codex-app-dry.json"

/usr/bin/python3 - "$TMP/list.json" "$TMP/codex-dry.json" "$TMP/claude-dry.json" "$TMP/gemini-dry.json" "$TMP/codex-app-dry.json" <<'PY'
import json, sys
catalog=json.load(open(sys.argv[1]))
ids={item["id"] for item in catalog}
required={"codex-cli","codex-app","claude-code-cli","gemini-cli","clash-verge-rev"}
assert required <= ids, ids
for path, token in [
    (sys.argv[2], "@openai/codex"),
    (sys.argv[3], "@anthropic-ai/claude-code"),
    (sys.argv[4], "@google/gemini-cli"),
    (sys.argv[5], "developers.openai.com/codex/app"),
]:
    data=json.load(open(path))
    text=json.dumps(data)
    assert data["status"] == "dry_run", data
    assert token in text, (token, text)
PY

FORBIDDEN_GEMINI_PACKAGE="npm install -g gem""ini"
if grep -R "$FORBIDDEN_GEMINI_PACKAGE" "$TMP" "$PKG/assets/installers" 2>/dev/null; then
  echo "unofficial Gemini package name found" >&2
  exit 1
fi

echo "dogfood installer center OK: $TMP"
