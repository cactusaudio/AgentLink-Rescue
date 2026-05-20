# GPT Pro Audit Entry: AgentLink Rescue v0.5.1 Codex Phases Re-Audit

This package is an independent re-audit bundle for the AgentLink Rescue v0.5.1 Codex phase implementation.

It is not a release package and it is not a real macOS repair proof. It is a fixture/read-only/dry-run/source audit package for GPT Pro or another cloud reviewer.

The prior v0.5.1 audit correctly failed because rebuilt SQLite/FTS rows did not
return real card IDs and diagnosis could over-label route-expanded candidates as
FTS hits. This package is specifically meant to verify that fix.

## Start Here

Read this file first, then read:

```text
docs/testing/GPT_PRO_V0500_CODEX_PHASES_AUDIT_PROMPT.md
audit-inputs/v0500-codex-docs/AGENTS.md
audit-inputs/v0500-codex-docs/codex_prompt_phase_1_corpus_retriever.md
audit-inputs/v0500-codex-docs/codex_prompt_phase_2_collect_snapshot.md
audit-inputs/v0500-codex-docs/codex_prompt_phase_3_recipe_gate_report.md
governor/reports/V0500_CODEX_PHASES_FINAL.md
governor/reports/V0501_FTS_REAUDIT_FIX.md
```

## Main Command

From the extracted package root:

```bash
cd AgentLink-Rescue
FULL=1 OUT="$PWD/gpt-pro-results/v0501-fts-reaudit" bash scripts/gpt_pro_v0501_audit.sh
```

`scripts/gpt_pro_v0500_audit.sh` remains as a compatibility alias for the
original phase-audit line. If the cloud runtime is time-limited, run the same
command with `FULL=0`; that still runs the v0.5.1 phase audit, targeted Go
tests, vet, and CLI smoke, but skips the long historical `go test ./...` chaos
matrix.

## Hard Safety Rules

Do not run:

```text
sudo
agentlink rescue --yes
agentlink guided rescue --yes
networksetup mutation
route mutation
ifconfig mutation
launchctl mutation
pf mutation
VPN mutation
Keychain extraction
Wi-Fi password extraction
rm -rf outside a temporary sandbox
```

Allowed:

```text
go test ...
go vet ...
go build ...
agentlink-rescue index rebuild/stats
agentlink-rescue explain
agentlink-rescue diagnose --no-snapshot
agentlink-rescue collect --privacy strict
agentlink-rescue features
agentlink-rescue gate
agentlink-rescue report
```

## What This Audit Can Prove

- The 300-card network genome corpus is present, loadable, valid, and searchable.
- The new `agentlink-rescue` CLI surfaces run from clean source.
- The SQLite/FTS index can be rebuilt from `cards/failure_cards.json`.
- Rebuilt `cards_fts` returns non-null card IDs/titles from real FTS queries.
- `index stats --json` reports the concrete SQLite DB plus `ftsIdsUsable=true`
  and `ftsNullIds=0`.
- Diagnosis marks a card as selected by SQLite/FTS only when that card ID came
  from FTS, not merely from route expansion.
- Diagnosis uses SQLite/FTS candidate retrieval before deterministic card ranking.
- Runtime loads `ontology/symptom_routes.yaml` for route priors.
- Snapshot collection is read-only and redacted by default.
- Strict privacy mode redacts secrets and host-identifying details.
- Docker config collection emits a safe summary only; it must not dump auth
  blobs, identity tokens, registry names, credential-store values, usernames,
  or passwords.
- Diagnosis returns card-backed hypotheses.
- Recipe gate decisions are conservative around high-risk, AoIP, MDM, and missing verifier/rollback cases.
- Markdown and JSON reports can be generated from a snapshot.
- The implementation does not require model weights, llama runtimes, DMGs, package zips, or host asset caches.

## What This Audit Cannot Prove

- Real macOS network mutation.
- GUI behavior.
- signing/notarization.
- Gemma/Qwen/OpenCode behavior.
- Commercial release readiness outside this v0.5 phase surface.

## Package Integrity

Run:

```bash
shasum -a 256 -c CHECKSUMS.txt
```

`CHECKSUMS.txt` deliberately does not include itself.
