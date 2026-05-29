# AgentLink v0.5.2 Release-Readiness Hardening

Date: 2026-05-20
Updated: 2026-05-24 post-alpha hardening and portable-beta packaging

Status: implementation in review, not pushed.

## Objective

Close the release blockers identified after v0.5.1 phase-review alpha:

- native/long-lived SQLite access instead of Python subprocess SQLite;
- real macOS read-only collector dogfood;
- GUI proof;
- clean release packaging;
- signing/notarization truth;
- disposable mutation + rollback proof;
- removal of hard-coded `scenarioBoost` ranking in favor of corpus route metadata.

## Implemented

### Native SQLite / FTS

`internal/genome` now uses Go `database/sql` with `modernc.org/sqlite` for:

- rebuilding `network_genome_v0_2.sqlite` from `cards/failure_cards.json`;
- inspecting `cards` / `cards_fts` stats;
- querying FTS candidate IDs.

The runtime no longer shells out to `python3` for genome SQLite rebuild, stats,
or retrieval. Regression coverage includes a source-level guard that rejects
`exec.Command("python3"` in `internal/genome/genome.go`.

Targeted proof:

```text
go test ./internal/genome -count=1
ok cactus-agentlink-rescue/internal/genome
```

### Corpus-Driven Ranking Boosts

The hard-coded `scenarioBoost()` function was removed. Route-specific ranking
boosts now live in:

```text
agentlink_rescue_network_genome_v0_2/ontology/symptom_routes.yaml
```

Diagnosis output preserves provenance with `route <id> corpus ranking boost`
entries. The browser/Codex real macOS dogfood produced:

```text
CODEX-API-001
DEV-CURL-001
MAC-PROXY-002
```

All three carried both SQLite/FTS provenance and corpus-ranking provenance.

### Real macOS Read-Only Collector Dogfood

Command surface:

```text
agentlink-rescue collect --privacy strict
agentlink-rescue diagnose --symptom "browser works but codex fails" --snapshot <snapshot>
agentlink-rescue report --format markdown/json
```

Evidence directory:

```text
.tmp/v052/mac-dogfood-20260520T145000Z
```

Result:

- strict snapshot wrote `privacyMode: strict`;
- macOS read-only collectors completed;
- no `sudo`, network mutation, rescue `--yes`, route mutation, `networksetup`
  mutation, `ifconfig` mutation, `launchctl` mutation, or Keychain extraction;
- scan found no API-key, GitHub, Slack, AWS, `/Users/jack`, `/Users/bowei`,
  10/8, 192.168/16, 172.16/12, 169.254/16, or 198.18/15 leakage in snapshot or
  reports.

Strict redaction was additionally tightened for link-local `169.254/16` and
fake-IP `198.18/15` ranges after the first macOS dogfood exposed them.

The first full audit rerun also caught a strict-redaction JSON bug: the
`/Users/...` path regex could consume a closing quote in JSON report paths.
That was fixed at the redaction source and covered by a JSON-parse regression
test.

### GUI And Package Proof

Core package:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-core.zip
sha256 f62193d218bef2c3be955b6cc6175f179dadf2d175df4682eaf5373fab935245
```

Core GUI package:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-core-gui.zip
sha256 331ab5f0e3e31cf46049dff22ee9f857220160801014b8376830c6ecd55097d9
```

Both zips passed `unzip -tq`. Package scan found no model weights, runtime
binaries, archive payloads, `node_modules`, `.build` payloads, or files over
the allowed package surface.

GUI proof:

```text
bash scripts/dogfood_gui_core.sh
dogfood GUI core OK
```

Full audit proof:

```text
FULL=1 OUT=gpt-pro-results/v0520-release-readiness bash scripts/gpt_pro_v0501_audit.sh
status: completed
full: true
```

Audit stats:

```text
cards=300
layers=15
ftsRows=300
ftsIdsUsable=true
ftsNullIds=0
graphEdges=3427
```

Full regression inside the audit:

```text
go test ./... -count=1 -timeout 35m
internal/chaos 828.979s
all packages pass
```

### Disposable Mutation And Rollback

Mutation proof remains scoped to temporary homes and disposable fixtures.

Passed:

```text
bash scripts/dogfood_guided_rescue.sh
bash scripts/dogfood_temp_home.sh
bash scripts/dogfood_autorollback.sh
bash scripts/dogfood_tun_repair_dryrun.sh
bash scripts/dogfood_clean_baseline_last_resort.sh
```

The temp-home runs executed `--yes` only against disposable `$HOME` values,
verified restore to the pre-change absent `.zshrc` state, and confirmed fake
secrets did not enter reports.

The live Mac currently triggers protected topology boundaries for TUN and
clean-baseline dry runs. The dogfood scripts now accept that as the correct
product behavior only when the report is `protected_topology_report_only`, has
no actions, and includes a protected-topology refusal warning.

### Signing / Notarization

Local identity scan:

```text
security find-identity -v -p codesigning
     0 valid identities found
```

Signing and notarization remain a credentials and distribution boundary. The
current proof is unsigned local package integrity plus GUI selftest, not public
release readiness.

## Remaining Release Boundaries

Still not proven by this hardening pass:

- notarized public distribution;
- real macOS mutating network repair on disposable hardware/network fixtures;
- real GUI manual visual QA beyond selftest;
- Gemma/Qwen model-runtime behavior;
- third-party external audit of this exact new package.

## 2026-05-24 Hardening Addendum

Post-alpha commit:

```text
01ced16 Harden AgentLink field beta
```

Scope:

- private diagnostic/session/report/journal/snapshot-style files tightened to
  `0600`;
- private AgentLink-owned directories tightened to `0700`;
- user config-file patching preserves existing file modes and defaults new
  files to `0600`;
- support-bundle zipping skips symlinks and non-regular files;
- dead code removed;
- fuzz harnesses added for redaction, planner decisions, recipe decoding, and
  diagnosis-graph JSON build input;
- accidental root-level `agentlink` build artifact ignored.

Verification rerun:

```text
go test ./... -count=1 -timeout 35m
go vet ./...
staticcheck ./...
```

Portable beta:

```text
/Volumes/CTS Dark/Cactus-AgentLink-Rescue-v0.5.2-field-beta-hardened-portable-universal-20260524T012027Z.zip
SHA256 e9347c39c50c42920240cadca55f3a28f49079561822604919f8dda81caa17eb
```

The portable bundle contains a universal app, fallback launcher, manifest,
checksum verifier, and no model/runtime/DMG payload. It is ad-hoc signed and
verified locally, not notarized.

## Truth Notes

- The protected-topology TUN/clean-baseline dogfood changes are harness truth
  repairs. They do not weaken product safety; they require stricter report-only
  refusal when a protected topology is detected.
- SQLite performance is now in-process Go SQLite. It is no longer a Python
  subprocess architecture.
- Ranking is not purely emergent. It is explicitly corpus-encoded in
  `symptom_routes.yaml`, which is the intended auditable control surface.
