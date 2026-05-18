#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STAMP="${STAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
OUT_ROOT="${OUT_ROOT:-$HOME/CactusLocalAgent/.asset-cache/AgentLink-Rescue/release-artifacts/gpt-pro-topology-research-$STAMP}"
PKG_ROOT="$OUT_ROOT/AgentLink-Rescue"
ZIP_PATH="$OUT_ROOT/agentlink-gpt-pro-protected-topology-research-$STAMP.zip"
SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || true)"
SOURCE_DIRTY="false"
if [ -n "$(git -C "$ROOT" status --short 2>/dev/null || true)" ]; then
  SOURCE_DIRTY="true"
fi

for tool in python3 rsync zip unzip shasum; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "$tool is required to package GPT Pro topology research artifacts" >&2
    exit 2
  fi
done

rm -rf "$OUT_ROOT"
mkdir -p "$PKG_ROOT"

rsync -a "$ROOT/" "$PKG_ROOT/" \
  --exclude '.git/' \
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
  --exclude '*.tar.gz'

mkdir -p "$PKG_ROOT/external-evidence"
copy_if_present() {
  local src="$1"
  if [ -f "$src" ]; then
    cp "$src" "$PKG_ROOT/external-evidence/"
  fi
}
copy_if_present "/Users/jack/Downloads/agentlink-rescue-dev-memo_dante-vlan-route-trap_2026-05-18.md"
copy_if_present "/Users/jack/Downloads/agentlink_gpt_pro_stress_report_20260518T074925Z.md"
copy_if_present "/Users/jack/Downloads/agentlink_gpt_pro_stress_summary_20260518T074925Z.json"
copy_if_present "/Users/jack/Downloads/agentlink_gpt_pro_stress_results_20260518T074925Z_logs.tgz"

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
    "package": "agentlink-gpt-pro-protected-topology-research",
    "sourceCommit": source_commit,
    "sourceTreeDirtyAtPackageTime": source_dirty,
    "sourceTruthNote": "sourceCommit identifies the base commit; when sourceTreeDirtyAtPackageTime is true, the package also includes uncommitted review diffs.",
    "intent": "GPT Pro protected-topology omission research pack; no model weights, runtimes, DMGs, release zips, or host asset cache.",
    "primaryPrompt": "docs/testing/GPT_PRO_V034_PROTECTED_TOPOLOGY_GATE_PROMPT.md",
    "readFirst": "README_FIRST_FOR_GPT_PRO_TOPOLOGY_RESEARCH.md",
    "relatedStressGuide": "docs/testing/GPT_PRO_PRERELEASE_STRESS_GUIDE.md",
    "host": {"system": platform.system(), "machine": platform.machine()},
    "includedExternalEvidence": sorted(p.name for p in (root / "external-evidence").glob("*")),
    "keyFixtures": [
        "testdata/chaos/network/mazes/m13-dante-vlan-no-internet-trap.json",
        "testdata/chaos/network/mazes/m14-aes67-audio-vlan-route-trap.json",
        "testdata/chaos/network/mazes/m15-ndi-video-vlan-stale-default.json",
        "testdata/chaos/network/mazes/m16-atem-camera-control-static-lan.json",
        "testdata/chaos/network/mazes/m17-artnet-lighting-linklocal.json",
        "testdata/chaos/network/mazes/m18-iscsi-nas-direct-no-internet.json",
        "testdata/chaos/network/mazes/m19-thunderbolt-bridge-direct-mac.json",
        "testdata/chaos/network/mazes/m20-lab-instrument-static-subnet.json",
        "testdata/chaos/network/mazes/m21-vm-bridge-active-false-internet.json",
        "testdata/chaos/network/mazes/m22-mdm-scoped-dns-proxy.json",
        "testdata/chaos/network/mazes/m23-usb-management-wifi-internet.json",
        "testdata/chaos/network/mazes/m24-tun-owned-no-protected-topology.json",
        "testdata/chaos/network/mazes/m25-unbound-raw-audio-false-positive.json"
    ],
    "keyReports": [
        "governor/reports/DANTE_VLAN_ROUTE_TRAP_PATCH_AUDIT_20260518.md",
        "governor/reports/V0340_PROTECTED_TOPOLOGY_IMPLEMENTATION.md",
        "governor/reports/GPT_PRO_CLOUD_GATE_PASS_20260518T074925Z.md",
        "governor/reports/GPT_PRO_CLOUD_GATE_REPAIR_20260518.md"
    ],
    "excluded": ["*.gguf", "*.safetensors", "*.dmg", "*.zip", "*.tar.gz", ".git", ".gpt-pro-build", "gpt-pro-results", "node_modules", ".build", "dist", "bin"],
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

(cd "$OUT_ROOT" && zip -qr "$ZIP_PATH" AgentLink-Rescue)
unzip -tq "$ZIP_PATH" >/dev/null

sha="$(shasum -a 256 "$ZIP_PATH" | awk '{print $1}')"
cat <<EOF
{
  "status": "packaged",
  "zipPath": "$ZIP_PATH",
  "sha256": "$sha",
  "packageRoot": "$PKG_ROOT",
  "prompt": "$PKG_ROOT/docs/testing/GPT_PRO_V034_PROTECTED_TOPOLOGY_GATE_PROMPT.md",
  "readFirst": "$PKG_ROOT/README_FIRST_FOR_GPT_PRO_TOPOLOGY_RESEARCH.md"
}
EOF
