#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

MODEL="assets/models/gemma-4-E4B-it-Q4_K_M.gguf"
HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
  arm64) ARCH="arm64" ;;
  x86_64) ARCH="amd64" ;;
  *) echo "unsupported macOS arch: $HOST_ARCH" >&2; exit 1 ;;
esac
RUNTIME_DIR="assets/runtimes/llama.cpp/$ARCH"
RUNTIME="$RUNTIME_DIR/llama-cli"

if [ ! -f "$MODEL" ] || [ ! -x "$RUNTIME" ]; then
  if [ "${DOWNLOAD:-0}" = "1" ]; then
    scripts/fetch_brain_assets.sh
  else
    echo "brain assets missing; run DOWNLOAD=1 scripts/package_brain.sh or scripts/fetch_brain_assets.sh" >&2
    exit 1
  fi
fi

MODEL_SHA="$(/usr/bin/shasum -a 256 "$MODEL" | awk '{print $1}')"
if [ ! -x "$RUNTIME" ]; then
  echo "llama-cli runtime missing or not executable: $RUNTIME" >&2
  exit 1
fi
MODEL_SIZE="$(/usr/bin/stat -f %z "$MODEL")"
RUNTIME_SHA="$(/usr/bin/shasum -a 256 "$RUNTIME" | awk '{print $1}')"
LOCK_SHA="$(/usr/bin/python3 - "$ROOT/assets/manifests/manifest.lock.json" <<'PY' 2>/dev/null || true
import json, sys
try:
    print(json.load(open(sys.argv[1])).get("model", {}).get("sha256", ""))
except Exception:
    print("")
PY
)"
if [ -n "$LOCK_SHA" ] && [ "$LOCK_SHA" != "$MODEL_SHA" ]; then
  echo "brain model checksum mismatch against manifest.lock.json" >&2
  echo "expected: $LOCK_SHA" >&2
  echo "actual:   $MODEL_SHA" >&2
  exit 1
fi
mkdir -p assets/manifests
cat > assets/manifests/manifest.lock.json <<JSON
{
  "schemaVersion": 3,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "defaultBrainProfile": "gemma-4-e4b-it-q4km",
  "model": {
    "profile": "gemma-4-e4b-it-q4km",
    "path": "$MODEL",
    "filename": "$(basename "$MODEL")",
    "sha256": "$MODEL_SHA",
    "sizeBytes": $MODEL_SIZE
  },
  "runtime": {
    "backend": "llama.cpp",
    "path": "$RUNTIME",
    "binary": "llama-cli",
    "sha256": "$RUNTIME_SHA",
    "arch": "$ARCH"
  }
}
JSON

scripts/build.sh

OUT="$ROOT/dist/Cactus-AgentLink-Rescue"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.1-brain-gemma4-e4b-q4km.zip"
mkdir -p "$ROOT/dist"
rm -rf "$OUT"
rm -f "$ZIP"
mkdir -p "$OUT/bin" "$OUT/rules" "$OUT/recipes" "$OUT/docs/offline" "$OUT/assets/models" "$OUT/assets/runtimes/llama.cpp/$ARCH" "$OUT/assets/manifests" "$OUT/assets/licenses" "$OUT/scripts"

cp bin/agentlink "$OUT/bin/agentlink"
cp README.md LICENSE "$OUT/"
cp rules/*.json "$OUT/rules/"
cp recipes/*.json "$OUT/recipes/"
cp docs/offline/*.md "$OUT/docs/offline/"
cp packaging/agentlink.command "$OUT/agentlink.command"
cp packaging/rescue.sh "$OUT/rescue.sh"
cp packaging/README_IF_OFFLINE.txt "$OUT/README_IF_OFFLINE.txt"
cp assets/manifests/*.json "$OUT/assets/manifests/"
cp "$MODEL" "$OUT/assets/models/"
cp "$RUNTIME_DIR/llama-cli" "$OUT/assets/runtimes/llama.cpp/$ARCH/"
if [ -f "$RUNTIME_DIR/llama-completion" ]; then
  cp "$RUNTIME_DIR/llama-completion" "$OUT/assets/runtimes/llama.cpp/$ARCH/"
fi
cp "$RUNTIME_DIR"/*.dylib "$OUT/assets/runtimes/llama.cpp/$ARCH/" 2>/dev/null || true
if [ -f "$RUNTIME_DIR/LICENSE" ]; then
  cp "$RUNTIME_DIR/LICENSE" "$OUT/assets/runtimes/llama.cpp/$ARCH/"
fi
cp -R assets/licenses/* "$OUT/assets/licenses/"
cp assets/README.md "$OUT/assets/README.md"
cp scripts/fetch_gemma_model.sh scripts/fetch_llamacpp_runtime.sh scripts/fetch_brain_assets.sh "$OUT/scripts/"

chmod +x "$OUT/bin/agentlink" "$OUT/agentlink.command" "$OUT/rescue.sh" "$OUT/scripts/"*.sh "$OUT/assets/runtimes/llama.cpp/$ARCH/llama-cli" "$OUT/assets/runtimes/llama.cpp/$ARCH/llama-completion" 2>/dev/null || true
find "$OUT" -depth \( -name .DS_Store -o -name __MACOSX -o -name '._*' -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +

(
  cd "$ROOT/dist"
  COPYFILE_DISABLE=1 zip -r -X "$(basename "$ZIP")" "Cactus-AgentLink-Rescue" >/dev/null
)

if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
  echo "brain release zip contains Finder metadata" >&2
  exit 1
fi

echo "brain package created: $OUT"
echo "brain zip created: $ZIP"
/usr/bin/shasum -a 256 "$ZIP"
