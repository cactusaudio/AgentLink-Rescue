#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STAMP="${STAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_ROOT="${OUT_ROOT:-$HOME/CactusLocalAgent/.asset-cache/AgentLink-Rescue/release-artifacts/gpt-pro-v052-gui-beta-$STAMP}"
PKG_ROOT="$OUT_ROOT/AgentLink-Rescue"
ZIP_PATH="$OUT_ROOT/agentlink-gpt-pro-v052-gui-field-beta-audit-$STAMP.zip"
SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || true)"
SOURCE_DIRTY="false"
if [ -n "$(git -C "$ROOT" status --short 2>/dev/null || true)" ]; then
  SOURCE_DIRTY="true"
fi

GO_BIN="${GO_BIN:-${GO:-$(command -v go || true)}}"
if [ -z "$GO_BIN" ]; then
  echo "go is required to vendor dependencies for the GPT Pro audit package" >&2
  exit 2
fi

for tool in python3 rsync zip shasum unzip; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "$tool is required to package GPT Pro v0.5.2 GUI beta audit artifacts" >&2
    exit 2
  fi
done

rm -rf "$OUT_ROOT"
mkdir -p "$PKG_ROOT"

rsync -a "$ROOT/" "$PKG_ROOT/" \
  --exclude '.git/' \
  --exclude '.DS_Store' \
  --exclude '.cactus/flywheel/runs/' \
  --exclude '.tmp/' \
  --exclude '.gpt-pro-build/' \
  --exclude 'gpt-pro-results/' \
  --exclude 'dist/' \
  --exclude 'bin/' \
  --exclude 'gui/build/' \
  --exclude '.asset-cache/' \
  --exclude '.build/' \
  --exclude 'DerivedData/' \
  --exclude 'node_modules/' \
  --exclude 'vendor/' \
  --exclude '__pycache__/' \
  --exclude '*.gguf' \
  --exclude '*.safetensors' \
  --exclude '*.dmg' \
  --exclude '*.zip' \
  --exclude '*.tar.gz'

git -C "$ROOT" status --short --untracked-files=all > "$PKG_ROOT/AUDIT_SOURCE_STATUS.txt" 2>/dev/null || true
if [ ! -s "$PKG_ROOT/AUDIT_SOURCE_STATUS.txt" ]; then
  echo "clean" > "$PKG_ROOT/AUDIT_SOURCE_STATUS.txt"
fi
git -C "$ROOT" diff --binary HEAD > "$PKG_ROOT/AUDIT_TRACKED_DIFF.patch" 2>/dev/null || true

(cd "$PKG_ROOT" && "$GO_BIN" mod vendor)

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
    "package": "agentlink-gpt-pro-v052-gui-field-beta-audit",
    "sourceCommit": source_commit,
    "expectedTag": "v0.5.2-gui-field-beta-alpha",
    "expectedCLIReportedVersion": "agentlink 0.5.1",
    "sourceTreeDirtyAtPackageTime": source_dirty,
    "sourceTruthNote": "This package should be produced from the committed/tagged v0.5.2 GUI field beta source tree. v0.5.2 is the GUI/audit-package line; the shared CLI core still reports agentlink 0.5.1 until the public release version bump.",
    "vendoredDependenciesGeneratedAtPackageTime": True,
    "intent": "GPT Pro source/runtime audit for AgentLink Rescue v0.5.2 GUI controlled field beta readiness.",
    "expectedEntryPoint": "FULL=1 bash scripts/gpt_pro_v052_gui_beta_audit.sh",
    "primaryPrompt": "docs/testing/GPT_PRO_V052_GUI_FIELD_BETA_AUDIT_PROMPT.md",
    "host": {"system": platform.system(), "machine": platform.machine()},
    "excluded": ["*.gguf", "*.safetensors", "*.dmg", "*.zip", "*.tar.gz", ".git", ".cactus/flywheel/runs", ".tmp", ".gpt-pro-build", "gpt-pro-results", "dist", "bin", "gui/build", "node_modules", ".build"],
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

forbidden="$(find "$PKG_ROOT" \( -name '*.gguf' -o -name '*.safetensors' -o -name '*.dmg' -o -name '*.zip' -o -name '*.tar.gz' -o -name 'node_modules' -o -path '*/.cactus/flywheel/runs/*' -o -path '*/.tmp/*' -o -path '*/.build/*' -o -path '*/gui/build/*' -o -size +50M \) -print)"
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
