# Codex Config Repair

`codex-config-parse-repair` is only a syntax recovery recipe.

It may replace malformed `~/.codex/config.toml` with a minimal valid TOML template after snapshot/backup, but it does not claim a provider or model is operational.

It does not write:

- literal API keys
- `env_key` secrets
- a hardcoded operational model
- provider compatibility claims

Provider setup remains separate:

- `codex-deepseek-provider-config` writes the managed DeepSeek provider template with `name`, `base_url`, `env_key`, and `model`.
- Online provider compatibility is not proven until an explicit smoke test passes.

