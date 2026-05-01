#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${AGENTLINK_BIN:-$ROOT/dist/Cactus-AgentLink-Rescue/bin/agentlink}"

if [ ! -x "$BIN" ]; then
  "$ROOT/scripts/build.sh"
  "$ROOT/scripts/package.sh"
fi

TMPHOME="$(mktemp -d)"
cleanup() {
  if [ "${KEEP_TMPHOME:-0}" = "1" ]; then
    echo "KEEP_TMPHOME=1; temp home kept: $TMPHOME"
  else
    rm -rf "$TMPHOME"
  fi
}
trap cleanup EXIT

FAKE_SECRET="sk-test-THIS_SHOULD_NOT_LEAK-temp-home-1234567890"

run_home() {
  HOME="$TMPHOME" DEEPSEEK_API_KEY="$FAKE_SECRET" "$BIN" "$@"
}

assert_contains() {
  file="$1"
  needle="$2"
  if ! grep -F "$needle" "$file" >/dev/null; then
    echo "missing expected text in $file: $needle" >&2
    exit 1
  fi
}

run_home doctor --json > "$TMPHOME/doctor.json"

run_home recipe run macos-zsh-path-repair --dry-run > "$TMPHOME/path-dry-run.txt"
if [ -e "$TMPHOME/.zshrc" ]; then
  echo "dry-run created or changed .zshrc" >&2
  exit 1
fi

run_home recipe run macos-zsh-path-repair --yes > "$TMPHOME/path-run-1.txt"
run_home recipe run macos-zsh-path-repair --yes > "$TMPHOME/path-run-2.txt"

block_count="$(grep -c '^# >>> AGENTLINK_PATH_BLOCK >>>$' "$TMPHOME/.zshrc")"
marker_occurrences="$(grep -c 'AGENTLINK_PATH_BLOCK' "$TMPHOME/.zshrc")"
echo "managed block count: $block_count"
echo "marker occurrences: $marker_occurrences"
if [ "$block_count" != "1" ]; then
  echo "managed block duplicated" >&2
  exit 1
fi

run_home restore last > "$TMPHOME/path-restore.txt"
if [ -e "$TMPHOME/.zshrc" ]; then
  echo ".zshrc was not restored to prior absent state" >&2
  exit 1
fi

mkdir -p "$TMPHOME/.codex"
printf '[broken\n' > "$TMPHOME/.codex/config.toml"
cp "$TMPHOME/.codex/config.toml" "$TMPHOME/config.before"
run_home recipe run codex-config-parse-repair --dry-run > "$TMPHOME/codex-parse-dry-run.txt"
if ! cmp -s "$TMPHOME/config.before" "$TMPHOME/.codex/config.toml"; then
  echo "codex parse dry-run changed malformed config" >&2
  exit 1
fi

run_home recipe run codex-config-parse-repair --yes > "$TMPHOME/codex-parse-run.txt"
run_home recipe run codex-deepseek-provider-config --yes > "$TMPHOME/codex-provider-run.txt"

assert_contains "$TMPHOME/.codex/config.toml" 'name = "DeepSeek"'
assert_contains "$TMPHOME/.codex/config.toml" 'base_url = "https://api.deepseek.com"'
assert_contains "$TMPHOME/.codex/config.toml" 'env_key = "DEEPSEEK_API_KEY"'
assert_contains "$TMPHOME/.codex/config.toml" 'model = "deepseek-v4-flash"'
if grep -F "$FAKE_SECRET" "$TMPHOME/.codex/config.toml" >/dev/null; then
  echo "literal API key was written to config" >&2
  exit 1
fi

run_home report --for-codex --latest > "$TMPHOME/agent-dispatch.json"
if grep -R 'THIS_SHOULD_NOT_LEAK' "$TMPHOME/agent-dispatch.json" "$TMPHOME/Library/Application Support/Cactus AgentLink Rescue" 2>/dev/null; then
  echo "fake secret leaked into report/session output" >&2
  exit 1
fi

echo "dogfood temp HOME OK: $TMPHOME"
