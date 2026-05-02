#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

case "$(uname -m)" in
  arm64)
    ARCH="arm64"
    DIR="macos-arm64"
    PATTERN="aarch64.*\\.dmg$"
    ;;
  x86_64)
    ARCH="amd64"
    DIR="macos-amd64"
    PATTERN="x64.*\\.dmg$"
    ;;
  *)
    echo "unsupported macOS arch: $(uname -m)" >&2
    exit 1
    ;;
esac

BASE="$ROOT/assets/installers/clash-verge-rev"
OUTDIR="$BASE/$DIR"
LOCK="$BASE/manifest.lock.json"
mkdir -p "$OUTDIR"

if [ "${FORCE:-0}" != "1" ]; then
  EXISTING="$(find "$OUTDIR" -maxdepth 1 -name '*.dmg' -print -quit)"
  if [ -n "$EXISTING" ] && [ "${KEEP_EXISTING:-1}" = "1" ]; then
    SHA="$(/usr/bin/shasum -a 256 "$EXISTING" | awk '{print $1}')"
    SIZE="$(/usr/bin/stat -f %z "$EXISTING")"
    echo "using existing Clash Verge Rev DMG: $EXISTING"
    echo "sha256: $SHA"
    echo "size: $SIZE"
    exit 0
  fi
fi

META="$(mktemp)"
trap 'rm -f "$META" "$TMPFILE"' EXIT
if ! /usr/bin/curl -fsSL "https://api.github.com/repos/clash-verge-rev/clash-verge-rev/releases/latest" -o "$META"; then
  echo "failed to fetch Clash Verge Rev release metadata" >&2
  echo "Manual source: https://github.com/clash-verge-rev/clash-verge-rev/releases" >&2
  exit 1
fi

read -r VERSION ASSET_NAME URL < <(/usr/bin/python3 - "$META" "$PATTERN" <<'PY'
import json, re, sys
meta=json.load(open(sys.argv[1]))
pat=re.compile(sys.argv[2], re.I)
for a in meta.get("assets", []):
    name=a.get("name","")
    if pat.search(name):
        print(meta.get("tag_name",""), name, a.get("browser_download_url",""))
        break
else:
    sys.exit(2)
PY
)

if [ -z "${URL:-}" ]; then
  echo "could not locate matching Clash Verge Rev DMG for $ARCH" >&2
  echo "Manual source: https://github.com/clash-verge-rev/clash-verge-rev/releases" >&2
  exit 1
fi

DEST="$OUTDIR/$ASSET_NAME"
TMPFILE="$DEST.part"
if [ "${FORCE:-0}" = "1" ]; then
  rm -f "$DEST" "$TMPFILE"
fi

echo "downloading $ASSET_NAME"
/usr/bin/curl -L --continue-at - "$URL" -o "$TMPFILE"

if /usr/bin/file "$TMPFILE" | grep -Ei 'HTML|text' >/dev/null; then
  mv "$TMPFILE" "$TMPFILE.bad"
  echo "download did not produce a DMG; saved bad file: $TMPFILE.bad" >&2
  exit 1
fi

mv "$TMPFILE" "$DEST"
SHA="$(/usr/bin/shasum -a 256 "$DEST" | awk '{print $1}')"
SIZE="$(/usr/bin/stat -f %z "$DEST")"
cat > "$LOCK" <<JSON
{
  "schemaVersion": 1,
  "version": "$VERSION",
  "assetName": "$ASSET_NAME",
  "downloadURL": "$URL",
  "sha256": "$SHA",
  "sizeBytes": $SIZE,
  "fetchedAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "arch": "$ARCH"
}
JSON

echo "Clash Verge Rev DMG: $DEST"
echo "sha256: $SHA"
echo "size: $SIZE"
