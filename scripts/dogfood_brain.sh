#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -x bin/agentlink ]; then
  scripts/build.sh
fi

ARCH="$(uname -m)"
MODEL="$ROOT/assets/models/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf"
LLAMA="$ROOT/assets/runtimes/llama.cpp/$ARCH/llama-cli"
if [ ! -f "$MODEL" ] || [ ! -x "$LLAMA" ]; then
  echo "brain assets missing; run scripts/fetch_brain_assets.sh" >&2
  exit 1
fi

TMPHOME="$(mktemp -d)"
if [ "${KEEP_TMPHOME:-0}" != "1" ]; then
  trap 'rm -rf "$TMPHOME"' EXIT
else
  echo "KEEP_TMPHOME=1; preserving $TMPHOME"
fi

export HOME="$TMPHOME"
export AGENTLINK_MODEL_PATH="$MODEL"
export AGENTLINK_LLAMA_CLI="$LLAMA"
export DEEPSEEK_API_KEY="sk-test-THIS_SHOULD_NOT_LEAK-brain-dogfood"

bin/agentlink brain doctor
bin/agentlink brain selftest
bin/agentlink brain plan --target path

bin/agentlink repair --auto --brain --target path --dry-run
if [ -f "$TMPHOME/.zshrc" ]; then
  echo "brain dry-run modified temp HOME" >&2
  exit 1
fi

bin/agentlink repair --auto --brain --target path --yes
BLOCKS="$(grep -c '>>> AGENTLINK_PATH_BLOCK >>>' "$TMPHOME/.zshrc")"
if [ "$BLOCKS" != "1" ]; then
  echo "managed block count was $BLOCKS, expected 1" >&2
  exit 1
fi

bin/agentlink report --for-codex --latest > "$TMPHOME/agent-dispatch.json"
if grep -R 'THIS_SHOULD_NOT_LEAK' "$TMPHOME/agent-dispatch.json" "$TMPHOME/Library/Application Support/Cactus AgentLink Rescue" 2>/dev/null; then
  echo "brain report leaked fake secret" >&2
  exit 1
fi

bin/agentlink restore last
if [ -f "$TMPHOME/.zshrc" ]; then
  echo "brain rollback did not restore absent .zshrc" >&2
  exit 1
fi

echo "dogfood brain OK: $TMPHOME"
