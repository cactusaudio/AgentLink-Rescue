#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STAMP="${STAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_ROOT="${OUT_ROOT:-$HOME/CactusLocalAgent/.asset-cache/AgentLink-Rescue/release-artifacts/gpt-pro-v0501-reaudit-$STAMP}"
PKG_ROOT="$OUT_ROOT/AgentLink-Rescue"
ZIP_PATH="$OUT_ROOT/agentlink-gpt-pro-v0501-codex-phases-reaudit-$STAMP.zip"
INPUT_DOCS="${INPUT_DOCS:-/Users/jack/Downloads/agentlink_rescue_v0_5_codex_docs}"
SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || true)"
SOURCE_DIRTY="false"
if [ -n "$(git -C "$ROOT" status --short 2>/dev/null || true)" ]; then
  SOURCE_DIRTY="true"
fi

for tool in python3 rsync zip shasum unzip; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "$tool is required to package GPT Pro v0.5 audit artifacts" >&2
    exit 2
  fi
done

rm -rf "$OUT_ROOT"
mkdir -p "$PKG_ROOT"

rsync -a "$ROOT/" "$PKG_ROOT/" \
  --exclude '.git/' \
  --exclude '.DS_Store' \
  --exclude '.gpt-pro-build/' \
  --exclude 'gpt-pro-results/' \
  --exclude 'dist/' \
  --exclude 'bin/' \
  --exclude '.asset-cache/' \
  --exclude '.build/' \
  --exclude 'DerivedData/' \
  --exclude 'node_modules/' \
  --exclude '__pycache__/' \
  --exclude '*.gguf' \
  --exclude '*.safetensors' \
  --exclude '*.dmg' \
  --exclude '*.zip' \
  --exclude '*.tar.gz' \
  --exclude '*.png' \
  --exclude '*.jpg' \
  --exclude '*.jpeg'

mkdir -p "$PKG_ROOT/audit-inputs/v0500-codex-docs"
if [ -d "$INPUT_DOCS" ]; then
  rsync -a "$INPUT_DOCS/" "$PKG_ROOT/audit-inputs/v0500-codex-docs/"
else
  echo "warning: input docs not found: $INPUT_DOCS" >&2
fi

git -C "$ROOT" status --short --untracked-files=all > "$PKG_ROOT/AUDIT_SOURCE_STATUS.txt" 2>/dev/null || true
git -C "$ROOT" diff --binary HEAD > "$PKG_ROOT/AUDIT_TRACKED_DIFF.patch" 2>/dev/null || true

python3 - "$PKG_ROOT" "$STAMP" "$SOURCE_COMMIT" "$SOURCE_DIRTY" <<'PY'
import json, pathlib, platform, sys, time
root = pathlib.Path(sys.argv[1])
stamp = sys.argv[2]
source_commit = sys.argv[3]
source_dirty = sys.argv[4] == "true"
manifest = {
    "schemaVersion": 1,
    "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "stamp": stamp,
    "package": "agentlink-gpt-pro-v0501-codex-phases-reaudit",
    "sourceCommit": source_commit,
    "sourceTreeDirtyAtPackageTime": source_dirty,
    "sourceTruthNote": "This package intentionally includes uncommitted review-diff files for v0.5 Codex phase audit.",
    "intent": "GPT Pro source/runtime re-audit for AgentLink Rescue v0.5.1 corpus, snapshot, gate, report, and rebuilt SQLite/FTS provenance fixes.",
    "expectedEntryPoint": "FULL=1 bash scripts/gpt_pro_v0501_audit.sh",
    "host": {"system": platform.system(), "machine": platform.machine()},
    "excluded": ["*.gguf", "*.safetensors", "*.dmg", "*.zip", "*.tar.gz", ".git", ".gpt-pro-build", "gpt-pro-results", "dist", "bin", "node_modules", ".build"],
}
(root / "PACKAGE_MANIFEST.json").write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
PY

rm -f "$PKG_ROOT/CHECKSUMS.txt"
python3 - "$PKG_ROOT" <<'PY'
import hashlib, pathlib, sys
root = pathlib.Path(sys.argv[1])
lines = []
for p in sorted(root.rglob("*")):
    if not p.is_file():
        continue
    rel = p.relative_to(root).as_posix()
    if rel == "CHECKSUMS.txt":
        continue
    h = hashlib.sha256()
    with p.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    lines.append(f"{h.hexdigest()}  {rel}\n")
(root / "CHECKSUMS.txt").write_text("".join(lines))
PY

(cd "$PKG_ROOT" && shasum -a 256 -c CHECKSUMS.txt >/dev/null)

forbidden="$(find "$PKG_ROOT" \( -name '*.gguf' -o -name '*.safetensors' -o -name '*.dmg' -o -name '*.zip' -o -name '*.tar.gz' -o -name 'node_modules' -o -path '*/.build/*' -o -size +50M \) -print)"
if [ -n "$forbidden" ]; then
  echo "forbidden package contents:" >&2
  echo "$forbidden" >&2
  exit 1
fi

(cd "$OUT_ROOT" && zip -qr -X "$ZIP_PATH" AgentLink-Rescue)
unzip -tq "$ZIP_PATH" >/dev/null

sha="$(shasum -a 256 "$ZIP_PATH" | awk '{print $1}')"
cat <<EOF
{
  "status": "packaged",
  "zipPath": "$ZIP_PATH",
  "sha256": "$sha",
  "packageRoot": "$PKG_ROOT"
}
EOF
