#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

MODEL="assets/models/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf"
ARCH="$(uname -m)"
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

scripts/build.sh

OUT="$ROOT/dist/Cactus-AgentLink-Rescue"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.3.0-brain-qwen3-4b-q4km.zip"
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
[ ! -f assets/manifest.lock.json ] || cp assets/manifest.lock.json "$OUT/assets/manifest.lock.json"
cp "$MODEL" "$OUT/assets/models/"
cp -R "$RUNTIME_DIR"/. "$OUT/assets/runtimes/llama.cpp/$ARCH/"
cp -R assets/licenses/* "$OUT/assets/licenses/"
cp assets/README.md "$OUT/assets/README.md"
cp scripts/fetch_qwen_model.sh scripts/fetch_llamacpp_runtime.sh scripts/fetch_brain_assets.sh "$OUT/scripts/"

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
