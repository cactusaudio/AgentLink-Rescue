# Rescue Orchestrator

The v0.5.1 Rescue Orchestrator is the user-facing rescue loop owned by the `agentlink` CLI kernel. `agentlink guided rescue` remains as a deprecated v0.4 compatibility alias; new GUI, scripts, and docs should use `agentlink orchestrator rescue`.

Command:

```bash
agentlink orchestrator rescue --target auto --dry-run --json
agentlink orchestrator rescue --target clash-tun --dry-run --json
agentlink orchestrator rescue --target clash-tun --yes --json
```

State machine:

1. Start
2. CollectFacts
3. Classify
4. DeterministicCandidateSelection
5. GemmaSuperviseIfAvailable
6. ValidatePlan
7. DryRun
8. Snapshot
9. ExecuteReversibleRecipe
10. Verify
11. RetryOrRollback
12. FinalReport

Policy:

- Default mode is dry-run. It is read-only and does not create a mutation snapshot.
- `--yes` may execute only `read_only`, `safe_patch`, and `reversible_patch` recipes.
- Writable recipes still create a snapshot before mutation.
- Verifier failure after mutation triggers rollback where a snapshot exists.
- Gemma can propose a rescue path only through RescuePlanDecision JSON.
- RescuePlanDecision validation rejects unknown recipes, low-confidence repairs, high-risk actions, invalid JSON, raw shell, and any unsupported path.
- Network rescue safe/standard-system-reset/deep is never auto-run by the GUI. Privileged repair is converted into a Terminal repair ticket.
- If Clash/Mihomo TUN signatures are high-confidence, targeted `tun` repair is preferred over `standard` or `standard-system-reset`.
- `clean-baseline` is a last-resort consent path only after targeted repair, rollback/restart gate, or support review shows no safer remaining action.
- Privileged and destructive actions are refused.
- No sudo, password handling, paid API smoke test, remote mutation, queue, RAG, daemon, or helper is used.

Loop budget:

- Default max cycles: 3.
- Default max runtime: 10 minutes.
- Stop after repeated verifier failures or repeated equivalent unsafe decisions.
- If no safe candidate exists, stop with `manual_action_required` or `no_safe_action`.

GUI boundary:

The GUI calls `agentlink orchestrator rescue ... --json` and renders the report. The GUI does not run the loop itself. Normal mode shows one main Fix button; Developer Mode exposes Expert Console surfaces for manual diagnostics and command previews.
