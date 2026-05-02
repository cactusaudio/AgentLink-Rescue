#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BEFORE="$ROOT/docs/gui-dogfood/v0.4.1/before"
AFTER="$ROOT/docs/gui-dogfood/v0.4.1/after"
CORE_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.1-core-gui.zip"
BRAIN_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.1-brain-gui-gemma4-e4b-q4km.zip"

mkdir -p "$BEFORE" "$AFTER"

for name in dashboard doctor brain plan dry-run rescue rollback reports settings; do
  if [ ! -s "$BEFORE/$name.png" ]; then
    echo "missing before screenshot: $BEFORE/$name.png" >&2
    if [ "${STRICT_UI_SCREENSHOTS:-0}" = "1" ]; then
      exit 1
    fi
  fi
done

if [ -f "$CORE_ZIP" ]; then
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  /usr/bin/unzip -q "$CORE_ZIP" -d "$TMP"
  APP="$TMP/Cactus AgentLink Rescue.app"
  PKG="$APP/Contents/Resources/agentlink"
  xattr -cr "$APP" 2>/dev/null || true
  chmod +x "$APP/Contents/MacOS/CactusAgentLinkRescue" "$PKG/bin/agentlink" 2>/dev/null || true
  AGENTLINK_BRAIN_HOME="$TMP/empty-brain" PATH="/usr/bin:/bin:/usr/sbin:/sbin" "$APP/Contents/MacOS/CactusAgentLinkRescue" --selftest-gui > "$TMP/core-gui-selftest.json"
  /usr/bin/python3 - "$TMP/core-gui-selftest.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
assert res["brainDoctor"]["brainPackAvailable"] is False, res
PY
  rm -rf "$TMP"
  trap - EXIT
else
  echo "core GUI zip missing; run scripts/package_gui_core.sh first" >&2
fi

if [ -f "$BRAIN_ZIP" ]; then
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  /usr/bin/unzip -q "$BRAIN_ZIP" -d "$TMP"
  APP="$TMP/Cactus AgentLink Rescue.app"
  PKG="$APP/Contents/Resources/agentlink"
  xattr -cr "$APP" 2>/dev/null || true
  chmod +x "$APP/Contents/MacOS/CactusAgentLinkRescue" "$PKG/bin/agentlink" "$PKG"/assets/runtimes/llama.cpp/*/llama-* 2>/dev/null || true
  HOME="$TMP/home" "$APP/Contents/MacOS/CactusAgentLinkRescue" --selftest-gui > "$TMP/brain-gui-selftest.json"
  /usr/bin/python3 - "$TMP/brain-gui-selftest.json" <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
assert res["brainDoctor"]["brainPackAvailable"] is True, res
assert res["brainDoctor"]["packageLocalAssets"] is True, res
PY
  rm -rf "$TMP"
  trap - EXIT
else
  echo "brain GUI zip missing; run scripts/package_gui_brain.sh first" >&2
fi

AFTER_COUNT="$(find "$AFTER" -name '*.png' -type f -size +0c | wc -l | tr -d ' ')"
if [ "$AFTER_COUNT" -eq 0 ]; then
  echo "GUI screenshot dogfood: after screenshots not captured by script; use computer-use or screencapture and save them under $AFTER"
  if [ "${STRICT_UI_SCREENSHOTS:-0}" = "1" ]; then
    exit 1
  fi
else
  echo "GUI screenshot dogfood: found $AFTER_COUNT after screenshots"
fi

echo "dogfood GUI screenshots OK"
