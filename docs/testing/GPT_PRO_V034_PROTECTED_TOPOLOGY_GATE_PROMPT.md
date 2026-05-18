# GPT Pro Prompt - AgentLink v0.3.4 Protected Topology Gate

You are an independent pre-release auditor for Cactus AgentLink Rescue.

This package is not asking you to prove real macOS network mutation. It is asking
you to verify the v0.3.4 protected-topology safety gate in a cloud sandbox using
only fixture, read-only, dry-run, and Go-test surfaces.

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

Allowed:

```text
shasum -a 256 -c CHECKSUMS.txt
go build -o .gpt-pro-build/agentlink-linux-amd64 ./cmd/agentlink
go test ...
go vet ...
agentlink version
agentlink manifest
agentlink chaos list
agentlink chaos run --fixture ...
agentlink diagnose-graph --from ...
agentlink recipe list
agentlink recipe inspect ...
agentlink recipe run ... --dry-run
bash scripts/run_sandbox_mazes.sh
FULL=1 bash scripts/gpt_pro_stress_test.sh
read and analyze source/docs/fixtures/reports
create proposed fixtures or patch sketches only in your output directory
```

On Linux/non-macOS, `unsupported platform` is a failure for fixture/read-only/dry-run
surfaces and expected only for real mutation surfaces.

## v0.3.4 Claim To Verify

AgentLink should treat protected topology as a deterministic repair boundary,
not merely as a Raw-text failure class.

The protected boundary should survive:

1. no `DiagnosticReport.Raw`;
2. AES67/RAVENNA-only audio evidence;
3. adjacent protected topologies beyond Dante;
4. unbound Raw audio text that must NOT false-positive;
5. TUN/fake-IP red herring when route ownership is protected;
6. model/advisory/graph surfaces trying to route toward broad repair;
7. normal TUN-owned failure when no protected topology exists.

## Required Commands

Run from extracted package root:

```bash
shasum -a 256 -c CHECKSUMS.txt
go build -o .gpt-pro-build/agentlink-linux-amd64 ./cmd/agentlink
.gpt-pro-build/agentlink-linux-amd64 version
FULL=1 OUT="$PWD/gpt-pro-results/v034-full" bash scripts/gpt_pro_stress_test.sh
go test ./internal/diagnose ./internal/classify ./internal/diagnosisgraph ./internal/repair ./internal/orchestrator -count=1
go test ./internal/chaos -run TestChaosFixturesPass -count=1
go vet ./internal/diagnose ./internal/classify ./internal/diagnosisgraph ./internal/repair ./internal/orchestrator ./internal/toolmanifest
```

Then manually inspect at least these fixtures:

```text
testdata/chaos/network/mazes/m13-dante-vlan-no-internet-trap.json
testdata/chaos/network/mazes/m14-aes67-audio-vlan-route-trap.json
testdata/chaos/network/mazes/m15-ndi-video-vlan-stale-default.json
testdata/chaos/network/mazes/m22-mdm-scoped-dns-proxy.json
testdata/chaos/network/mazes/m24-tun-owned-no-protected-topology.json
testdata/chaos/network/mazes/m25-unbound-raw-audio-false-positive.json
```

For m13, create a temporary copy of the embedded diagnostic JSON with `raw`
removed, run:

```bash
.gpt-pro-build/agentlink-linux-amd64 diagnose-graph --from <tmp-diagnostic-without-raw.json> --json
```

Expected:

```text
primaryClass = PROTECTED_AUDIO_VLAN_ROUTE_TRAP
repairCorridor.mutationAllowed = false
recommendedNextTools are read-only/report tools
no network_baseline_reset / remove_tun_residue / clean_stale_proxy_baseline recommendation
```

For m25, expected:

```text
failureClasses include DNS_FAIL
failureClasses do NOT include PROTECTED_AUDIO_VLAN_ROUTE_TRAP
failureClasses do NOT include PROTECTED_TOPOLOGY_CONSTRAINT
```

For m24, expected:

```text
failureClasses include CLASH_TUN_ACTIVE_OR_STALE
recommendedRepair = tun
recommendedRepairLevel = tun
```

## Source Areas To Inspect

```text
internal/diagnose/types.go
internal/diagnose/topology.go
internal/classify/classify.go
internal/diagnose/tun.go
internal/diagnosisgraph/graph.go
internal/repair/repair.go
internal/orchestrator/orchestrator.go
internal/toolmanifest/manifest.go
docs/offline/PROTECTED_TOPOLOGY_SAFETY.md
governor/reports/V0340_PROTECTED_TOPOLOGY_IMPLEMENTATION.md
docs/testing/AGENTLINK_SANDBOX_MAZES.md
```

## Required Output Sections

Return one Markdown report and one machine-readable JSON summary with:

```text
Verdict
Package / Host
Commands Run
Suite Results
v0.3.4 Protected Topology Claim Assessment
m13 Without Raw Result
AES67 / Adjacent Topology Result
Unbound Raw False-Positive Result
TUN Control Case Result
Repair Corridor / Graph Assessment
Model-Facing Safety Assessment
Remaining Unsafe Gaps
False-Positive Risks
False-Negative Risks
Proposed Additional Fixtures
Safety / Mutation Audit
Final Recommendation
```

Use these verdict classes:

```text
PASS_FOR_CLOUD_FIXTURE_READONLY_DRYRUN_GATE
FAIL_HOLD_FOR_PROTECTED_TOPOLOGY_SAFETY
INCONCLUSIVE_CLOUD_LIMITATION
HARNESS_FAILURE
PACKAGE_FAILURE
```

Do not inflate a cloud fixture pass into real macOS repair proof.
