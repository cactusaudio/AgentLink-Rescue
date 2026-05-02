#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.3-brain-gemma4-e4b-q4km.zip"

if [ ! -f "$ZIP" ]; then
  "$ROOT/scripts/package_brain.sh"
fi

TMP="$(mktemp -d)"
TMPHOME="$TMP/home"
mkdir -p "$TMPHOME"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
    rm -rf "$TMP"
  else
    echo "kept temp brain chat dir: $TMP"
  fi
}
trap cleanup EXIT

/usr/bin/unzip -q "$ZIP" -d "$TMP/pkg"
PKG="$TMP/pkg/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
xattr -cr "$PKG" 2>/dev/null || true
chmod +x "$BIN" "$PKG"/assets/runtimes/llama.cpp/*/llama-* 2>/dev/null || true

FAKE='sk-test-THIS_SHOULD_NOT_LEAK-brain-chat'
DEEPSEEK_API_KEY="$FAKE" HOME="$TMPHOME" "$BIN" brain chat --prompt "Say JSON is not required here. Explain what AgentLink does in one sentence." --json > "$TMP/chat.json"
/usr/bin/python3 - "$TMP/chat.json" <<'PY'
import json, sys
data=json.load(open(sys.argv[1]))
assert data["ok"] is True, data
resp=data.get("response","")
assert resp.strip(), data
bad=["I executed", "I ran", "I changed your system", "sudo "]
assert not any(b in resp for b in bad), resp
PY
if grep -R 'THIS_SHOULD_NOT_LEAK' "$TMP" 2>/dev/null; then
  echo "fake secret leaked in brain chat dogfood" >&2
  exit 1
fi

echo "dogfood brain chat OK: $TMP"
