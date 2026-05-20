# AGENTS.md — AgentLink Rescue v0.5

## Product mission

AgentLink Rescue v0.5 is a local-first macOS network rescue runtime.

It compiles `AgentLink Rescue Network Genome Pack v0.2` into a deterministic diagnostic system:

```text
snapshot -> redaction -> feature extraction -> symptom routing -> card retrieval -> discriminator plan -> verifier -> human-confirmed recipe/report
```

This is not a chatbot and not a prose encyclopedia. Treat the genome pack as a structured failure graph and runtime corpus.

The product goal is not to “solve every possible network problem.” The product goal is to handle the highest-value class of observable, classifiable, reversible local macOS network failures, especially:

- macOS network service/interface/IP/DHCP/route/DNS problems
- system proxy / shell proxy / PAC / app-specific proxy conflicts
- VPN / Network Extension / TUN / Clash Verge / mihomo failures
- developer connectivity failures affecting Codex, Git, npm, Homebrew, Docker, curl, Ollama, local agents
- Dante / RAVENNA / AES67 / PTP / multicast / AoIP diagnostic workflows

The runtime must make network rescue safer, more deterministic, and easier to audit.

---

## Source of truth

The authoritative corpus is located at:

```text
agentlink_rescue_network_genome_v0_2/
```

Important files:

```text
agentlink_rescue_network_genome_v0_2/README.md
agentlink_rescue_network_genome_v0_2/cards/failure_cards.yaml
agentlink_rescue_network_genome_v0_2/cards/failure_cards.json
agentlink_rescue_network_genome_v0_2/cards/failure_cards_compact.json
agentlink_rescue_network_genome_v0_2/schema/failure_card.schema.yaml
agentlink_rescue_network_genome_v0_2/topology/failure_graph.yaml
agentlink_rescue_network_genome_v0_2/topology/diagnosis_dag.yaml
agentlink_rescue_network_genome_v0_2/ontology/layers.yaml
agentlink_rescue_network_genome_v0_2/ontology/symptom_routes.yaml
agentlink_rescue_network_genome_v0_2/runtime/diagnosis_runtime_spec.md
agentlink_rescue_network_genome_v0_2/runtime/data_contracts.yaml
agentlink_rescue_network_genome_v0_2/collectors/macos_network_snapshot_v0_2.sh
agentlink_rescue_network_genome_v0_2/collectors/proxy_snapshot_v0_2.sh
agentlink_rescue_network_genome_v0_2/collectors/aoip_manual_snapshot_checklist.md
agentlink_rescue_network_genome_v0_2/recipes/recipe_contracts.md
agentlink_rescue_network_genome_v0_2/verifiers/verifier_library.md
agentlink_rescue_network_genome_v0_2/rollbacks/rollback_templates.md
agentlink_rescue_network_genome_v0_2/qa/QA_REPORT.md
agentlink_rescue_network_genome_v0_2/compiler/codex_handoff_prompt.md
```

Do not rewrite the corpus as long prose.
Do not invent fixes outside the cards, recipes, verifier contracts, and runtime spec.
Do not silently weaken safety gates.

When implementation details conflict, prioritize in this order:

1. Safety gates in this `AGENTS.md`
2. `runtime/diagnosis_runtime_spec.md`
3. `runtime/data_contracts.yaml`
4. `recipes/recipe_contracts.md`
5. `verifiers/verifier_library.md`
6. `rollbacks/rollback_templates.md`
7. `cards/failure_cards.yaml` / `.json`
8. Convenience or UX preferences

---

## Core implementation principle

Build-time expansion, runtime compression.

At build time:

- Parse and index the large genome corpus.
- Validate schemas.
- Build SQLite / FTS / graph indexes.
- Preserve card IDs, layers, symptoms, observations, discriminators, likely causes, safe checks, recipes, verifiers, rollbacks, risk classes, false positives, related cards, and source anchors.

At runtime:

- Show the user only the smallest safe next step.
- Default to top 3 hypotheses, not an exhaustive dump.
- Prefer discriminating checks over generic suggestions.
- Every diagnosis must be backed by failure card IDs.
- Every proposed repair must pass recipe gate.

Runtime output should usually include:

```text
- top 3 hypotheses
- why each matched
- observations used
- what would confirm/falsify each hypothesis
- next safest read-only check
- recipe gate result
- verifier
- rollback, if mutation is proposed
```

---

## Expected CLI first

