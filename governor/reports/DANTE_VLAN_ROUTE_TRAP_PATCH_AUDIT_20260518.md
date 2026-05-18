# Dante/VLAN Route Trap Patch Audit - 2026-05-18

## Verdict

The memo's product concern is valid: AgentLink must not treat an intentionally protected Dante/MTRX audio VLAN interface as an ordinary broken internet path.

The initial patch added the right failure class and m13 fixture, but it had two safety gaps:

1. `diagnosisgraph.Build` selected the first non-OK class as primary. In the m13 fixture, `DNS_FAIL` can appear before `PROTECTED_AUDIO_VLAN_ROUTE_TRAP`, which would route the model-facing graph toward `agentlink.network_baseline_reset`.
2. `repair.LevelSafe` renews DHCP on all hardware ports. If a protected trap is classified as `safe`, a user-approved safe rescue could still DHCP-renew the protected Dante interface.

Both gaps are now closed.

## Product Behavior Now Locked

- `PROTECTED_AUDIO_VLAN_ROUTE_TRAP` is a first-class class.
- TUN repair is suppressed when protected audio route evidence is present.
- Diagnosis graph promotes `PROTECTED_AUDIO_VLAN_ROUTE_TRAP` above secondary classes such as `DNS_FAIL`.
- Diagnosis graph recommends only:
  - `agentlink.route_snapshot`
  - `agentlink.network_snapshot`
  - `agentlink.incident_report`
- Diagnosis graph does not recommend TUN residue removal, network baseline reset, or stale proxy cleanup for this class.
- `repair.Run` refuses mutating rescue when this class is present.
- `buildActions` returns no mutating action plan for this class at any rescue level.
- `orchestrator.Run` stops at manual action before Gemma/model supervision can promote the case into repair.

## Fixture

```text
testdata/chaos/network/mazes/m13-dante-vlan-no-internet-trap.json
```

Expected:

```text
failureClass: PROTECTED_AUDIO_VLAN_ROUTE_TRAP
recommendedRepairLevel: safe
```

Observed:

```text
failureClasses: DNS_FAIL, PROTECTED_AUDIO_VLAN_ROUTE_TRAP
recommendedRepairLevel: safe
status: passed
```

The secondary `DNS_FAIL` is allowed as a symptom. It no longer owns graph routing.

## Verification

```text
go test ./internal/classify ./internal/diagnose ./internal/diagnosisgraph ./internal/repair ./internal/orchestrator -count=1
agentlink chaos run --fixture testdata/chaos/network/mazes/m13-dante-vlan-no-internet-trap.json --json
REPORT_PATH=/tmp/agentlink-dante-vlan-sandbox-report.json bash scripts/run_sandbox_mazes.sh
agentlink diagnose-graph --from /tmp/m13-diagnostic.json --json
```

Observed:

```text
targeted Go tests: pass
m13 fixture: passed
sandbox mazes: 13/13
diagnosis graph primaryClass: PROTECTED_AUDIO_VLAN_ROUTE_TRAP
diagnosis graph recommended tools: route_snapshot, network_snapshot, incident_report
```

## Remaining Truth Boundary

The current detector relies on explicit protected-interface evidence in `DiagnosticReport.Raw`, which is excellent for fixtures, support-bundle enrichment, and operator-provided context, but is not yet a complete live Mac discovery system.

The mainline product upgrade should promote this evidence into structured fields:

- protected interfaces,
- special-purpose services by interface,
- Dante/RAVENNA/_netaudio/mDNS service families,
- alternate internet path proof,
- forbidden repair actions,
- switch/VLAN provenance when user/admin supplies it.

Until then, AgentLink should treat this as a high-value guardrail when the evidence exists, not as proof that every Dante/VLAN topology can be discovered automatically.

