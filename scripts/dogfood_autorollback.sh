#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
go test ./internal/repair -run 'TestCriticalWorsened' -count=1
echo "dogfood_autorollback OK"
