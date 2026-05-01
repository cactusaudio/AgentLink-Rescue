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
scripts/package.sh

PKG="$ROOT/dist/Cactus-AgentLink-Rescue"
BIN="$PKG/bin/agentlink"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.2.2.zip"

"$BIN" version | grep '0.2.2'
"$BIN" selftest
"$BIN" doctor --json > /tmp/agentlink-release-doctor.json
/usr/bin/python3 -m json.tool /tmp/agentlink-release-doctor.json >/dev/null
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

if find "$PKG" \( -name '.DS_Store' -o -name '._*' -o -name '__MACOSX' \) -print | grep .; then
  echo "package folder contains Finder metadata" >&2
  exit 1
fi
if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|/\._'; then
  echo "release zip contains Finder metadata" >&2
  exit 1
fi

file "$BIN" | grep 'Mach-O universal binary'

FORBIDDEN_FIELD="api_""key_env"
if find README.md docs packaging recipes internal scripts testdata "$PKG" -type f ! -name agentlink -print0 | xargs -0 grep -n "$FORBIDDEN_FIELD"; then
  echo "forbidden legacy Codex TOML key field found" >&2
  exit 1
fi

echo "release check OK"
