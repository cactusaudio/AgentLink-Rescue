# AgentLink Rescue v0.5 Codex Phases

## Verdict

Implemented the three Codex phase prompts from:

```text
/Users/jack/Downloads/agentlink_rescue_v0_5_codex_docs/
```

This is a local review diff, not a pushed release.

## Source Intake

- Added the `AgentLink Rescue Network Genome Pack v0.2` corpus under
  `agentlink_rescue_network_genome_v0_2/`.
- Corpus zip source:
  `/Users/jack/Downloads/agentlink_rescue_network_genome_v0_2.zip`
- Zip SHA256:
  `c91ea911cd390638578148df0c37437121b4ed00d5d6e0a17e9db59db37f600b`
- Loaded corpus:
  - cards: `300`
  - layers: `15`
  - SQLite/FTS rows: `300`
  - graph edges: `3427`
  - corpus version: `0.2`

## Phase 1: Corpus Loader + Retriever

Implemented:

```text
agentlink-rescue index rebuild
agentlink-rescue index stats
agentlink-rescue explain --card <CARD_ID>
agentlink-rescue diagnose --symptom "<symptom>" --no-snapshot
```

Behavior:

- Loads and validates all 300 failure cards from
  `cards/failure_cards.json`.
- Validates duplicate IDs, schema-required fields, unknown layers, risk
  classes, and malformed repair contracts.
- Rebuilds a durable SQLite/FTS index from `cards/failure_cards.json` using
  the local Python stdlib `sqlite3` bridge. Diagnosis uses SQLite/FTS candidate
  retrieval first, then deterministic Go ranking over card-backed candidates,
  without adding cgo or external Go dependencies.
- Loads `ontology/symptom_routes.yaml` for symptom route priors before ranking;
  hard-coded routes are fallback only.
- Returns top-3 card-backed hypotheses and cites card IDs.

Verified examples:

```text
agentlink-rescue index stats
agentlink-rescue index rebuild --json
agentlink-rescue explain --card MAC-PROXY-002
agentlink-rescue diagnose --symptom "browser works but codex fails" --no-snapshot
agentlink-rescue diagnose --symptom "clash tun broke dns" --no-snapshot
agentlink-rescue diagnose --symptom "vpn loses lan" --no-snapshot
agentlink-rescue diagnose --symptom "dante devices invisible" --no-snapshot
```

## Phase 2: Read-Only Snapshot + Feature Extraction

Implemented:

```text
agentlink-rescue collect --out <snapshot>
agentlink-rescue collect --out <snapshot> --privacy strict
agentlink-rescue features --snapshot <snapshot>
agentlink-rescue diagnose --symptom "<symptom>" --snapshot <snapshot>
```

Behavior:

- Collector uses read-only commands only.
- Writes:
  - `raw/`
  - `redacted/`
  - `features/`
  - `snapshot_manifest.json`
  - `features/snapshot_features.json`
- Safety correction: `raw/` is redacted by default. A true unredacted local raw
  archive requires explicit `AGENTLINK_ALLOW_UNREDACTED_RAW=1`.
- Docker config collection no longer dumps `~/.docker/config.json`; it emits
  boolean/count metadata only and never includes registry names, auth blobs,
  identity tokens, credential-store values, usernames, or passwords.
- Strict privacy redacts proxy credentials, tokens, private keys, usernames,
  local paths, generic hostnames, and full private IP addresses including
  `10/8`, `172.16/12`, and `192.168/16`.
- Feature extraction now records the snapshot privacy mode correctly. Reports
  inherit strict privacy when the snapshot is strict.
- `NO_PROXY=localhost,127.0.0.1` no longer creates a localhost proxy false
  positive; only effective `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` or system
  proxy fields drive localhost-proxy risk features.
- Feature extractor emits explicit `true` / `false` / `unknown-like string`
  values and avoids aggressive inference.

Verified live snapshot path:

```text
/tmp/agentlink-v05-snapshot
```

It was used only as local read-only evidence and can be regenerated.

## Phase 3: Recipe Gate + Verifier + Reports

Implemented:

