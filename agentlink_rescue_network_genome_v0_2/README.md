# AgentLink Rescue Network Genome Pack v0.2

This pack completes the requested v0.2A and v0.2B step.

## What changed from v0.1

- **v0.2A expansion:** 34 seed cards expanded to **300 structured failure cards**.
- **v0.2B topology:** cards compiled into a diagnosis graph, symptom routes, layer DAG, decision tables, indexes, and Codex handoff prompt.
- Added read-only macOS collectors, proxy collector, AoIP manual checklist, runtime contracts, verifier library, rollback templates, SQLite index, and QA report.

## Product stance

The pack is not an AI-written encyclopedia for a small model to read verbatim. It is a build-time failure genome that v0.5 should compile into:

```text
snapshot -> feature extraction -> symptom route -> card retrieval -> discriminator plan -> verifier -> human-confirmed recipe/report
```

## Safety stance

- Read-only collection can be automatic.
- Low/medium local repairs require explicit human confirmation and rollback.
- Dante/AES67/RAVENNA/PTP/switch/QoS/VLAN are **manual-only / never-auto**.
- MDM, EDR, corporate VPN, and policy-locked settings are report/escalation paths, not bypass paths.

## Key files

- `cards/failure_cards.yaml` — full 300-card corpus.
- `topology/failure_graph.yaml` — v0.2B graph.
- `topology/diagnosis_dag.yaml` — runtime decision DAG.
- `ontology/symptom_routes.yaml` — initial symptom routing.
- `indexes/*.json` — lightweight retrieval indexes.
- `database/network_genome_v0_2.sqlite` — queryable card/index database.
- `compiler/codex_handoff_prompt.md` — direct prompt for Codex implementation.
- `collectors/macos_network_snapshot_v0_2.sh` — read-only collector.
- `qa/QA_REPORT.md` — validation summary.
