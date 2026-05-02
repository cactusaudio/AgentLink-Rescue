# Qwen Brain Pack

Cactus AgentLink Rescue v0.3 can use a local Qwen GGUF model through `llama-cli`.

The Brain is planner-only. It produces `PlannerDecision` JSON. It does not execute commands, shell scripts, network rescue, or arbitrary text from the model. The deterministic recipe runner remains the only executor.

Default model:

- Model ID: `qwen3-4b-instruct-2507-q4km`
- GGUF: `Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf`
- Repo: `bartowski/Qwen_Qwen3-4B-Instruct-2507-GGUF`
- SHA256: `2fde00ce69dd4899c70d020845e2638353015bba0fdf161b3eb965f2bca4464e`

Asset resolution:

1. Package-local assets under `assets/models/` and `assets/runtimes/`.
2. User cache under `~/Library/Application Support/Cactus AgentLink Rescue/assets/`.
3. Environment overrides: `AGENTLINK_MODEL_PATH`, `AGENTLINK_LLAMA_CLI`, `AGENTLINK_BRAIN_HOME`.

Core package users can run:

```sh
./bin/agentlink brain fetch
```

Source checkout users can run:

```sh
./scripts/fetch_brain_assets.sh
```

No telemetry is sent. Brain inference is local.
