# Last-Good Profiles

Last-Good Profiles save known-good AI-path configuration so AgentLink can restore the path back to AI agents without reinstalling everything.

Saved scope in v0.4.4:

- Codex config
- Claude settings
- Gemini settings
- redacted proxy readiness facts

Restore behavior:

- user-level only
- no sudo
- snapshot before restore
- rollback through `agentlink restore last`
- no project files, dependency caches, browser data, shell history, or private keys

CLI:

```bash
agentlink last-good save --json
agentlink last-good list --json
agentlink last-good inspect <id> --json
agentlink last-good restore --last --yes --json
```

