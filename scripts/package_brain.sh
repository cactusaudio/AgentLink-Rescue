#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
. "$ROOT/scripts/lib/asset_cache.sh"

if ! command -v go >/dev/null 2>&1 && [ -x /tmp/agentlink-go-current/go/bin/go ]; then
  export PATH="/tmp/agentlink-go-current/go/bin:$PATH"
fi

HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
  arm64) ARCH="arm64" ;;
  x86_64) ARCH="amd64" ;;
  *) echo "unsupported macOS arch: $HOST_ARCH" >&2; exit 1 ;;
esac
MODEL="$(resolve_model_path 2>/dev/null || true)"
RUNTIME_DIR="$(resolve_llama_runtime_dir "$ARCH" 2>/dev/null || true)"
RUNTIME="$RUNTIME_DIR/llama-cli"

if [ -z "$MODEL" ] || [ -z "$RUNTIME_DIR" ] || [ ! -f "$MODEL" ] || [ ! -x "$RUNTIME" ]; then
  if [ "${DOWNLOAD:-0}" = "1" ]; then
    scripts/fetch_brain_assets.sh
    MODEL="$(resolve_model_path 2>/dev/null || true)"
    RUNTIME_DIR="$(resolve_llama_runtime_dir "$ARCH" 2>/dev/null || true)"
    RUNTIME="$RUNTIME_DIR/llama-cli"
  else
    echo "brain assets missing; set AGENTLINK_ASSET_CACHE, run DOWNLOAD=1 scripts/package_brain.sh, or run scripts/fetch_brain_assets.sh" >&2
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
scripts/build.sh

OUT="$ROOT/dist/Cactus-AgentLink-Rescue"
ZIP="$ROOT/dist/Cactus-AgentLink-Rescue-v0.4.5-brain-gemma4-e4b-q4km.zip"
mkdir -p "$ROOT/dist"
rm -rf "$OUT"
rm -f "$ZIP"
mkdir -p "$OUT/bin" "$OUT/rules" "$OUT/recipes" "$OUT/docs/offline" "$OUT/assets/models" "$OUT/assets/runtimes/llama.cpp/$ARCH" "$OUT/assets/manifests" "$OUT/assets/licenses" "$OUT/assets/installers" "$OUT/scripts"

cp bin/agentlink "$OUT/bin/agentlink"
cp README.md LICENSE "$OUT/"
cp rules/*.json "$OUT/rules/"
cp recipes/*.json "$OUT/recipes/"
cp docs/offline/*.md "$OUT/docs/offline/"
cp packaging/agentlink.command "$OUT/agentlink.command"
cp packaging/rescue.sh "$OUT/rescue.sh"
cp packaging/README_IF_OFFLINE.txt "$OUT/README_IF_OFFLINE.txt"
cp assets/manifests/*.json "$OUT/assets/manifests/"
cat > "$OUT/assets/manifests/manifest.lock.json" <<JSON
{
  "schemaVersion": 3,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "defaultBrainProfile": "gemma-4-e4b-it-q4km",
  "model": {
    "profile": "gemma-4-e4b-it-q4km",
    "path": "assets/models/$(/usr/bin/basename "$MODEL")",
    "filename": "$(/usr/bin/basename "$MODEL")",
    "sha256": "$MODEL_SHA",
    "sizeBytes": $MODEL_SIZE
  },
  "runtime": {
    "backend": "llama.cpp",
    "path": "assets/runtimes/llama.cpp/$ARCH/llama-cli",
    "binary": "llama-cli",
    "sha256": "$RUNTIME_SHA",
    "arch": "$ARCH"
  }
}
JSON
copy_model_to_staging "$OUT/assets/models"
copy_runtime_to_staging "$OUT/assets/runtimes/llama.cpp/$ARCH" "$ARCH"
cp -R assets/licenses/* "$OUT/assets/licenses/"
cp -R assets/installers/* "$OUT/assets/installers/"
if [ -d "$ASSET_CACHE/installers" ]; then
  rsync -a --exclude='*.dmg' "$ASSET_CACHE/installers/" "$OUT/assets/installers/"
fi
cp assets/README.md "$OUT/assets/README.md"
find "$OUT/assets/installers" -name '*.dmg' -delete
cp scripts/fetch_gemma_model.sh scripts/fetch_llamacpp_runtime.sh scripts/fetch_brain_assets.sh scripts/fetch_clash_verge_rev.sh "$OUT/scripts/"

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