Implement CLI before GUI.

Expected commands:

```bash
agentlink-rescue collect --out ./snapshot
agentlink-rescue diagnose --symptom "browser works but codex fails" --snapshot ./snapshot
agentlink-rescue diagnose --symptom "clash tun broke dns" --snapshot ./snapshot
agentlink-rescue diagnose --symptom "vpn loses lan" --snapshot ./snapshot
agentlink-rescue diagnose --symptom "dante devices invisible" --snapshot ./snapshot
agentlink-rescue explain --card MAC-PROXY-002
agentlink-rescue report --format markdown --snapshot ./snapshot
agentlink-rescue verify --recipe <recipe_id> --snapshot ./snapshot
```

If the repo already has an app/runtime architecture, integrate with it.
If no runtime exists, create the smallest maintainable CLI package.

Acceptable first implementation stacks:

- Python CLI, if fastest and testable
- TypeScript CLI, if repo is already TS-heavy
- Swift CLI / SwiftUI later, if repo is already native macOS-heavy

Do not overbuild GUI before corpus loading, retrieval, diagnosis, recipe gating, and reports work.

---

## Runtime pipeline

The runtime pipeline must be:

```text
collector
  -> redactor
  -> feature extractor
  -> symptom router
  -> card retriever/ranker
  -> discriminator planner
  -> verifier metadata
  -> recipe gate
  -> report generator
```

### 1. Collector

Collectors must be read-only by default.

Collect macOS network state relevant to diagnosis:

```text
- macOS version
- network services
- service order
- active interfaces
- interface/link state
- IP/DHCP summary
- route table and default route
- DNS state
- system proxy state
- shell proxy environment
- launchctl proxy-like environment hints if safe
- VPN / utun hints
- Network Extension hints where observable without mutation
- Clash Verge / mihomo process/config hints where safe
- Git proxy config
- npm proxy/registry config
- Homebrew proxy environment hints
- Docker proxy config hints
- curl connectivity hints where safe
- AoIP/Dante/AES67 requested-domain checklist, manual-only
```

Do not collect secrets.
Do not mutate network settings during collection.

### 2. Redactor

The redactor must remove or mask:

```text
- API keys
- bearer tokens
- OAuth tokens
- passwords
- proxy credentials
- VPN secrets
- Wi-Fi passwords
- private keys
- auth headers
- long unique IDs when not diagnostically needed
```

Optional privacy mode should also redact:

```text
- SSIDs
- hostnames
- usernames
- local paths
- private IP details beyond coarse subnet shape
```

Never send unredacted snapshots to cloud.
Never store unredacted snapshots unless the user explicitly requests an unredacted diagnostic archive.

### 3. Feature extractor

Convert raw/redacted snapshot files into structured observations.

Useful features include:

```text
- system proxy enabled
- system proxy points to localhost
- system proxy points to a dead local port
- shell HTTP_PROXY/HTTPS_PROXY/ALL_PROXY present
- shell proxy conflicts with system proxy
- NO_PROXY bypass may affect target
- PAC configured
- DNS resolver mismatch
- DNS resolution fails while IP reachability works
- default route through utun
- VPN/utun present
- LAN route absent while VPN owns default route
- Clash/Verge/mihomo hints present
- TUN/DNS/route risk present
- Git proxy override present
- npm proxy/registry override present
- Homebrew proxy environment present
- Docker host/container proxy mismatch risk
- TLS/certificate trust issue hints
- Dante/AES67 symptom domain detected, manual-only
```

### 4. Symptom router

Use:

```text
ontology/symptom_routes.yaml
```

Route common symptoms before retrieval:

```text
- no internet
- browser works but terminal fails
- Codex/API fails
- Git/npm/brew/Docker cannot connect
- Clash/Verge TUN broke DNS
- VPN loses LAN
- DNS fails but IP works
- localhost proxy dead
- Dante devices invisible
- Dante visible but no audio
- AES67/RAVENNA clock/media issue
```

### 5. Card retrieval and ranking

Retrieve cards by combining:

```text
- user symptom text
- extracted observations
- symptom route prior
- FTS match
- layer/domain prior
- card tags
- discriminator matches
```

Ranking should roughly follow:

```text
score =
  symptom_match
  + observation_match
  + discriminator_match * 2
  + domain_route_prior
  - risk_penalty
  - lower_layer_uncertainty_penalty
```

Return top 3 hypotheses by default.
Always cite card IDs.
Never invent a cause without citing a card.

