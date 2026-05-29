# OpenCode Bridge

OpenCode Bridge remains experimental in the v0.5.x line. It is not a first-class AgentLink harness and is not part of the system rescue execution path.

AgentLink may help install OpenCode from official sources and generate a local Gemma provider template through `llama-server`, but the lab verdict is negative for this release:

- OpenCode 1.14.33 installed.
- Gemma local server worked through `/v1/models` and `/v1/chat/completions`.
- OpenCode + Gemma ran 9 tasks / 18 attempts.
- Passed: 0.
- Failed: 9.
- Top failure class: OpenCode provider/config failure.
- Default OpenCode run context overflowed AgentLink's 8k server profile and remained too slow after a manual 16k restart.
- Plugin scaffold status is `scaffold_available`; real plugin loading is not verified.

Allowed statuses:

- `experimental`
- `scaffold_available`
- `template_generated`
- `manual_merge_required`
- `not_verified`

Do not claim OpenCode Local Agent ready, configured, verified, or working in the current v0.5.2 field beta.

Safe commands:

```bash
agentlink opencode doctor --json
agentlink opencode install --dry-run --json
agentlink opencode configure-local-gemma --dry-run --json
agentlink brain server start --port 8080 --json
```

OpenCode must not run `sudo`, `networksetup`, `route`, `ifconfig`, `launchctl`, or `killall` as system rescue. Use AgentLink Rescue Orchestrator or Terminal Repair Tickets for that.

Gemma direct Supervisor remains the AgentLink rescue path. OpenCode may be revisited later for Local Toolkit coding/report-analysis tasks.
