# Rescue Orchestrator

v0.5.0 turns AgentLink from a feature panel into a bounded local rescue orchestrator.

Flow:

```text
Facts collection
Gemma Rescue Supervisor
validated RescuePlan JSON
deterministic recipe/action runner
Terminal-assisted sudo ticket when needed
verifier
automatic rollback if worsened
restart gate if macOS runtime state cannot be released
incident/support report
final status
```

The GUI is only a shell. It calls package-local `bin/agentlink`. The runner owns execution. The verifier owns truth. Writable/system repair requires snapshot or restore point. Gemma cannot execute shell.

TUN-specific policy:

- If `CLASH_TUN_ACTIVE_OR_STALE` is high-confidence, targeted TUN repair is preferred.
- Standard/deep network reset are fallback layers.
- Clean reinstall is final resort only.

