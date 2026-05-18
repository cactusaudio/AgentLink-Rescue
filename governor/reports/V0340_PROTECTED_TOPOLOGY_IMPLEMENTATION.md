# v0.3.4 Protected Topology Implementation

## Verdict

Implemented as a review diff. This is not a release/tag/push.

The GPT Pro omission finding was valid: the m13 hotfix was too dependent on
`DiagnosticReport.Raw` and modeled protected topology as a failure class instead
of a repair boundary. v0.3.4 moves the boundary into structured topology and
repair-corridor enforcement.

## Product Changes

- Added `DiagnosticReport.topology`.
- Added protected constraints, root-cause candidates, red herrings, and
  `repairCorridor`.
- Added manifest-based live/default topology intake so protection does not
  require verbose Raw.
- Bound Raw topology evidence to the same interface before it can classify a
  protected trap.
- Added AES67/RAVENNA and adjacent protected topology taxonomy.
- Suppressed TUN repair when protected route ownership conflict exists.
- Forced repair/orchestrator/model arbitration to respect deterministic repair
  corridor.
- Extended diagnosis graph with topology/corridor fields.
- Added GUI safety copy for protected topologies.
- Expanded sandbox maze pack from 13 to 25 scenarios.

## Evidence

Focused tests:

```text
go test ./internal/diagnose ./internal/classify ./internal/diagnosisgraph ./internal/repair ./internal/orchestrator -count=1
PASS
```

Sandbox mazes:

```text
bash scripts/run_sandbox_mazes.sh
PASS 25/25
```

Key regression probes:

```text
m13 without Raw -> primaryClass PROTECTED_AUDIO_VLAN_ROUTE_TRAP
m14 AES67-only -> PROTECTED_AUDIO_VLAN_ROUTE_TRAP
m25 unbound raw audio -> DNS_FAIL, not protected topology
```

Full gate:

```text
go test ./... -count=1 -timeout 30m
PASS
internal/chaos 766.711s

go vet ./...
PASS
```

Hygiene:

```text
ports 18080/18081: no listeners
tracked forbidden artifacts: none
ignored pre-existing artifacts: dist/ and gui/CactusAgentLinkRescue/.build/
```

## Truth Boundary

This proves fixture/read-only/dry-run protected topology safety. It does not
prove real macOS mutation, switch/VLAN changes, signed distribution, or model
runtime behavior. Those remain separate gates.

The current repair corridor is deliberately conservative: when protected
constraints are active, AgentLink refuses mutation and recommends read-only
snapshots/support bundle/incident report. Scoped non-protected repair can be a
future feature only after a separate verifier proves the action cannot touch
protected services/interfaces.