### 6. Discriminator planner

For each top hypothesis, output:

```text
- why it matched
- observations used
- what would confirm it
- what would falsify it
- next safest read-only check
```

The v0.5 CLI should work deterministically without requiring an LLM.
A local model may be added later for explanation/planning, but it must not override hard safety gates.

### 7. Recipe gate

All repair suggestions must pass recipe gate.

Gate outputs:

```text
read_only_allowed
human_confirmed_allowed
manual_only
report_only
blocked
```

Hard blocking rules:

```text
- no verifier => blocked
- mutating action without rollback => blocked
- unknown mutation => blocked
- risk `never_auto` => manual_only or blocked
- L10 multicast/discovery, L11 PTP/clock, L12 AoIP media => manual_only unless explicitly read-only
- MDM/EDR/corporate policy issue => report_only
- TLS verification bypass => blocked as durable repair
- global pf flush => blocked
- switch/VLAN/QoS/IGMP/PTP leader changes => manual_only/report_only
- Dante/AES67/RAVENNA clock/media changes => manual_only/report_only
```

Read-only collectors and checks may run automatically.
Mutating repairs require explicit human confirmation.
No mutating repair may be shown as safe unless it has both verifier and rollback.

### 8. Report generator

Generate both Markdown and compact JSON reports.

Report should include:

```text
- timestamp
- tool version
- corpus version
- snapshot summary
- redaction status
- user symptom
- extracted observations
- top hypotheses
- card IDs
- why each matched
- checks already run
- suggested next checks
- recipe gate result
- allowed recipes
- verifier
- rollback
- blocked actions and reason
- manual-only domains and reason
```

---

## Safety gates

These rules are non-negotiable:

```text
- Read-only collectors may run automatically.
- Mutating repairs require explicit human confirmation.
- No recipe without verifier.
- No mutating recipe without rollback.
- Dante/AES67/RAVENNA/PTP/switch/QoS/VLAN changes are manual-only or never-auto.
- MDM, EDR, corporate VPN, and policy-locked profiles are report-only, not bypass paths.
- Never auto-bypass TLS verification as a durable fix.
- Never flush all pf rules automatically.
- Never send unredacted snapshots to cloud.
- Never store API keys, tokens, Wi-Fi passwords, VPN secrets, private IP topology, or hostnames without redaction unless the user explicitly exports an unredacted diagnostic archive.
```

Do not add “temporary convenience” exceptions to these gates.

---

## Implementation expectations

Prefer small, testable modules.

Suggested module boundaries:

```text
corpus_loader
schema_validator
index_builder
snapshot_collector
redactor
feature_extractor
symptom_router
retriever
ranker
discriminator_planner
recipe_gate
report_generator
cli
```

Expected tests:

```text
- corpus validation
- loading 300 cards
- SQLite/FTS search
- symptom routing
- redaction
- feature extraction from synthetic snapshots
- diagnosis ranking
- recipe gate
- Markdown report generation
- JSON report generation
```

Synthetic diagnosis cases:

```text
- browser works but CLI fails due stale shell proxy
- system proxy points to dead localhost port
- VPN captures default route and LAN disappears
- Clash/Verge TUN DNS issue
- Codex/API fails while browser works
- Git proxy stale
- npm proxy or registry stale
- Docker host/container proxy mismatch
- Dante devices invisible due wrong NIC/manual-only
- AES67/PTP issue/manual-only
```

---

## Do not do

Do not:

```text
- rewrite the genome pack into prose
- invent fixes not grounded in cards/contracts
- weaken safety gates
- auto-mutate network settings in first implementation
- auto-edit Dante/AES67/RAVENNA/PTP/switch/QoS/VLAN settings
- bypass TLS verification as a durable fix
- collect secrets
- send snapshots to cloud
- build GUI before CLI works
- hide blocked actions from the report
```

---

## Definition of done for v0.5 CLI

A first acceptable v0.5 CLI must satisfy:

```text
- `agentlink-rescue collect` creates a redacted snapshot.
- `agentlink-rescue diagnose` returns top 3 card-backed hypotheses.
- Every diagnosis cites card IDs.
- Every proposed repair is passed through recipe gate.
- Manual-only domains stay manual-only.
- Markdown and JSON reports are generated.
- Tests pass.
```

When finishing a task, report:

```text
- files changed
- commands run
- test results
- remaining gaps
```
