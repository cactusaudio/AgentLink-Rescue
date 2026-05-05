# `.cactus/flywheel/` — per-project flywheel state

Created and managed by the `cactus-flywheel` Claude Code skill.

## Files at this level

- `config.json` — run configuration (copy of `default-config.json` you can edit)
- `config.schema.json` — JSON Schema for validation
- `hooks/errors.log` — anything that went wrong inside a hook
- `hooks/events.jsonl` — observability log of hook firings
- `runs/current` — plain text, contains the current run id
- `runs/<run_id>/` — one directory per run; see SKILL.md

## You generally do not edit these by hand

The Python dispatcher (`~/.claude/hooks/cactus-flywheel/flywheel.py`) writes
state atomically and updates the ledger. User scripts in
`~/scripts/cactus-flywheel/` are thin wrappers around it.

## Disable the loop temporarily

```
export CACTUS_FLYWHEEL_DISABLE=1   # every hook becomes a no-op
```

or finalise the run:

```
scripts/cactus-flywheel/finalize.sh
```

## Inspect

```
scripts/cactus-flywheel/status.sh
scripts/cactus-flywheel/status.sh --json
```
