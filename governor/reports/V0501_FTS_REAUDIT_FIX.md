# AgentLink Rescue v0.5.1 FTS Re-Audit Fix

## Verdict

GPT Pro correctly failed the prior v0.5.1 audit package because the rebuilt
SQLite/FTS index had usable row counts but did not return stored card IDs from
`cards_fts`. This patch closes that specific failure mode and strengthens the
audit harness so the same false pass cannot recur.

## Fixed

- Rebuilt `cards_fts` now stores retrievable `id` and `title` values instead of
  using a contentless FTS table that returns `NULL`.
- `index stats --json` now inspects the actual index database used by diagnosis
  and reports:
  - `ftsIdsUsable`
  - `ftsNullIds`
  - the concrete `sqlitePath`
- Diagnosis candidate selection now tracks provenance. The reason
  `SQLite/FTS candidate retrieval selected this card` is attached only to cards
  whose IDs came from the FTS query.
- Route-expanded cards remain allowed, but they are no longer labeled as FTS
  hits.
- The GPT Pro audit script now asserts:
  - rebuilt FTS row count is `300`
  - `ftsIdsUsable=true`
  - `ftsNullIds=0`
  - a real FTS query returns non-empty `id,title`
  - FTS-labeled diagnosis hypotheses are actually present in the FTS result set

## Verification

Local verification run on 2026-05-20:

```text
go test ./internal/genome ./internal/cli -count=1
go vet ./...
FULL=0 OUT=/tmp/agentlink-v0501-fts-reaudit bash scripts/gpt_pro_v0500_audit.sh
```

Observed:

```json
{
  "cards": 300,
  "layers": 15,
  "ftsRows": 300,
  "ftsIdsUsable": true,
  "ftsNullIds": 0
}
```

Representative rebuilt FTS query:

```text
browser OR codex OR fails
```

returned real card IDs and titles, including `CODEX-API-001` and related
`DEV-CODEX-*` cards.

## Boundary

This is still a source/runtime audit package. It is not a real macOS mutation
proof, not GUI proof, not model proof, and not a signed release package.
