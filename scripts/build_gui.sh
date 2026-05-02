#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUI_ROOT="$ROOT/gui/CactusAgentLinkRescue"
OUT="$ROOT/gui/build"

mkdir -p "$OUT"
swift build -c release --package-path "$GUI_ROOT"
cp "$GUI_ROOT/.build/release/CactusAgentLinkRescue" "$OUT/CactusAgentLinkRescue"
chmod +x "$OUT/CactusAgentLinkRescue"

echo "built $OUT/CactusAgentLinkRescue"
