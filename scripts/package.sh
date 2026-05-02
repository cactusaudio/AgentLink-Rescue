#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x bin/agentlink ]; then
  "$ROOT/scripts/build.sh"
fi

OUT="$ROOT/dist/Cactus-AgentLink-Rescue"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.3.1-core.zip"
mkdir -p "$ROOT/dist"
rm -rf "$OUT"
rm -f "$ZIP"
find "$ROOT/dist" -maxdepth 1 \( -name .DS_Store -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +
mkdir -p "$OUT/bin" "$OUT/rules" "$OUT/recipes" "$OUT/docs/offline" "$OUT/assets/manifests" "$OUT/assets/licenses" "$OUT/assets/models" "$OUT/assets/runtimes" "$OUT/scripts"

cp bin/agentlink "$OUT/bin/agentlink"
cp README.md LICENSE "$OUT/"
cp rules/*.json "$OUT/rules/"
cp recipes/*.json "$OUT/recipes/"
cp docs/offline/*.md "$OUT/docs/offline/"
cp packaging/agentlink.command "$OUT/agentlink.command"
cp packaging/rescue.sh "$OUT/rescue.sh"
cp packaging/README_IF_OFFLINE.txt "$OUT/README_IF_OFFLINE.txt"
cp assets/manifests/*.json "$OUT/assets/manifests/"
cp -R assets/licenses/* "$OUT/assets/licenses/"
cp assets/README.md "$OUT/assets/README.md"
cp assets/models/.gitkeep "$OUT/assets/models/.gitkeep"
cp assets/runtimes/.gitkeep "$OUT/assets/runtimes/.gitkeep"
cp scripts/fetch_qwen_model.sh scripts/fetch_llamacpp_runtime.sh scripts/fetch_brain_assets.sh "$OUT/scripts/"

chmod +x "$OUT/bin/agentlink" "$OUT/agentlink.command" "$OUT/rescue.sh" "$OUT/scripts/"*.sh

find "$OUT" -depth \( -name .DS_Store -o -name __MACOSX -o -name '._*' -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +
if find "$OUT/assets/models" -name '*.gguf' -print | grep .; then
  echo "core package must not contain GGUF model files" >&2
  exit 1
fi
if find "$OUT/assets/runtimes" -type f -name 'llama-cli' -print | grep .; then
  echo "core package must not contain llama.cpp runtime binaries" >&2
  exit 1
fi

(
  cd "$ROOT/dist"
  COPYFILE_DISABLE=1 zip -r -X "$(basename "$ZIP")" "Cactus-AgentLink-Rescue" >/dev/null
)

if unzip -l "$ZIP" | grep -E '__MACOSX|\.DS_Store|AppleDouble|/\._' >/dev/null; then
  echo "release zip contains Finder metadata" >&2
  exit 1
fi

echo "package created: $OUT"
echo "zip created: $ZIP"
