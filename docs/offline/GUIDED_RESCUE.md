# Guided Rescue

Guided Rescue is the user-facing rescue loop owned by the `agentlink` CLI kernel.

Command:

```bash
agentlink guided rescue --target auto --dry-run --json
agentlink guided rescue --target auto --yes --json
```

State machine:

1. Start
2. CollectFacts
3. Classify
4. DeterministicCandidateSelection
5. BrainPlanIfNeeded
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
- Gemma can propose a recipe only through PlannerDecision JSON.
- PlannerDecision validation rejects unknown recipes, low-confidence repairs, high-risk actions, invalid JSON, and any unsupported path.
- Network rescue safe/standard/deep is never auto-run by Guided Rescue. It is reported as a copyable Terminal command.
- Privileged and destructive actions are refused.
- No sudo, password handling, paid API smoke test, remote mutation, queue, RAG, daemon, or helper is used.

Loop budget:

- Default max cycles: 3.
- Default max runtime: 10 minutes.
- Stop after repeated verifier failures or repeated equivalent unsafe decisions.
- If no safe candidate exists, stop with `manual_action_required` or `no_safe_action`.

GUI boundary:

The GUI calls `agentlink guided rescue ... --json` and renders the report. The GUI does not run the loop itself.
