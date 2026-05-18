# Read This First - GPT Pro Protected Topology Research

This package is not a release gate. It is an independent research/audit pack for finding AgentLink Rescue omissions around protected or intentionally specialized network topologies.

Read this first, then read:

```text
docs/testing/GPT_PRO_V034_PROTECTED_TOPOLOGY_GATE_PROMPT.md
```

## Hard Safety Rules

Do not run:

```text
sudo
agentlink rescue --yes
agentlink guided rescue --yes
real networksetup mutation
real route mutation
real ifconfig mutation
real launchctl mutation
rm -rf outside a temporary sandbox
```

Allowed actions:

```text
go test ...
go vet ...
agentlink version
agentlink selftest
agentlink manifest
agentlink chaos list
agentlink chaos run --fixture ...
agentlink diagnose-graph --from ...
agentlink recipe list
agentlink recipe inspect ...
agentlink recipe run ... --dry-run
bash scripts/run_sandbox_mazes.sh
bash scripts/gpt_pro_stress_test.sh
read and analyze source/docs/fixtures/reports
create proposed fixtures or patch sketches in a temporary output directory
```

This package is for research and fixture/read-only/dry-run validation only. Do not claim real macOS repair, GUI behavior, installer behavior, signing/notarization, or model behavior from this cloud run.

## Research Target

The motivating incident is a protected Dante/MTRX audio VLAN route-selection trap:

- a Mac appears offline;
- default route points to a protected Dante audio VLAN interface;
- another interface is the intended management/internet path;
- TUN/fake-IP evidence is a red herring;
- broad DHCP renew / route flush / TUN cleanup / proxy reset / switch VLAN mutation can make the environment dirtier or break audio topology.

Your task is to verify the v0.3.4 topology-safe diagnostic/repair model and still search for omissions that could dirty specialized networks while trying to restore ordinary internet access.

## Deliverables

Return:

```text
1. PASS/FAIL/INCONCLUSIVE for the research task itself.
2. Top omitted protected-topology classes.
3. False-positive risks.
4. False-negative risks.
5. Proposed structured topology schema.
6. Classifier precedence and repair-boundary rules.
7. At least 20 proposed new fixture scenarios.
8. Top 10 highest-priority product changes.
9. Tests that should be added before release.
10. Clear distinction between deterministic executor requirements and model/advisory-only behavior.
```
