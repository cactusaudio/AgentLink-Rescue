# Codex handoff prompt — AgentLink Rescue v0.5 from Network Genome v0.2

You are implementing AgentLink Rescue v0.5. Do not rewrite the knowledge base as prose. Compile it.

## Inputs

- `cards/failure_cards.yaml` — 300 failure cards.
- `topology/failure_graph.yaml` — layer/card/observation/verifier graph.
- `topology/diagnosis_dag.yaml` — runtime decision DAG.
- `ontology/symptom_routes.yaml` — initial symptom routing.
- `collectors/*` — read-only snapshot collectors.
- `runtime/data_contracts.yaml` — IO contracts.

## Build tasks

1. Validate all cards against `schema/failure_card.schema.yaml`.
2. Build a SQLite database with tables: cards, tags, layers, edges, observations, verifiers, source_anchors.
3. Add FTS over id/title/symptoms/discriminators/likely_causes/tags.
4. Implement read-only collector runner with redaction.
5. Implement symptom router and card retriever.
6. Implement planner prompt that forbids invented fixes and cites card IDs.
7. Implement recipe gate:
   - no verifier => block;
   - no rollback for mutation => block;
   - L10/L11/L12 => manual-only;
   - policy/MDM => report-only.
8. Implement CLI first, GUI second.

## CLI target

```bash
agentlink-rescue diagnose --symptom "browser works but codex fails" --snapshot ./snapshot
agentlink-rescue collect --out ./snapshot
agentlink-rescue explain --card MAC-PROXY-002
agentlink-rescue report --format markdown
```

## Minimal UI target

- Start Guided Rescue
- Show observations collected
- Show top 3 hypotheses with card IDs
- Show next safest check
- Show copyable human-confirmed command only when allowed
- Show rollback and verifier side-by-side

## Hard constraints

- Never auto-mutate Dante/AES67/RAVENNA/switch/PTP/QoS/VLAN.
- Never auto-bypass TLS verification as a durable fix.
- Never flush all pf rules automatically.
- Never edit MDM-managed profiles.
- Never send unredacted snapshot to cloud.
