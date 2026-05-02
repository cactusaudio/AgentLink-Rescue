#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODEL_PATH="$("$ROOT/scripts/fetch_qwen_model.sh" | tail -1)"
RUNTIME_PATH="$("$ROOT/scripts/fetch_llamacpp_runtime.sh" | tail -1)"
LOCK="$ROOT/assets/manifest.lock.json"

MODEL_SHA="$(/usr/bin/shasum -a 256 "$MODEL_PATH" | awk '{print $1}')"
MODEL_SIZE="$(/usr/bin/stat -f %z "$MODEL_PATH")"
RUNTIME_SHA="$(/usr/bin/shasum -a 256 "$RUNTIME_PATH" | awk '{print $1}')"
MODEL_REL="${MODEL_PATH#$ROOT/}"
RUNTIME_REL="${RUNTIME_PATH#$ROOT/}"

cat > "$LOCK" <<JSON
{
  "schemaVersion": 1,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "modelPath": "$MODEL_REL",
  "modelSHA256": "$MODEL_SHA",
  "modelSizeBytes": $MODEL_SIZE,
  "runtimePath": "$RUNTIME_REL",
  "runtimeSHA256": "$RUNTIME_SHA"
}
JSON

echo "$LOCK"
