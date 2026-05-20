# GPT Pro Prompt: AgentLink Rescue v0.5.1 Codex Phases Re-Audit

You are auditing an AgentLink Rescue v0.5.1 Codex phase implementation package.

This is an independent source/runtime audit, not a product release gate and not a real macOS network repair proof.

The previous GPT Pro audit correctly failed this line because the rebuilt
SQLite/FTS table had 300 rows but returned `NULL` for card IDs/titles, and
diagnosis could label route-expanded candidates as `SQLite/FTS` hits. This
re-audit must directly verify that those two failures are closed.

## Objective

Determine whether the package genuinely implements the three Codex phase prompts:

1. Phase 1: corpus loader, validator, index rebuild, retriever, and card-backed diagnosis.
2. Phase 2: read-only snapshot collection, redaction, feature extraction, and snapshot-backed diagnosis.
3. Phase 3: recipe gate, verifier/rollback safety classification, and markdown/json reports.

This package also includes the follow-up fixes from the first GPT Pro review:

- Docker config must be summarized, never dumped.
- Strict privacy must cover Docker auth/identity tokens, generic hostnames,
  full private IPs, user paths, and strict reports.
- Feature privacyMode must be preserved.
- `NO_PROXY` must not trigger localhost-proxy false positives.
- `symptom_routes.yaml` must affect diagnosis routes without code changes.
- SQLite/FTS must be used for candidate retrieval, not only built.
- Schema constraints must drive validation beyond hard-coded happy-path checks.

Report product failures, harness failures, cloud-environment limitations, stale-binary contamination, and inconclusive surfaces separately.

## Required First Reads

From the extracted package root, read:

```text
README_FIRST_FOR_GPT_PRO_V0500_AUDIT.md
PACKAGE_MANIFEST.json
audit-inputs/v0500-codex-docs/AGENTS.md
audit-inputs/v0500-codex-docs/codex_prompt_phase_1_corpus_retriever.md
audit-inputs/v0500-codex-docs/codex_prompt_phase_2_collect_snapshot.md
audit-inputs/v0500-codex-docs/codex_prompt_phase_3_recipe_gate_report.md
governor/reports/V0500_CODEX_PHASES_FINAL.md
governor/reports/V0501_FTS_REAUDIT_FIX.md
```

Then inspect the implementation:

```text
cmd/agentlink-rescue/main.go
internal/cli/cli.go
internal/cli/genome.go
internal/genome/genome.go
internal/cli/genome_cli_test.go
internal/genome/genome_test.go
agentlink_rescue_network_genome_v0_2/
```

## Main Command

Run:

```bash
FULL=1 OUT="$PWD/gpt-pro-results/v0501-fts-reaudit" bash scripts/gpt_pro_v0501_audit.sh
```

If the cloud sandbox cannot finish the long full suite, rerun:

```bash
FULL=0 OUT="$PWD/gpt-pro-results/v0501-fts-reaudit-fast" bash scripts/gpt_pro_v0501_audit.sh
```

Do not mark the package fully green if only `FULL=0` was run. Mark it as targeted phase audit pass, full historical regression not run.

## Manual Isolation Commands

Use these only if the main script fails or you need attribution:

```bash
go version
go build -trimpath -ldflags "-s -w" -o .gpt-pro-build/agentlink-rescue ./cmd/agentlink-rescue
.gpt-pro-build/agentlink-rescue index rebuild --json
.gpt-pro-build/agentlink-rescue index stats --json
.gpt-pro-build/agentlink-rescue explain --card MAC-PROXY-002 --json
.gpt-pro-build/agentlink-rescue diagnose --symptom "browser works but codex fails" --no-snapshot --json
.gpt-pro-build/agentlink-rescue diagnose --symptom "clash tun broke dns" --no-snapshot --json
.gpt-pro-build/agentlink-rescue diagnose --symptom "vpn loses lan" --no-snapshot --json
.gpt-pro-build/agentlink-rescue diagnose --symptom "dante devices invisible" --no-snapshot --json
.gpt-pro-build/agentlink-rescue gate --card DANTE-CLOCK-001 --json
.gpt-pro-build/agentlink-rescue gate --card MAC-PROXY-002 --json
go test ./internal/genome ./internal/cli -count=1
go vet ./...
go test ./... -count=1 -timeout 35m
```

For privacy isolation, create a fake HOME with Docker auth and rerun collect:

