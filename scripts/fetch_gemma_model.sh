#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODEL_DIR="$ROOT/assets/models"
MODEL_FILE="gemma-4-E4B-it-Q4_K_M.gguf"
URL="https://huggingface.co/unsloth/gemma-4-E4B-it-GGUF/resolve/main/gemma-4-E4B-it-Q4_K_M.gguf?download=true"
TARGET="$MODEL_DIR/$MODEL_FILE"
TMP="$TARGET.part"
BAD="$TARGET.bad"
LOCK="$ROOT/assets/manifests/manifest.lock.json"
MIN_FREE_KB=$((8 * 1024 * 1024))

mkdir -p "$MODEL_DIR" "$ROOT/assets/manifests"

FREE_KB="$(/bin/df -k "$MODEL_DIR" | awk 'NR==2 {print $4}')"
if [ "${FREE_KB:-0}" -lt "$MIN_FREE_KB" ]; then
  echo "not enough free disk space for Gemma model; need at least 8GB free" >&2
  exit 1
fi

is_bad_download() {
  local file="$1"
  [ -s "$file" ] || return 0
  if /usr/bin/head -c 512 "$file" | LC_ALL=C grep -Eiq '<html|<!doctype|AccessDenied|Invalid username|Repository Not Found'; then
    return 0
  fi
  return 1
}

write_lock() {
  local model_sha="$1"
  local model_size="$2"
  local runtime_path=""
  local runtime_sha=""
  local arch=""
  case "$(uname -m)" in
    arm64) arch="arm64" ;;
    x86_64) arch="amd64" ;;
    *) arch="$(uname -m)" ;;
  esac
  runtime_path="assets/runtimes/llama.cpp/$arch/llama-cli"
  if [ -f "$ROOT/$runtime_path" ]; then
    runtime_sha="$(/usr/bin/shasum -a 256 "$ROOT/$runtime_path" | awk '{print $1}')"
  fi
  cat > "$LOCK" <<JSON
{
  "schemaVersion": 3,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "defaultBrainProfile": "gemma-4-e4b-it-q4km",
  "model": {
    "profile": "gemma-4-e4b-it-q4km",
    "path": "assets/models/$MODEL_FILE",
    "filename": "$MODEL_FILE",
    "sha256": "$model_sha",
    "sizeBytes": $model_size
  },
  "runtime": {
    "backend": "llama.cpp",
    "path": "$runtime_path",
    "binary": "llama-cli",
    "sha256": "$runtime_sha",
    "arch": "$arch"
  }
}
JSON
}

if [ -f "$TARGET" ] && [ "${FORCE:-0}" != "1" ]; then
  if is_bad_download "$TARGET"; then
    mv "$TARGET" "$BAD"
    echo "existing model looked like an HTML/error response; moved to $BAD" >&2
    exit 1
  fi
  SHA="$(/usr/bin/shasum -a 256 "$TARGET" | awk '{print $1}')"
  SIZE="$(/usr/bin/stat -f %z "$TARGET")"
  write_lock "$SHA" "$SIZE"
  echo "model path: $TARGET"
  echo "model size: $SIZE"
  echo "model sha256: $SHA"
  echo "$TARGET"
  exit 0
fi

if [ "${FORCE:-0}" = "1" ]; then
  rm -f "$TARGET" "$TMP"
fi

echo "downloading Gemma 4 E4B GGUF to $TMP"
/usr/bin/curl -fL --continue-at - "$URL" -o "$TMP"

if is_bad_download "$TMP"; then
  mv "$TMP" "$BAD"
  echo "download looked like an HTML/error response; moved to $BAD" >&2
  exit 1
fi

SHA="$(/usr/bin/shasum -a 256 "$TMP" | awk '{print $1}')"
SIZE="$(/usr/bin/stat -f %z "$TMP")"
mv "$TMP" "$TARGET"
write_lock "$SHA" "$SIZE"

echo "model path: $TARGET"
echo "model size: $SIZE"
echo "model sha256: $SHA"
echo "$TARGET"
