# Gemma Brain Pack

Cactus AgentLink Rescue v0.4.5 can use a local Gemma 4 E4B-it GGUF model through `llama-cli`.

The Brain is planner-only. It produces `PlannerDecision` JSON. It does not execute commands, shell scripts, network rescue, or arbitrary text from the model. The deterministic recipe runner remains the only executor.

Default model:

- Model ID: `gemma-4-e4b-it-q4km`
- Model name: `gemma-4-E4B-it`
- GGUF: `gemma-4-E4B-it-Q4_K_M.gguf`
- Repo: `unsloth/gemma-4-E4B-it-GGUF`
- Approximate model size: 5.07 GB
- License: Apache-2.0

Asset resolution:

1. `AGENTLINK_MODEL_PATH` and `AGENTLINK_LLAMA_CLI` overrides.
2. Package-local assets under `<package-root>/assets/models/` and `<package-root>/assets/runtimes/llama.cpp/<arch>/`.
3. User cache under `~/Library/Application Support/Cactus AgentLink Rescue/assets/`.
4. Source checkout assets under `./assets/` when running from the development tree.
5. `PATH` fallback for `llama-cli` only.

Package-local model path:

```text
assets/models/gemma-4-E4B-it-Q4_K_M.gguf
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
shasum -a 256 assets/models/gemma-4-E4B-it-Q4_K_M.gguf
```

Core package users can run while online:

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
