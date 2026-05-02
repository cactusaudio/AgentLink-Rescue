#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go not found; set PATH or install Go for release checks" >&2
  exit 1
fi

GOFILES="$(find . -name '*.go' -not -path './dist/*')"
UNFORMATTED="$(gofmt -l $GOFILES)"
if [ -n "$UNFORMATTED" ]; then
  echo "gofmt check failed:" >&2
  echo "$UNFORMATTED" >&2
  exit 1
fi

go test ./...
go vet ./...
scripts/build.sh
scripts/build_gui.sh
scripts/package.sh

PKG="$ROOT/dist/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
CORE_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.4-core.zip"
BRAIN_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.4-brain-gemma4-e4b-q4km.zip"
CORE_GUI_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.4-core-gui.zip"
BRAIN_GUI_ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.4-brain-gui-gemma4-e4b-q4km.zip"

"$BIN" version | grep '0.4.4'
"$BIN" selftest
"$BIN" doctor --json > /tmp/agentlink-release-doctor.json
/usr/bin/python3 -m json.tool /tmp/agentlink-release-doctor.json >/dev/null
"$BIN" readiness doctor --json > /tmp/agentlink-release-readiness.json
/usr/bin/python3 -m json.tool /tmp/agentlink-release-readiness.json >/dev/null
"$BIN" dev doctor --json > /tmp/agentlink-release-dev.json
/usr/bin/python3 -m json.tool /tmp/agentlink-release-dev.json >/dev/null
"$BIN" support bundle --json > /tmp/agentlink-release-support-bundle.json
/usr/bin/python3 -m json.tool /tmp/agentlink-release-support-bundle.json >/dev/null
"$BIN" brain doctor --json > /tmp/agentlink-release-brain-doctor.json
/usr/bin/python3 -m json.tool /tmp/agentlink-release-brain-doctor.json >/dev/null
"$BIN" recipe list
"$BIN" recipe inspect codex-deepseek-provider-config
"$BIN" recipe run codex-deepseek-provider-config --dry-run

DEEPSEEK_API_KEY='sk-test-THIS_SHOULD_NOT_LEAK-release-check' "$BIN" keys doctor > /tmp/agentlink-release-keys.txt
if grep -R 'THIS_SHOULD_NOT_LEAK' /tmp/agentlink-release-keys.txt "$HOME/Library/Application Support/Cactus AgentLink Rescue" 2>/dev/null; then
  echo "release key redaction smoke failed" >&2
  exit 1
fi

scripts/dogfood_temp_home.sh
scripts/dogfood_proxy_config.sh
scripts/dogfood_runtime_core.sh
scripts/dogfood_guided_rescue.sh
scripts/dogfood_readiness.sh
scripts/dogfood_installer_center.sh
scripts/dogfood_gui_core.sh
scripts/dogfood_gui_screenshots.sh

SOURCE_MODEL="$ROOT/assets/models/gemma-4-E4B-it-Q4_K_M.gguf"
case "$(uname -m)" in
  arm64) SOURCE_ARCH="arm64" ;;
  x86_64) SOURCE_ARCH="amd64" ;;
  *) echo "unsupported macOS arch: $(uname -m)" >&2; exit 1 ;;
esac
SOURCE_LLAMA="$ROOT/assets/runtimes/llama.cpp/$SOURCE_ARCH/llama-cli"
if [ -f "$SOURCE_MODEL" ] && [ -x "$SOURCE_LLAMA" ]; then
  scripts/dogfood_runtime_assets.sh
  scripts/package_brain.sh
  scripts/dogfood_runtime_brain.sh
  scripts/dogfood_brain_chat.sh
  scripts/dogfood_gui_brain.sh
  scripts/dogfood_gui_screenshots.sh
else
  echo "brain assets missing; run scripts/fetch_brain_assets.sh or scripts/package_brain.sh DOWNLOAD=1"
fi

if find "$PKG" \( -name '.DS_Store' -o -name '._*' -o -name '__MACOSX' \) -print | grep .; then
  echo "package folder contains Finder metadata" >&2
  exit 1
fi
if unzip -l "$CORE_ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "release zip contains Finder metadata" >&2
  exit 1
fi
if [ -f "$BRAIN_ZIP" ] && unzip -l "$BRAIN_ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "brain release zip contains Finder metadata" >&2
  exit 1
fi
if unzip -l "$CORE_GUI_ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "core GUI zip contains Finder metadata" >&2
  exit 1
fi
if [ -f "$BRAIN_GUI_ZIP" ] && unzip -l "$BRAIN_GUI_ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "brain GUI zip contains Finder metadata" >&2
  exit 1
fi

if ! git diff --quiet -- assets/manifests/manifest.lock.json; then
  echo "manifest.lock.json changed during package/dogfood; package scripts must not churn source lockfile" >&2
  git diff -- assets/manifests/manifest.lock.json >&2
  exit 1
fi

file "$BIN" | grep 'Mach-O universal binary'

FORBIDDEN_FIELD="api_""key_env"
if find README.md docs packaging recipes internal scripts testdata "$PKG" -type f ! -name agentlink -print0 | xargs -0 grep -n "$FORBIDDEN_FIELD"; then
  echo "forbidden legacy Codex TOML key field found" >&2
  exit 1
fi
LEGACY_MODEL_PATTERN="Q""wen\\|q""wen\\|Q""WEN"
if grep -R "$LEGACY_MODEL_PATTERN" -n README.md docs packaging recipes internal scripts assets/manifests assets/README.md --exclude-dir=assets/models --exclude-dir=assets/runtimes --exclude='*.zip'; then
  echo "active legacy model reference found" >&2
  exit 1
fi
FORBIDDEN_GEMINI_PACKAGE="npm install -g gem""ini"
if grep -R "$FORBIDDEN_GEMINI_PACKAGE" -n README.md docs packaging recipes internal scripts assets/installers --exclude='*.zip'; then
  echo "unofficial Gemini package name found" >&2
  exit 1
fi

echo "release check OK"
