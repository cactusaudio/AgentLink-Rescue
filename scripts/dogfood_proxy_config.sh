#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${AGENTLINK_BIN:-$ROOT/dist/Cactus-AgentLink-Rescue/bin/agentlink}"

if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
  "$ROOT/scripts/package.sh"
fi

GIT_BIN="$(command -v git || true)"
NPM_BIN="$(command -v npm || true)"
if [ -z "$GIT_BIN" ]; then
  echo "SKIP: git not available for isolated proxy dogfood"
  exit 0
fi
if [ -z "$NPM_BIN" ]; then
  echo "SKIP: npm not available for isolated proxy dogfood"
  exit 0
fi

TMPHOME="$(mktemp -d)"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" = "1" ]; then
    echo "KEEP_TMPHOME=1; temp home kept: $TMPHOME"
  else
    rm -rf "$TMPHOME"
  fi
}
trap cleanup EXIT

export HOME="$TMPHOME"
export GIT_CONFIG_GLOBAL="$TMPHOME/.gitconfig"
export NPM_CONFIG_USERCONFIG="$TMPHOME/.npmrc"

OLD_PROXY="http://user:pass@127.0.0.1:7897"

npmrc_value() {
  key="$1"
  [ -f "$NPM_CONFIG_USERCONFIG" ] || return 1
  awk -F= -v key="$key" '$1 == key {print substr($0, index($0, "=") + 1); found=1} END {exit found ? 0 : 1}' "$NPM_CONFIG_USERCONFIG"
}

"$GIT_BIN" config --global http.proxy "$OLD_PROXY"
"$GIT_BIN" config --global --unset https.proxy >/dev/null 2>&1 || true
"$NPM_BIN" config set proxy "$OLD_PROXY" >/dev/null
"$NPM_BIN" config delete https-proxy >/dev/null 2>&1 || true

"$BIN" recipe run npm-git-proxy-conflict-repair --yes --param mode=clean > "$TMPHOME/proxy-clean.txt"

if "$GIT_BIN" config --global --get http.proxy >/dev/null 2>&1; then
  echo "git http.proxy was not unset during clean" >&2
  exit 1
fi
if "$GIT_BIN" config --global --get https.proxy >/dev/null 2>&1; then
  echo "git https.proxy should remain unset during clean" >&2
  exit 1
fi
npm_proxy_after="$("$NPM_BIN" config get proxy 2>/dev/null || true)"
npm_https_after="$("$NPM_BIN" config get https-proxy 2>/dev/null || true)"
case "$npm_proxy_after" in ""|"null"|"undefined") ;; *) echo "npm proxy was not unset: $npm_proxy_after" >&2; exit 1 ;; esac
case "$npm_https_after" in ""|"null"|"undefined") ;; *) echo "npm https-proxy should remain unset: $npm_https_after" >&2; exit 1 ;; esac

"$BIN" report --for-codex --latest > "$TMPHOME/proxy-agent-dispatch.json"
if grep -F 'user:pass' "$TMPHOME/proxy-agent-dispatch.json" "$TMPHOME/Library/Application Support/Cactus AgentLink Rescue"/sessions/*/* 2>/dev/null; then
  echo "proxy credentials leaked into reports" >&2
  exit 1
fi

"$BIN" restore last > "$TMPHOME/proxy-restore.txt"

restored_git_proxy="$("$GIT_BIN" config --global --get http.proxy)"
if [ "$restored_git_proxy" != "$OLD_PROXY" ]; then
  echo "git http.proxy was not restored" >&2
  exit 1
fi
if "$GIT_BIN" config --global --get https.proxy >/dev/null 2>&1; then
  echo "git https.proxy missing key was not preserved as unset" >&2
  exit 1
fi
restored_npm_proxy="$(npmrc_value proxy || true)"
if [ "$restored_npm_proxy" != "$OLD_PROXY" ]; then
  echo "npm proxy was not restored" >&2
  exit 1
fi
restored_npm_https="$(npmrc_value https-proxy || true)"
case "$restored_npm_https" in ""|"null"|"undefined") ;; *) echo "npm https-proxy missing key was not preserved as unset: $restored_npm_https" >&2; exit 1 ;; esac

echo "dogfood proxy config OK: $TMPHOME"
