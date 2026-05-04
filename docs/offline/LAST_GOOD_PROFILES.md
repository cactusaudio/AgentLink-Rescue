# Last-Good Profiles

Last-Good Profiles save known-good AI-path configuration so AgentLink can restore the path back to AI agents without reinstalling everything.

Saved scope in v0.5.0:

- Codex config
- Claude settings
- Gemini settings
- redacted proxy readiness facts

Restore behavior:

- user-level only
- no sudo
- snapshot before restore
- only restores known AI config paths:
  - `~/.codex/config.toml`
  - `~/.claude/settings.json`
  - `~/.claude.json`
  - `~/.gemini/settings.json`
  - `~/.config/gemini/settings.json`
- refuses manifest paths outside that allowlist, including path traversal
- refuses stored files outside the selected Last-Good profile directory
- preserves original file permissions when known
- defaults restored config permissions to `0600` when original mode is unavailable
- rollback through `agentlink restore last`
- no project files, dependency caches, browser data, shell history, or private keys

CLI:

```bash
agentlink last-good save --json
agentlink last-good list --json
agentlink last-good inspect <id> --json
agentlink last-good restore --last --yes --json
```
