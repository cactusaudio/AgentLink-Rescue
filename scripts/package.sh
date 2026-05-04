#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

"$ROOT/scripts/build.sh"

OUT="$ROOT/dist/Cactus-AgentLink-Rescue"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.5.0-core.zip"
mkdir -p "$ROOT/dist"
rm -rf "$OUT"
rm -f "$ZIP"
find "$ROOT/dist" -maxdepth 1 \( -name .DS_Store -o -name .AppleDouble -o -name AppleDouble \) -exec rm -rf {} +
mkdir -p "$OUT/bin" "$OUT/rules" "$OUT/recipes" "$OUT/docs/offline" "$OUT/assets/manifests" "$OUT/assets/licenses" "$OUT/assets/models" "$OUT/assets/runtimes" "$OUT/assets/installers" "$OUT/scripts" "$OUT/opencode"

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
cp -R assets/installers/* "$OUT/assets/installers/"
cp assets/README.md "$OUT/assets/README.md"
cp -R opencode/agentlink-plugin "$OUT/opencode/agentlink-plugin"
cp assets/models/.gitkeep "$OUT/assets/models/.gitkeep"
cp assets/runtimes/.gitkeep "$OUT/assets/runtimes/.gitkeep"
find "$OUT/assets/installers" -name '*.dmg' -delete
cp scripts/fetch_gemma_model.sh scripts/fetch_llamacpp_runtime.sh scripts/fetch_brain_assets.sh scripts/fetch_clash_verge_rev.sh "$OUT/scripts/"

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
