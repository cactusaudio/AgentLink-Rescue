#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.3.1-core.zip"

if [ ! -f "$ZIP" ]; then
  "$ROOT/scripts/package.sh"
fi

TMP="$(mktemp -d)"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
    rm -rf "$TMP"
  else
    echo "kept temp core runtime dir: $TMP"
  fi
}
trap cleanup EXIT

/usr/bin/unzip -q "$ZIP" -d "$TMP"
PKG="$TMP/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
EMPTY_BRAIN_HOME="$TMP/empty-brain-home"
mkdir -p "$EMPTY_BRAIN_HOME"

xattr -cr "$PKG" 2>/dev/null || true
chmod +x "$BIN" "$PKG/rescue.sh" "$PKG/agentlink.command" 2>/dev/null || true

if find "$PKG/assets/models" -name '*.gguf' -print | grep .; then
  echo "core package contains GGUF model" >&2
  exit 1
fi
if find "$PKG/assets/runtimes" -type f -name 'llama-cli' -print | grep .; then
  echo "core package contains llama-cli runtime" >&2
  exit 1
fi

"$BIN" version | grep '0.3.1'
"$BIN" selftest
"$BIN" doctor --json > "$TMP/core-doctor.json"
/usr/bin/python3 -m json.tool "$TMP/core-doctor.json" >/dev/null

AGENTLINK_BRAIN_HOME="$EMPTY_BRAIN_HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" "$BIN" brain doctor --json > "$TMP/core-brain-doctor.json"
/usr/bin/python3 - "$TMP/core-brain-doctor.json" <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
assert doc["brainPackAvailable"] is False, doc
assert doc["modelExists"] is False, doc
assert doc["runtimeExists"] is False, doc
assert "qwen model" in doc.get("missingAssets", []), doc
assert "llama-cli runtime" in doc.get("missingAssets", []), doc
assert doc.get("fetchCommands"), doc
PY

if AGENTLINK_BRAIN_HOME="$EMPTY_BRAIN_HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" "$BIN" brain selftest > "$TMP/selftest.out" 2> "$TMP/selftest.err"; then
  echo "core brain selftest unexpectedly succeeded" >&2
  exit 1
fi
grep -E 'brain assets missing|brain selftest failed' "$TMP/selftest.err" "$TMP/selftest.out" >/dev/null

"$BIN" rescue --level safe --dry-run > "$TMP/rescue-safe-dry-run.txt"
"$BIN" recipe list > "$TMP/recipe-list.txt"

echo "dogfood runtime core OK: $TMP"
