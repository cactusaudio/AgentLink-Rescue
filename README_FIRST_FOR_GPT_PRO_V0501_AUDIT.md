# GPT Pro Audit Entry: AgentLink Rescue v0.5.1 Codex Phases Re-Audit

This is the canonical v0.5.1 GPT Pro audit entry.

Read:

```text
README_FIRST_FOR_GPT_PRO_V0500_AUDIT.md
docs/testing/GPT_PRO_V0500_CODEX_PHASES_AUDIT_PROMPT.md
governor/reports/V0501_FTS_REAUDIT_FIX.md
```

The `V0500` filenames are retained as compatibility aliases for the original
phase-audit line. For v0.5.1, run:

```bash
FULL=1 OUT="$PWD/gpt-pro-results/v0501-fts-reaudit" bash scripts/gpt_pro_v0501_audit.sh
```

The hard acceptance point is rebuilt SQLite/FTS truth, not row count:

```text
ftsRows == 300
ftsIdsUsable == true
ftsNullIds == 0
cards_fts query returns non-null id/title values
FTS whyMatched appears only for card IDs returned by FTS
```
