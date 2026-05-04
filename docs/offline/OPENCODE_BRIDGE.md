# OpenCode Bridge

OpenCode is integrated as a local coding/report-analysis harness, not as the system rescue executor.

AgentLink may help install OpenCode from official sources and configure a local Gemma provider through `llama-server`.

Safe commands:

```bash
agentlink opencode doctor --json
agentlink opencode install --dry-run --json
agentlink opencode configure-local-gemma --dry-run --json
agentlink brain server start --port 8080 --json
```

OpenCode must not run `sudo`, `networksetup`, `route`, `ifconfig`, `launchctl`, or `killall` as system rescue. Use AgentLink Guided Rescue or Terminal Repair Tickets for that.

