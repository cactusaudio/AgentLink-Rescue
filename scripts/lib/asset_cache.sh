#!/bin/bash

ASSET_CACHE="${AGENTLINK_ASSET_CACHE:-$HOME/CactusLocalAgent/.asset-cache/AgentLink-Rescue}"
GEMMA_MODEL_FILE="gemma-4-E4B-it-Q4_K_M.gguf"

asset_host_arch() {
  case "$(uname -m)" in
    arm64) echo "arm64" ;;
    x86_64) echo "amd64" ;;
    *) return 1 ;;
  esac
}

resolve_model_path() {
  if [ -n "${AGENTLINK_MODEL_PATH:-}" ] && [ -f "$AGENTLINK_MODEL_PATH" ]; then
    echo "$AGENTLINK_MODEL_PATH"
    return 0
  fi
  if [ -n "${ROOT:-}" ] && [ -f "$ROOT/assets/models/$GEMMA_MODEL_FILE" ]; then
    echo "$ROOT/assets/models/$GEMMA_MODEL_FILE"
    return 0
  fi
  if [ -f "$ASSET_CACHE/models/$GEMMA_MODEL_FILE" ]; then
    echo "$ASSET_CACHE/models/$GEMMA_MODEL_FILE"
    return 0
  fi
  return 1
}

resolve_llama_runtime_dir() {
  local arch="${1:-}"
  if [ -z "$arch" ]; then
    arch="$(asset_host_arch)" || return 1
  fi
  if [ -n "${AGENTLINK_LLAMA_CLI:-}" ] && [ -x "$AGENTLINK_LLAMA_CLI" ]; then
    dirname "$AGENTLINK_LLAMA_CLI"
    return 0
  fi
  if [ -n "${ROOT:-}" ] && [ -x "$ROOT/assets/runtimes/llama.cpp/$arch/llama-cli" ]; then
    echo "$ROOT/assets/runtimes/llama.cpp/$arch"
    return 0
  fi
  if [ -x "$ASSET_CACHE/runtimes/llama.cpp/$arch/llama-cli" ]; then
    echo "$ASSET_CACHE/runtimes/llama.cpp/$arch"
    return 0
  fi
  return 1
}

resolve_clash_dmg_dir() {
  local installer_arch="${1:-}"
  if [ -z "$installer_arch" ]; then
    case "$(uname -m)" in
      arm64) installer_arch="macos-arm64" ;;
      x86_64) installer_arch="macos-amd64" ;;
      *) return 1 ;;
    esac
  fi
  if [ -n "${ROOT:-}" ] && find "$ROOT/assets/installers/clash-verge-rev/$installer_arch" -maxdepth 1 -name '*.dmg' -print -quit 2>/dev/null | grep . >/dev/null; then
    echo "$ROOT/assets/installers/clash-verge-rev/$installer_arch"
    return 0
  fi
  if find "$ASSET_CACHE/installers/clash-verge-rev/$installer_arch" -maxdepth 1 -name '*.dmg' -print -quit 2>/dev/null | grep . >/dev/null; then
    echo "$ASSET_CACHE/installers/clash-verge-rev/$installer_arch"
    return 0
  fi
  return 1
}

copy_model_to_staging() {
  local dest="$1"
  local model
  model="$(resolve_model_path)"
  mkdir -p "$dest"
  cp "$model" "$dest/"
}

copy_runtime_to_staging() {
  local dest="$1"
  local arch="${2:-}"
  local runtime_dir
  runtime_dir="$(resolve_llama_runtime_dir "$arch")"
  mkdir -p "$dest"
  cp "$runtime_dir/llama-cli" "$dest/"
  if [ -f "$runtime_dir/llama-completion" ]; then
    cp "$runtime_dir/llama-completion" "$dest/"
  fi
  cp "$runtime_dir"/*.dylib "$dest/" 2>/dev/null || true
  if [ -f "$runtime_dir/LICENSE" ]; then
    cp "$runtime_dir/LICENSE" "$dest/"
  fi
}

copy_clash_dmg_to_staging() {
  local dest="$1"
  local installer_arch="${2:-}"
  local dmg_dir
  dmg_dir="$(resolve_clash_dmg_dir "$installer_arch")"
  mkdir -p "$dest"
  cp "$dmg_dir"/*.dmg "$dest/"
  if [ -f "$(dirname "$dmg_dir")/manifest.lock.json" ]; then
    cp "$(dirname "$dmg_dir")/manifest.lock.json" "$(dirname "$dest")/"
  fi
}

verify_sha256_if_known() {
  local file="$1"
  local expected="${2:-}"
  if [ -z "$expected" ]; then
    return 0
  fi
  local actual
  actual="$(/usr/bin/shasum -a 256 "$file" | awk '{print $1}')"
  [ "$actual" = "$expected" ]
}
