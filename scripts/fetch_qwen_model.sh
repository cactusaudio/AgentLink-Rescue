#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODEL_DIR="$ROOT/assets/models"
MODEL_FILE="Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf"
URL="https://huggingface.co/bartowski/Qwen_Qwen3-4B-Instruct-2507-GGUF/resolve/main/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf?download=true"
SHA256_EXPECTED="2fde00ce69dd4899c70d020845e2638353015bba0fdf161b3eb965f2bca4464e"

mkdir -p "$MODEL_DIR"
TARGET="$MODEL_DIR/$MODEL_FILE"
TMP="$TARGET.part"

if [ -f "$TARGET" ] && [ "${FORCE:-0}" != "1" ]; then
  if [ "${KEEP_EXISTING:-0}" = "1" ]; then
    echo "$TARGET"
    exit 0
  fi
  ACTUAL="$(/usr/bin/shasum -a 256 "$TARGET" | awk '{print $1}')"
  if [ "$ACTUAL" = "$SHA256_EXPECTED" ]; then
    echo "$TARGET"
    exit 0
  fi
  echo "existing model checksum mismatch; set FORCE=1 to replace" >&2
  exit 1
fi

echo "downloading Qwen GGUF to $TMP"
/usr/bin/curl -fL --continue-at - "$URL" -o "$TMP"

ACTUAL="$(/usr/bin/shasum -a 256 "$TMP" | awk '{print $1}')"
if [ "$ACTUAL" != "$SHA256_EXPECTED" ]; then
  echo "checksum mismatch for $TMP" >&2
  echo "expected: $SHA256_EXPECTED" >&2
  echo "actual:   $ACTUAL" >&2
  exit 1
fi

mv "$TMP" "$TARGET"
echo "$TARGET"