```bash
mkdir -p /tmp/agentlink-fake-home/.docker
cat >/tmp/agentlink-fake-home/.docker/config.json <<'JSON'
{"auths":{"registry.example.com":{"auth":"dXNlcjpzdXBlcnNlY3JldA==","identitytoken":"dockertoken-SECRET-1234567890"}},"credsStore":"osxkeychain","credHelpers":{"registry.example.com":"desktop"}}
JSON
HOME=/tmp/agentlink-fake-home .gpt-pro-build/agentlink-rescue collect --out /tmp/agentlink-audit-snapshot --privacy strict --json
grep -R "dXNlcjpzdXBlcnNlY3JldA\\|dockertoken-SECRET\\|osxkeychain\\|registry.example.com\\|desktop" /tmp/agentlink-audit-snapshot && echo "LEAK"
```

The grep must find no live Docker values in either `raw/` or `redacted/`.

## Expected Evidence

Minimum expected evidence:

```text
index rebuild: cards=300, layers=15, ftsRows=300
index stats: ftsIdsUsable=true and ftsNullIds=0
SQLite tables include cards and cards_fts
SQLite query against rebuilt cards_fts returns real non-null id/title values
explain MAC-PROXY-002 returns that card and its shell proxy/Codex-style symptom
diagnose "browser works but codex fails" returns card-backed hypotheses including MAC-PROXY-002 in top results
collect --privacy strict succeeds without running mutating commands
features --snapshot succeeds
snapshot-backed diagnose succeeds
DANTE-CLOCK-001 gate=manual_only
MAC-PROXY-002 gate=human_confirmed_allowed
markdown report exists
JSON report exists and parses
go test ./internal/genome ./internal/cli passes
go vet ./... passes
FULL=1: go test ./... passes
fake Docker config secrets absent from raw/redacted/reports
features/snapshot_features.json has privacyMode=strict
NO_PROXY-only fixture does not set system_proxy_localhost or shell_proxy_localhost
diagnosis whyMatched includes SQLite/FTS candidate retrieval evidence
every hypothesis with SQLite/FTS whyMatched has a card ID present in the FTS query result set
route-expanded-only candidates do not get SQLite/FTS whyMatched
corrupt index is rebuilt from corpus JSON
temporary symptom_routes.yaml changes diagnosis route output without code changes
```

Safety expectations:

```text
sudo executed? no
real network mutation executed? no
networksetup mutation executed? no
route mutation executed? no
ifconfig mutation executed? no
launchctl mutation executed? no
model weights present? no
runtime binaries present? no
release archives present? no
live secrets found? no
```

## Audit Questions

Answer these directly:

1. Does the package include the three original Codex phase prompts and the 300-card genome corpus?
2. Does the implementation actually load and validate the corpus rather than hard-code a few examples?
3. Does `index rebuild` produce a real index from `failure_cards.json`?
4. Are diagnosis hypotheses card-backed and explainable?
5. Does snapshot collection stay read-only?
6. Is raw evidence redacted by default?
7. Does strict privacy mode avoid leaking proxy credentials, tokens, hostnames, local paths, and private IP details?
8. Does strict privacy avoid leaking Docker `auth`, `identitytoken`, `credsStore`, `credHelpers`, registry names, usernames, passwords, or registry auth blobs?
9. Does the gate prevent automatic or model-driven repair for AoIP/Dante/AES67/PTP, MDM/policy, missing verifier/rollback, TLS-bypass, global pf flush, or never-auto cards?
10. Does the report surface recommended next steps without executing repairs?
11. Does runtime use `symptom_routes.yaml` and SQLite/FTS retrieval, rather than only hard-coded Go routes and all-card scans?
12. After `index rebuild`, does `select id,title from cards_fts ...` return real non-null IDs/titles, and does `whyMatched` only claim FTS for those IDs?
13. Are there any stale binary, cloud-platform, package-integrity, or harness defects?

## Failure Classification

Use these labels:

```text
product failure
harness failure
cloud environment limitation
stale binary contamination
inconclusive
documentation/package issue
```

Do not collapse cloud environment limitations into product failure unless the package explicitly claims that surface works in cloud.

## Report Format

Return:

```text
AgentLink v0.5 Codex Phases GPT Pro Audit Result

Package:
SHA256:
Host:
Go:
Built binaries:

Verdict:
- phase audit: PASS | FAIL | INCOMPLETE
- full regression: PASS | FAIL | NOT RUN | INCONCLUSIVE

Suite results:
- checksum:
- build agentlink:
- build agentlink-rescue:
- phase1 index/retrieve:
- phase2 collect/features:
- phase3 gate/report:
- targeted go tests:
- go vet:
- full go test:

Findings:
- product failures:
- harness failures:
- cloud limitations:
- stale binary risks:
- documentation/package issues:

Safety:
- sudo executed? no
- real network mutation executed? no
- model/runtime/archive files present? no
- secrets found? no

Recommendation:
- accept for v0.5 phase review | fix required | rerun required
```
