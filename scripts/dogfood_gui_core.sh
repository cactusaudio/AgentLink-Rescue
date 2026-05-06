#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.1-core-gui.zip"

scripts/package_gui_core.sh

TMP="$(mktemp -d)"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
    rm -rf "$TMP"
  else
    echo "kept temp GUI core dir: $TMP"
  fi
}
trap cleanup EXIT

/usr/bin/unzip -q "$ZIP" -d "$TMP"
APP="$TMP/Cactus AgentLink Rescue.app"
PKG="$APP/Contents/Resources/agentlink"
BIN="$PKG/bin/agentlink"
GUIBIN="$APP/Contents/MacOS/CactusAgentLinkRescue"
EMPTY_BRAIN_HOME="$TMP/empty-brain"
mkdir -p "$EMPTY_BRAIN_HOME"

xattr -cr "$APP" 2>/dev/null || true
chmod +x "$GUIBIN" "$BIN" 2>/dev/null || true

test -x "$GUIBIN"
test -x "$BIN"
test -f "$PKG/agentlink.command"
test -f "$PKG/rescue.sh"

if find "$PKG/assets/models" -name '*.gguf' -print | grep .; then
  echo "core GUI contains GGUF" >&2
  exit 1
fi
if find "$PKG/assets/runtimes" -type f -name 'llama-cli' -print | grep .; then
  echo "core GUI contains llama-cli" >&2
  exit 1
fi

"$BIN" version | grep '0.5.1'
"$BIN" doctor --json > "$TMP/doctor.json"
/usr/bin/python3 -m json.tool "$TMP/doctor.json" >/dev/null
AGENTLINK_BRAIN_HOME="$EMPTY_BRAIN_HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" "$BIN" brain doctor --json > "$TMP/brain-doctor.json"
/usr/bin/python3 - "$TMP/brain-doctor.json" <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
assert doc["brainPackAvailable"] is False, doc
assert doc["modelExists"] is False, doc
assert doc["runtimeExists"] is False, doc
assert doc.get("fetchCommands"), doc
PY

AGENTLINK_BRAIN_HOME="$EMPTY_BRAIN_HOME" PATH="/usr/bin:/bin:/usr/sbin:/sbin" "$GUIBIN" --selftest-gui > "$TMP/gui-selftest.json"
/usr/bin/python3 - "$TMP/gui-selftest.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
assert res["version"].strip() == "agentlink 0.5.1", res
assert res["brainDoctor"]["brainPackAvailable"] is False, res
PY

"$GUIBIN" --selftest-gui-long-output > "$TMP/gui-long-output-selftest.json"
/usr/bin/python3 - "$TMP/gui-long-output-selftest.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
assert res["longOutputTruncated"] is True, res
assert res["redactionClean"] is True, res
assert res["timeoutTimedOut"] is True, res
PY

if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "core GUI zip contains Finder metadata" >&2
  exit 1
fi

echo "dogfood GUI core OK: $TMP"
