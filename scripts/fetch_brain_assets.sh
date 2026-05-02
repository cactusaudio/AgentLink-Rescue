#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODEL_PATH="$("$ROOT/scripts/fetch_gemma_model.sh" | tail -1)"
RUNTIME_PATH="$("$ROOT/scripts/fetch_llamacpp_runtime.sh" | tail -1)"
LOCK="$ROOT/assets/manifests/manifest.lock.json"

MODEL_SHA="$(/usr/bin/shasum -a 256 "$MODEL_PATH" | awk '{print $1}')"
MODEL_SIZE="$(/usr/bin/stat -f %z "$MODEL_PATH")"
RUNTIME_SHA="$(/usr/bin/shasum -a 256 "$RUNTIME_PATH" | awk '{print $1}')"
MODEL_REL="${MODEL_PATH#$ROOT/}"
RUNTIME_REL="${RUNTIME_PATH#$ROOT/}"
MODEL_FILE="$(basename "$MODEL_PATH")"
RUNTIME_BIN="$(basename "$RUNTIME_PATH")"
RUNTIME_ARCH="$(basename "$(dirname "$RUNTIME_PATH")")"

mkdir -p "$(dirname "$LOCK")"
cat > "$LOCK" <<JSON
{
  "schemaVersion": 3,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "defaultBrainProfile": "gemma-4-e4b-it-q4km",
  "model": {
    "profile": "gemma-4-e4b-it-q4km",
    "path": "$MODEL_REL",
    "filename": "$MODEL_FILE",
    "sha256": "$MODEL_SHA",
    "sizeBytes": $MODEL_SIZE
  },
  "runtime": {
    "backend": "llama.cpp",
    "path": "$RUNTIME_REL",
    "binary": "$RUNTIME_BIN",
    "sha256": "$RUNTIME_SHA",
    "arch": "$RUNTIME_ARCH"
  }
}
JSON

echo "manifest lock: $LOCK"
echo "$LOCK"
