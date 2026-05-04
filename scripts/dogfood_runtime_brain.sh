#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.0-brain-gemma4-e4b-q4km.zip"

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
    echo "kept temp brain runtime dir: $TMP"
  fi
}
trap cleanup EXIT

REAL_ZSH="$HOME/.zshrc"
REAL_ZSH_BEFORE=""
if [ -f "$REAL_ZSH" ]; then
  REAL_ZSH_BEFORE="$(/usr/bin/shasum -a 256 "$REAL_ZSH" | awk '{print $1}')"
fi

/usr/bin/unzip -q "$ZIP" -d "$TMP"
PKG="$TMP/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
xattr -cr "$PKG" 2>/dev/null || true
chmod +x "$BIN" "$PKG/rescue.sh" "$PKG/agentlink.command" "$PKG"/assets/runtimes/llama.cpp/*/llama-* 2>/dev/null || true

"$BIN" version | grep '0.5.0'
HOME="$TMPHOME" "$BIN" brain doctor --json > "$TMP/brain-doctor.json"
/usr/bin/python3 - "$TMP/brain-doctor.json" <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
assert doc["brainPackAvailable"] is True, doc
assert doc["modelExists"] is True, doc
assert doc["modelSha256OK"] is True, doc
assert doc["runtimeExists"] is True, doc
assert doc["runtimeExecutable"] is True, doc
assert doc["packageLocalAssets"] is True, doc
assert "/Cactus-AgentLink-Rescue/assets/models/" in doc["modelPath"], doc
assert "/Cactus-AgentLink-Rescue/assets/runtimes/" in doc["runtimePath"], doc
assert doc.get("modelFamily") == "gemma", doc
assert doc.get("modelID") == "gemma-4-e4b-it-q4km", doc
PY

MODEL="$PKG/assets/models/gemma-4-E4B-it-Q4_K_M.gguf"
test -f "$MODEL"
ACTUAL_MODEL_SHA="$(/usr/bin/shasum -a 256 "$MODEL" | awk '{print $1}')"

HOME="$TMPHOME" "$BIN" brain selftest --json > "$TMP/brain-selftest.json"
/usr/bin/python3 - "$TMP/brain-selftest.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
assert res["decision"]["schemaVersion"] == 1, res
PY

HOME="$TMPHOME" "$BIN" brain plan --target path --json > "$TMP/brain-plan-path.json"
/usr/bin/python3 - "$TMP/brain-plan-path.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
decision=res["decision"]
assert decision["schemaVersion"] == 1, res
assert decision["selectedRecipe"]["id"] == "macos-zsh-path-repair", res
assert decision["confidence"] >= 0.55, res
PY

DEEPSEEK_API_KEY='sk-test-THIS_SHOULD_NOT_LEAK-runtime-brain' HOME="$TMPHOME" "$BIN" repair --auto --brain --target path --dry-run > "$TMP/brain-dry-run.txt"
if [ -e "$TMPHOME/.zshrc" ]; then
  echo "brain dry-run mutated temp HOME" >&2
  exit 1
fi

DEEPSEEK_API_KEY='sk-test-THIS_SHOULD_NOT_LEAK-runtime-brain' HOME="$TMPHOME" "$BIN" repair --auto --brain --target path --yes > "$TMP/brain-yes.txt"
if [ "$(grep -c '>>> AGENTLINK_PATH_BLOCK >>>' "$TMPHOME/.zshrc")" != "1" ]; then
  echo "managed PATH block count is not exactly 1" >&2
  exit 1
fi

HOME="$TMPHOME" "$BIN" restore last > "$TMP/restore-last.txt"
if [ -e "$TMPHOME/.zshrc" ]; then
  echo "restore last did not restore absent temp .zshrc" >&2
  exit 1
fi

HOME="$TMPHOME" "$BIN" report --for-codex --latest > "$TMP/codex-report.txt" || true
if grep -R 'THIS_SHOULD_NOT_LEAK' "$TMP" 2>/dev/null; then
  echo "fake secret leaked in brain runtime dogfood" >&2
  exit 1
fi

if [ -n "$REAL_ZSH_BEFORE" ]; then
  REAL_ZSH_AFTER="$(/usr/bin/shasum -a 256 "$REAL_ZSH" | awk '{print $1}')"
  if [ "$REAL_ZSH_BEFORE" != "$REAL_ZSH_AFTER" ]; then
    echo "real HOME .zshrc changed during brain runtime dogfood" >&2
    exit 1
  fi
fi

LEGACY_MODEL_PATTERN="q""wen|Q""wen"
if unzip -l "$ZIP" | grep -Ei "$LEGACY_MODEL_PATTERN"; then
  echo "brain runtime zip contains inactive model references" >&2
  exit 1
fi

echo "dogfood runtime brain OK: $TMP"