```text
agentlink-rescue gate --card <CARD_ID>
agentlink-rescue report --format markdown --snapshot <snapshot> --symptom "<symptom>" --out <report.md>
agentlink-rescue report --format json --snapshot <snapshot> --symptom "<symptom>" --out <report.json>
agentlink-rescue report --format json --snapshot <snapshot> --symptom "<symptom>" --out <report.json> --json
```

Gate results:

```text
read_only_allowed
human_confirmed_allowed
manual_only
report_only
blocked
```

Hard gate behavior:

- no verifier -> `blocked`
- mutating action without rollback -> `blocked`
- TLS verification bypass -> `blocked`
- global pf flush -> `blocked`
- `never_auto` -> `manual_only`
- high-risk network repair -> `manual_only`
- L10/L11/L12 AoIP/multicast/PTP/media -> `manual_only`
- policy/MDM/corporate-owned cases -> `report_only`
- low/medium reversible local repair with verifier + rollback ->
  `human_confirmed_allowed`

Verified examples:

```text
agentlink-rescue gate --card MAC-PROXY-002
agentlink-rescue gate --card DANTE-CLOCK-001
agentlink-rescue report --format markdown --snapshot /tmp/agentlink-v05-snapshot --symptom "browser works but codex fails" --out /tmp/agentlink-v05-report.md
agentlink-rescue report --format json --snapshot /tmp/agentlink-v05-snapshot --symptom "browser works but codex fails" --out /tmp/agentlink-v05-report.json
```

## Safety Boundary

No mutating repair execution was added in these phases.

The implemented surface is:

```text
corpus -> validation -> index/stats -> route -> retrieve/rank -> collect read-only -> redact -> features -> gated diagnosis -> reports
```

The collector does not run:

```text
sudo
networksetup mutation
route mutation
ifconfig mutation
launchctl mutation
pf mutation
VPN mutation
Keychain extraction
Wi-Fi password extraction
```

## Tests

Added tests for:

- all 300 cards loading
- duplicate card ID rejection
- missing required field rejection
- unknown layer rejection
- schema-derived risk and repair contract validation
- index stats
- SQLite/FTS candidate retrieval and corrupt index rebuild
- dynamic `symptom_routes.yaml` route loading
- symptom diagnosis top-3 output
- `MAC-PROXY-002` retrieval for browser/Codex split symptom
- redaction standard + strict, including Docker auth, hostnames, private IPs,
  and user paths
- collector manifest + default raw redaction + Docker config safe summary
- `NO_PROXY` false-positive guard
- synthetic feature extraction
- gate outcomes including human-confirmed, manual-only, blocked
- Markdown and JSON report generation
- CLI command surfaces for index/explain/diagnose/gate/report

Final verification on 2026-05-20:

```text
go test ./internal/cli ./internal/genome -count=1
go vet ./...
go test ./... -count=1 -timeout 35m
FULL=0 OUT=/tmp/agentlink-v0501-audit-privacy bash scripts/gpt_pro_v0500_audit.sh
CLI smoke: index rebuild/stats, explain, collect strict, features, diagnose,
gate, markdown report, JSON report
```

Observed final evidence:

```text
index rebuild: cards=300, ftsRows=300, layers=15
generated sqlite tables include cards, cards_fts, observations, recipes,
verifiers, rollbacks, source_anchors, edges
strict snapshot rawContainsUserPass=false
fake Docker config auth/identitytoken/credsStore/credHelpers not present in
raw, redacted, markdown report, or JSON report
features privacyMode=strict
diagnose snapshot top3: CODEX-API-001, DEV-CURL-001, MAC-PROXY-002
DANTE-CLOCK-001 gate=manual_only
MAC-PROXY-002 gate=human_confirmed_allowed
markdown report exists=true
JSON report exists=true
```

## Known Boundaries

- SQLite rebuild/query uses `python3` + stdlib `sqlite3` instead of a Go SQLite
  dependency. If the local Python SQLite bridge is unavailable, the system
  falls back to all-card scoring rather than failing diagnosis.
- Snapshot collection is local and read-only. It is not proof of real mutation
  repair.
- AoIP/Dante/AES67/PTP/switch-related cases remain manual-only/report-only by
  deterministic gate.
