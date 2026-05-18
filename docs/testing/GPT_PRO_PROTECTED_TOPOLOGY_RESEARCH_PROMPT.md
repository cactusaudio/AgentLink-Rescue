# GPT Pro Prompt - AgentLink Protected Topology Omission Research

You are GPT Pro acting as an independent pre-release research auditor for Cactus AgentLink Rescue.

You are not proving a release gate. You are searching for omissions, unsafe repair assumptions, and topology-model improvements.

## Context

AgentLink Rescue is a deterministic macOS rescue kernel. Its job is to diagnose and safely recover local networking and AI-tooling access without making the machine worse.

Recent stress testing found and fixed a cloud package failure, then a separate real-world omission:

```text
Mac Studio + Cisco SG350-10P + Dante/MTRX Studio network cutover.
Default route still pointed to en0.
en0 could not reach the LAN gateway.
en0 was intentionally being moved to a dedicated Dante audio VLAN.
en1 was the actual internet/management path and was healthy.
Clash TUN/fake-IP evidence was present but was a red herring.
Broad DHCP renew, route flush, TUN cleanup, proxy reset, or switch/VLAN mutation could damage the audio topology.
```

The current patch adds:

```text
PROTECTED_AUDIO_VLAN_ROUTE_TRAP
m13-dante-vlan-no-internet-trap fixture
TUN recommendation suppression for this trap
diagnosis graph read-only routing for this trap
repair/orchestrator mutation refusal for this trap
```

Your job is to evaluate whether this is enough, identify adjacent omitted topology classes, and propose a better topology-aware product design.

## Hard Safety Rules

Do not execute any real system repair or mutation.

Forbidden:

```text
sudo
agentlink rescue --yes
agentlink guided rescue --yes
networksetup mutation
route mutation
ifconfig mutation
launchctl mutation
rm -rf outside a temporary sandbox
real switch/router/VLAN mutation
real proxy app mutation
real Keychain/secret mutation
```

Allowed:

```text
read source/docs/fixtures/reports
go test ...
go vet ...
agentlink chaos list
agentlink chaos run --fixture ...
agentlink diagnose-graph --from ...
agentlink recipe list/inspect
agentlink recipe run ... --dry-run
bash scripts/run_sandbox_mazes.sh
create research output files under your temporary result directory
```

## Critical Product Principle

Protected topology detection is not automatically root-cause diagnosis.

Treat protected topology as a repair boundary condition:

```text
1. Detect protected topology constraints.
2. Freeze forbidden mutation surfaces.
3. Continue diagnosing the actual connectivity root cause.
4. Choose only repairs inside the allowed-safe corridor.
5. If the safe corridor is too narrow, produce support bundle / incident report / terminal ticket rather than mutate.
```

Do not recommend a design where "audio network detected" automatically becomes "the audio network is the cause." That would create false positives and hide the true failure.

## Required Investigation

Inspect at least these areas:

```text
internal/classify/
internal/diagnose/
internal/diagnosisgraph/
internal/repair/
internal/orchestrator/
internal/toolmanifest/
recipes/
testdata/chaos/network/
testdata/chaos/network/mazes/
docs/testing/
governor/reports/
```

Run enough commands to understand current behavior. Recommended:

```bash
go test ./internal/classify ./internal/diagnose ./internal/diagnosisgraph ./internal/repair ./internal/orchestrator -count=1
bash scripts/run_sandbox_mazes.sh
agentlink chaos run --fixture testdata/chaos/network/mazes/m13-dante-vlan-no-internet-trap.json --json
agentlink diagnose-graph --from <m13 diagnostic JSON> --json
```

If a command cannot run in your cloud environment, classify that as environment/harness limitation, not product behavior.

## Research Questions

Answer these concretely.

1. What protected or intentional network topologies are still missing?
2. Which missing cases could lead to destructive or dirty repairs?
3. Which cases could become too conservative, blocking a valid repair because a protected topology is merely present?
4. How should AgentLink separate:
   - root cause,
   - protected constraint,
   - red herring,
   - allowed repair corridor,
   - forbidden mutation surface?
5. What structured fields should replace or supplement `DiagnosticReport.Raw` matching?
6. What should be deterministic kernel logic versus Gemma/Qwen advisory logic?
7. What should the model see in the compact diagnosis graph?
8. How should the GUI explain this to a non-expert user without causing unsafe clicks?
9. How should terminal tickets and support bundles preserve protected-topology evidence?
10. What should be tested before calling this commercial-grade?

## Protected Topology Taxonomy To Consider

Do not limit yourself to these. Expand the list.

```text
Dante / AES67 / RAVENNA / MTRX / audio VLAN
NDI / video production VLAN
camera-control / PTZ / Blackmagic / ATEM networks
lighting / Art-Net / sACN / show-control networks
NAS / iSCSI / SMB direct / backup-only networks
Thunderbolt bridge / direct Mac-to-Mac links
lab instruments / measurement LAN
industrial controller / PLC / SCADA-like local nets
VM/container bridges that look like active interfaces
VPN split-tunnel or management-only interfaces
MDM/corporate profiles with intentional DNS/proxy/route policy
airport/awdl/sidecar/continuity interfaces
USB Ethernet management interface versus Wi-Fi internet path
multi-home failover with stale default route
static IP private control network with no internet by design
link-local service discovery network that should not be DHCP-renewed
campus/enterprise captive or 802.1X networks with policy constraints
proxy/TUN/fake-IP that is present but not root cause
```

## Design Output Requirements

Produce a structured report with these exact sections:

```text
# AgentLink Protected Topology Research Result

## Verdict
PASS / FAIL / INCONCLUSIVE for the research task.

## Executive Summary
Short, direct, product-focused.

## What Current Patch Gets Right
Concrete observations from source/fixtures/tests.

## Remaining Unsafe Gaps
Ranked, with failure mode and likely harm.

## False-Positive Risks
Cases where protected-topology detection could over-block real repair.

## False-Negative Risks
Cases where AgentLink could miss a protected topology and dirty the environment.

## Proposed Topology Model
Schema fields, examples, and where they should be collected.

## Classifier And Precedence Rules
Rules for root cause vs protected constraint vs red herring.

## Repair Corridor Matrix
For each topology class: allowed read-only, allowed dry-run, allowed mutation, forbidden mutation, required approval.

## Model-Facing Graph Changes
What the compact diagnosis graph should expose to Gemma/Qwen and what it must hide.

## GUI / Terminal Ticket UX
Exact user-facing behaviors for protected topology cases.

## New Fixtures
At least 20 proposed fixture scenarios, each with:
- id
- topology
- symptom
- protected constraint
- true root cause
- expected class(es)
- expected recommended repair level
- forbidden actions

## Proposed Tests
Concrete unit/integration/gate tests.

## Top 10 Product Changes
Prioritized implementation plan.

## Safety Audit
Confirm no forbidden commands were run.

## Final Recommendation
Release now / hold / needs targeted follow-up.
```

## Scoring Guidance

Do not give credit for vague safety statements. Prefer testable rules, fixture proposals, and precise schema.

Do not claim real Mac behavior from Linux cloud.

Do not treat deterministic kernel pass as model behavior.

Do not treat model commentary as proof of repair correctness.

Do not weaken existing verifiers.

If you find a flaw in the current patch, state it plainly and propose a minimal patch or test that would catch it.

