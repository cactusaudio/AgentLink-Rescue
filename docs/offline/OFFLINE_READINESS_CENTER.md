# Offline Readiness Center

Offline Readiness is scoped to AgentLink rescue readiness, not general offline development.

It checks:

- network and endpoint classifications
- proxy environment state
- Gemma Brain model/runtime availability
- recovery installers
- API key presence, redacted
- base tools needed by AI CLI installers

It does not index repositories, cache dependencies, run tests, generate patches, or manage project workspaces.

CLI:

```bash
agentlink readiness doctor --json
agentlink dev doctor --json
```

GUI:

Open `Readiness Center`.

