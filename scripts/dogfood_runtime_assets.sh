#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MODEL="assets/models/gemma-4-E4B-it-Q4_K_M.gguf"
case "$(uname -m)" in
  arm64) ARCH="arm64" ;;
  x86_64) ARCH="amd64" ;;
  *) echo "unsupported macOS arch: $(uname -m)" >&2; exit 1 ;;
esac
RUNTIME="assets/runtimes/llama.cpp/$ARCH/llama-cli"

if [ ! -f "$MODEL" ]; then
  echo "missing model: $MODEL" >&2
  exit 1
fi
ACTUAL_MODEL_SHA="$(/usr/bin/shasum -a 256 "$MODEL" | awk '{print $1}')"
LOCK_SHA="$(/usr/bin/python3 - assets/manifests/manifest.lock.json <<'PY' 2>/dev/null || true
import json, sys
try:
    print(json.load(open(sys.argv[1])).get("model", {}).get("sha256", ""))
except Exception:
    print("")
PY
)"
if [ -n "$LOCK_SHA" ] && [ "$LOCK_SHA" != "$ACTUAL_MODEL_SHA" ]; then
  echo "model checksum mismatch against manifest.lock.json" >&2
  exit 1
fi
if [ ! -x "$RUNTIME" ]; then
  echo "missing executable runtime: $RUNTIME" >&2
  exit 1
fi

if [ ! -x bin/agentlink ]; then
  scripts/build.sh
fi

./bin/agentlink version | grep '0.4.1'
./bin/agentlink brain doctor --json > /tmp/agentlink-runtime-assets-brain-doctor.json
/usr/bin/python3 - /tmp/agentlink-runtime-assets-brain-doctor.json <<'PY'
import json, sys
doc=json.load(open(sys.argv[1]))
assert doc["brainPackAvailable"] is True, doc
assert doc["modelSha256OK"] is True, doc
assert doc["runtimeExecutable"] is True, doc
assert doc.get("modelFamily") == "gemma", doc
assert doc.get("modelID") == "gemma-4-e4b-it-q4km", doc
PY
./bin/agentlink brain selftest --json > /tmp/agentlink-runtime-assets-brain-selftest.json
/usr/bin/python3 - /tmp/agentlink-runtime-assets-brain-selftest.json <<'PY'
import json, sys
res=json.load(open(sys.argv[1]))
assert res["ok"] is True, res
PY

echo "dogfood runtime assets OK"
