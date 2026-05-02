#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.3-core.zip"

if [ ! -f "$ZIP" ]; then
  "$ROOT/scripts/package.sh"
fi

TMP="$(mktemp -d)"
TMPHOME="$TMP/home"
mkdir -p "$TMPHOME"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
    rm -rf "$TMP"
  else
    echo "kept temp guided rescue dir: $TMP"
  fi
}
trap cleanup EXIT

REAL_ZSH="$HOME/.zshrc"
REAL_ZSH_BEFORE=""
if [ -f "$REAL_ZSH" ]; then
  REAL_ZSH_BEFORE="$(/usr/bin/shasum -a 256 "$REAL_ZSH" | awk '{print $1}')"
fi

/usr/bin/unzip -q "$ZIP" -d "$TMP/pkg"
PKG="$TMP/pkg/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
xattr -cr "$PKG" 2>/dev/null || true
chmod +x "$BIN" "$PKG/rescue.sh" "$PKG/agentlink.command" 2>/dev/null || true

HOME="$TMPHOME" "$BIN" guided rescue --target path --dry-run --json > "$TMP/guided-path-dry.json"
/usr/bin/python3 - "$TMP/guided-path-dry.json" <<'PY'
import json, sys
rep=json.load(open(sys.argv[1]))
assert rep["status"] == "dry_run_complete", rep
assert rep["selectedRecipe"] == "macos-zsh-path-repair", rep
assert rep["mode"] == "dry-run", rep
assert rep.get("rollbackAvailable") is False, rep
PY
if [ -e "$TMPHOME/.zshrc" ]; then
  echo "guided dry-run mutated temp HOME" >&2
  exit 1
fi

DEEPSEEK_API_KEY='sk-test-THIS_SHOULD_NOT_LEAK-guided' HOME="$TMPHOME" "$BIN" guided rescue --target path --yes --json > "$TMP/guided-path-yes.json"
/usr/bin/python3 - "$TMP/guided-path-yes.json" <<'PY'
import json, sys
rep=json.load(open(sys.argv[1]))
assert rep["status"] in ("repaired", "rolled_back"), rep
assert rep["selectedRecipe"] == "macos-zsh-path-repair", rep
assert rep["mode"] == "execute", rep
assert rep.get("snapshotID"), rep
assert rep.get("rollbackAvailable") is True, rep
PY
if [ "$(grep -c '>>> AGENTLINK_PATH_BLOCK >>>' "$TMPHOME/.zshrc")" != "1" ]; then
  echo "guided managed block count is not exactly 1" >&2
  exit 1
fi

HOME="$TMPHOME" "$BIN" restore last --json > "$TMP/guided-restore.json"
if [ -e "$TMPHOME/.zshrc" ]; then
  echo "guided restore did not restore absent temp .zshrc" >&2
  exit 1
fi

HOME="$TMPHOME" "$BIN" guided rescue --target auto --dry-run --json > "$TMP/guided-auto-dry.json"
/usr/bin/python3 - "$TMP/guided-auto-dry.json" <<'PY'
import json, sys
rep=json.load(open(sys.argv[1]))
assert rep["status"] in ("healthy", "dry_run_complete", "no_safe_action", "manual_action_required"), rep
assert rep["mode"] == "dry-run", rep
PY

if grep -R 'THIS_SHOULD_NOT_LEAK' "$TMP" 2>/dev/null; then
  echo "fake secret leaked in guided rescue dogfood" >&2
  exit 1
fi

if [ -n "$REAL_ZSH_BEFORE" ]; then
  REAL_ZSH_AFTER="$(/usr/bin/shasum -a 256 "$REAL_ZSH" | awk '{print $1}')"
  if [ "$REAL_ZSH_BEFORE" != "$REAL_ZSH_AFTER" ]; then
    echo "real HOME .zshrc changed during guided rescue dogfood" >&2
    exit 1
  fi
fi

echo "dogfood guided rescue OK: $TMP"
