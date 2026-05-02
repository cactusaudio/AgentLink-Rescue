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
BAD="$TARGET.bad"

if [ -f "$TARGET" ] && [ "${FORCE:-0}" != "1" ]; then
  if [ "${KEEP_EXISTING:-0}" = "1" ]; then
    ACTUAL="$(/usr/bin/shasum -a 256 "$TARGET" | awk '{print $1}')"
    echo "model path: $TARGET"
    echo "model sha256: $ACTUAL"
    echo "$TARGET"
    exit 0
  fi
  ACTUAL="$(/usr/bin/shasum -a 256 "$TARGET" | awk '{print $1}')"
  if [ "$ACTUAL" = "$SHA256_EXPECTED" ]; then
    echo "model path: $TARGET"
    echo "model sha256: $ACTUAL"
    echo "$TARGET"
    exit 0
  fi
  mv "$TARGET" "$BAD"
  echo "existing model checksum mismatch; moved to $BAD" >&2
  exit 1
fi

if [ "${FORCE:-0}" = "1" ]; then
  rm -f "$TARGET" "$TMP"
fi

echo "downloading Qwen GGUF to $TMP"
/usr/bin/curl -fL --continue-at - "$URL" -o "$TMP"

ACTUAL="$(/usr/bin/shasum -a 256 "$TMP" | awk '{print $1}')"
if [ "$ACTUAL" != "$SHA256_EXPECTED" ]; then
  echo "checksum mismatch for $TMP" >&2
  echo "expected: $SHA256_EXPECTED" >&2
  echo "actual:   $ACTUAL" >&2
  mv "$TMP" "$BAD"
  echo "bad download moved to $BAD" >&2
  exit 1
fi

mv "$TMP" "$TARGET"
echo "model path: $TARGET"
echo "model sha256: $ACTUAL"
echo "$TARGET"
