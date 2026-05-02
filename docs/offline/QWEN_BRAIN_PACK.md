# Qwen Brain Pack

Cactus AgentLink Rescue v0.4.0 can use a local Qwen GGUF model through `llama-cli`.

The Brain is planner-only. It produces `PlannerDecision` JSON. It does not execute commands, shell scripts, network rescue, or arbitrary text from the model. The deterministic recipe runner remains the only executor.

Default model:

- Model ID: `qwen3-4b-instruct-2507-q4km`
- GGUF: `Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf`
- Repo: `bartowski/Qwen_Qwen3-4B-Instruct-2507-GGUF`
- SHA256: `2fde00ce69dd4899c70d020845e2638353015bba0fdf161b3eb965f2bca4464e`

Asset resolution:

1. `AGENTLINK_MODEL_PATH` and `AGENTLINK_LLAMA_CLI` overrides.
2. Package-local assets under `<package-root>/assets/models/` and `<package-root>/assets/runtimes/llama.cpp/<arch>/`.
3. User cache under `~/Library/Application Support/Cactus AgentLink Rescue/assets/`.
4. Source checkout assets under `./assets/` when running from the development tree.
5. `PATH` fallback for `llama-cli` only.

Package-local model path:

```text
assets/models/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf
```

Package-local runtime path:

```text
assets/runtimes/llama.cpp/arm64/llama-cli
assets/runtimes/llama.cpp/amd64/llama-cli
```

Verify assets:

```sh
./bin/agentlink brain doctor --json
./bin/agentlink brain selftest --json
shasum -a 256 assets/models/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf
```

Core package users can run:

```sh
./bin/agentlink brain fetch
```

Source checkout users can run:

```sh
./scripts/fetch_brain_assets.sh
```

Create packages:

```sh
./scripts/package.sh
./scripts/package_brain.sh
```

The Core package does not contain the model or runtime. The Brain package contains both and can run `brain doctor`, `brain selftest`, `brain plan`, and Brain dry-runs offline.

No telemetry is sent. Brain inference is local.
