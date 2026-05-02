#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARCH="$(uname -m)"
case "$ARCH" in
  arm64) ASSET_ARCH="macos-arm64"; OUT_ARCH="arm64" ;;
  x86_64) ASSET_ARCH="macos-x64"; OUT_ARCH="amd64" ;;
  *) echo "unsupported macOS arch: $ARCH" >&2; exit 1 ;;
esac

OUT="$ROOT/assets/runtimes/llama.cpp/$OUT_ARCH"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

URL="${LLAMA_CPP_ZIP_URL:-}"
if [ -z "$URL" ]; then
  RELEASE_JSON="$(/usr/bin/curl -fL -s https://api.github.com/repos/ggml-org/llama.cpp/releases/latest || true)"
  URL="$(printf '%s\n' "$RELEASE_JSON" | grep -Eo 'https://[^" ]+' | grep "$ASSET_ARCH" | grep -E '\.(tar\.gz|zip)$' | grep -v kleidiai | head -1 || true)"
fi

if [ -z "$URL" ]; then
  echo "could not auto-detect llama.cpp runtime asset" >&2
  echo "Set LLAMA_CPP_ZIP_URL to a ggml-org/llama.cpp macOS $ARCH release asset." >&2
  exit 1
fi

mkdir -p "$OUT"
ARCHIVE="$TMPDIR/llama-runtime"
echo "downloading llama.cpp runtime: $URL"
/usr/bin/curl -fL "$URL" -o "$ARCHIVE"

rm -rf "$OUT.tmp"
mkdir -p "$OUT.tmp"
case "$URL" in
  *.zip) /usr/bin/unzip -q "$ARCHIVE" -d "$OUT.tmp" ;;
  *) /usr/bin/tar -xzf "$ARCHIVE" -C "$OUT.tmp" ;;
esac

CLI="$(find "$OUT.tmp" -type f -name llama-cli -perm -111 | head -1 || true)"
if [ -z "$CLI" ]; then
  CLI="$(find "$OUT.tmp" -type f -name llama-cli | head -1 || true)"
fi
if [ -z "$CLI" ]; then
  echo "llama-cli not found in downloaded runtime" >&2
  exit 1
fi

rm -rf "$OUT"
mkdir -p "$OUT"
cp -R "$(dirname "$CLI")"/. "$OUT/"
xattr -cr "$OUT" 2>/dev/null || true
chmod +x "$OUT/llama-cli" "$OUT/llama-completion" "$OUT/llama-server" 2>/dev/null || true
RUNTIME_SHA="$(/usr/bin/shasum -a 256 "$OUT/llama-cli" | awk '{print $1}')"
rm -rf "$OUT.tmp"

echo "runtime path: $OUT/llama-cli"
echo "runtime sha256: $RUNTIME_SHA"
echo "$OUT/llama-cli"
