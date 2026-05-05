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

Primary GUI policy:

- Normal mode presents one main action: Fix My Connection. Field packages may bias detection toward known MacBook network failure patterns, but the product promise remains fixing connectivity; Clash/TUN is only one possible cause.
- The main action runs the CLI Rescue Orchestrator first; it does not run sudo from the GUI.
- Developer Mode exposes Expert Console tools for Doctor, Brain, Installer Center, Readiness, raw rescue levels, rollback, reports, and OpenCode status.
- Disabled actions must have an operator-facing reason such as analyze first, no rollback checkpoint, requires Terminal admin repair, or Brain assets missing.

TUN-specific policy:

- If `CLASH_TUN_ACTIVE_OR_STALE` is high-confidence, targeted TUN repair is preferred.
- Standard/deep network reset are fallback layers.
- Clean network baseline reset is a last-resort consent gate, not standard repair.
- Clean reinstall is final resort only.

Supervisor semantics:

- `brainAvailable` means Gemma assets and runtime are present.
- `gemmaCalled` means Gemma actually produced a supervisor decision for this run.
- `supervisorMode` is `deterministic`, `gemma`, or `deterministic_with_gemma_commentary`.
- Gemma decisions are rejected if they select unknown recipes, raw shell, privileged repair without Terminal ticket, standard/deep first for high-confidence TUN, or low-confidence mutation.
