# Brain Runtime Policy

v0.3 Brain mode is bounded:

- Qwen is controller/planner, not executor.
- Qwen outputs only `PlannerDecision` JSON.
- The runner executes only recipe actions already present in the local catalog.
- Verifiers are the truth source.
- `repair --auto --brain --dry-run` never mutates files.
- `repair --auto --brain --yes` may execute read-only, safe patch, and reversible patch recipes with snapshot-first mutation.
- `network_action` requires explicit online authorization and is not used for paid API calls.
- `privileged_action` and `destructive_action` are refused in the automatic Brain loop.
- Network rescue safe/standard/deep remains an explicit CLI action and is not auto-invoked by Qwen.
- Planner prompts, outputs, session logs, and reports are redacted.
