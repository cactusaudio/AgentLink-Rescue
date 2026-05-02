#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.3-brain-gui-gemma4-e4b-q4km.zip"

scripts/package_gui_brain.sh

TMP="$(mktemp -d)"
TMPHOME="$TMP/home"
mkdir -p "$TMPHOME"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
    rm -rf "$TMP"
  else
    echo "kept temp GUI brain dir: $TMP"
  fi
}
trap cleanup EXIT

REAL_ZSH="$HOME/.zshrc"
REAL_ZSH_BEFORE=""
if [ -f "$REAL_ZSH" ]; then
  REAL_ZSH_BEFORE="$(/usr/bin/shasum -a 256 "$REAL_ZSH" | awk '{print $1}')"
fi

/usr/bin/unzip -q "$ZIP" -d "$TMP"
APP="$TMP/Cactus AgentLink Rescue.app"
PKG="$APP/Contents/Resources/agentlink"
BIN="$PKG/bin/agentlink"
GUIBIN="$APP/Contents/MacOS/CactusAgentLinkRescue"
MODEL="$PKG/assets/models/gemma-4-E4B-it-Q4_K_M.gguf"

xattr -cr "$APP" 2>/dev/null || true
chmod +x "$GUIBIN" "$BIN" "$PKG"/assets/runtimes/llama.cpp/*/llama-* 2>/dev/null || true

test -x "$GUIBIN"
test -x "$BIN"
test -f "$MODEL"
test -f "$PKG/assets/manifests/manifest.lock.json"
find "$PKG/assets/runtimes" -type f -name 'llama-cli' -print | grep .

ACTUAL_MODEL_SHA="$(/usr/bin/shasum -a 256 "$MODEL" | awk '{print $1}')"

HOME="$TMPHOME" "$BIN" version | grep '0.4.3'
HOME="$TMPHOME" "$BIN" brain doctor --json > "$TMP/brain-doctor.json"
/usr/bin/python3 - "$TMP/brain-doctor.json" <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
assert doc["brainPackAvailable"] is True, doc
assert doc["modelSha256OK"] is True, doc
assert doc["runtimeExecutable"] is True, doc
assert doc["packageLocalAssets"] is True, doc
assert doc.get("modelFamily") == "gemma", doc
assert doc.get("modelID") == "gemma-4-e4b-it-q4km", doc
PY

HOME="$TMPHOME" "$BIN" brain selftest --json > "$TMP/brain-selftest.json"
HOME="$TMPHOME" "$BIN" brain plan --target path --json > "$TMP/brain-plan.json"
HOME="$TMPHOME" "$BIN" repair --auto --brain --target path --dry-run --json > "$TMP/brain-dryrun.json"
DEEPSEEK_API_KEY='sk-test-THIS_SHOULD_NOT_LEAK-gui-brain' HOME="$TMPHOME" "$BIN" repair --auto --brain --target path --yes --json > "$TMP/brain-yes.json"

if [ "$(grep -c '>>> AGENTLINK_PATH_BLOCK >>>' "$TMPHOME/.zshrc")" != "1" ]; then
  echo "GUI brain temp HOME repair did not create exactly one managed block" >&2
  exit 1
fi
HOME="$TMPHOME" "$BIN" restore last --json > "$TMP/restore.json"
if [ -e "$TMPHOME/.zshrc" ]; then
  echo "GUI brain restore did not restore absent temp .zshrc" >&2
  exit 1
fi

HOME="$TMPHOME" "$GUIBIN" --selftest-gui > "$TMP/gui-selftest.json"
/usr/bin/python3 - "$TMP/gui-selftest.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
assert res["brainDoctor"]["brainPackAvailable"] is True, res
assert res["brainDoctor"].get("modelFamily") == "gemma", res
PY

if grep -R 'THIS_SHOULD_NOT_LEAK' "$TMP" 2>/dev/null; then
  echo "fake secret leaked in GUI brain dogfood" >&2
  exit 1
fi
if [ -n "$REAL_ZSH_BEFORE" ]; then
  REAL_ZSH_AFTER="$(/usr/bin/shasum -a 256 "$REAL_ZSH" | awk '{print $1}')"
  if [ "$REAL_ZSH_BEFORE" != "$REAL_ZSH_AFTER" ]; then
    echo "real HOME .zshrc changed during GUI brain dogfood" >&2
    exit 1
  fi
fi
if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "brain GUI zip contains Finder metadata" >&2
  exit 1
fi
LEGACY_MODEL_PATTERN="q""wen|Q""wen"
if unzip -l "$ZIP" | grep -Ei "$LEGACY_MODEL_PATTERN"; then
  echo "brain GUI zip contains inactive model references" >&2
  exit 1
fi

echo "dogfood GUI brain OK: $TMP"
