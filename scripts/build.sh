#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

if ! command -v go >/dev/null 2>&1 && [ -x "$HOME/CactusLocalAgent/.asset-cache/Cactus-Local-Agent-Pro/runtimes/go/bin/go" ]; then
  export PATH="$HOME/CactusLocalAgent/.asset-cache/Cactus-Local-Agent-Pro/runtimes/go/bin:$PATH"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to build agentlink" >&2
  exit 1
fi

mkdir -p bin dist/build

if [ -n "${GOOS:-}" ] || [ -n "${GOARCH:-}" ]; then
  : "${GOOS:=darwin}"
  : "${GOARCH:=$(go env GOARCH)}"
  echo "building bin/agentlink for ${GOOS}/${GOARCH}"
  GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/agentlink ./cmd/agentlink
  exit 0
fi

echo "building darwin/arm64"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/build/agentlink-arm64 ./cmd/agentlink

echo "building darwin/amd64"
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/build/agentlink-amd64 ./cmd/agentlink

if command -v lipo >/dev/null 2>&1; then
  echo "creating universal binary"
  lipo -create -output bin/agentlink dist/build/agentlink-arm64 dist/build/agentlink-amd64
else
  arch="$(uname -m)"
  if [ "$arch" = "arm64" ]; then
    cp dist/build/agentlink-arm64 bin/agentlink
  else
    cp dist/build/agentlink-amd64 bin/agentlink
  fi
fi

chmod +x bin/agentlink
echo "built $ROOT/bin/agentlink"
