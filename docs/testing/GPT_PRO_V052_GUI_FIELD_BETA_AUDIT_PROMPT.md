# GPT Pro Prompt: AgentLink Rescue v0.5.2 GUI Field Beta Audit

You are auditing an AgentLink Rescue v0.5.2 GUI field beta package.

This is a source/runtime audit and controlled field-beta readiness review. It is
not a public release gate, not a notarization proof, not model proof, and not
real mutating network repair proof.

## Objective

Determine whether v0.5.2 genuinely makes the native GUI usable as the primary
controlled field beta entry point:

1. The normal user starts at `Rescue`.
2. The primary action is `Check & Plan Rescue`.
3. The primary action is dry-run-only and does not run sudo.
4. `Run Field Beta Check` verifies package health, journal state, CLI selftest,
   read-only diagnosis, guided rescue dry-run, readiness, and support-bundle
   export.
5. The GUI surfaces support-bundle/report evidence without leaking local home
   paths in selftest output or screenshot proof.
6. Developer/repair affordances remain gated and do not become a broad one-click
   mutation path.
7. Existing v0.5.1 corpus/FTS/privacy/gate regressions remain green.

Report product failures, harness failures, cloud limitations, stale-binary
risks, documentation/package issues, and inconclusive surfaces separately.

## Required First Reads

From the extracted package root, read:

```text
README_FIRST_FOR_GPT_PRO_V052_GUI_BETA_AUDIT.md
PACKAGE_MANIFEST.json
governor/reports/V0521_GUI_FIELD_BETA_READINESS.md
docs/offline/GUI_DOGFOOD.md
docs/offline/GUI_SAFETY_MODEL.md
gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/AppState.swift
gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/AgentlinkClient.swift
gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/CommandRunner.swift
gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Views/StatusDashboardView.swift
scripts/dogfood_gui_core.sh
scripts/gpt_pro_v052_gui_beta_audit.sh
```

## Main Command

Run:

```bash
FULL=1 OUT="$PWD/gpt-pro-results/v052-gui-field-beta" bash scripts/gpt_pro_v052_gui_beta_audit.sh
```

If one-shot execution is killed by the cloud tool's wall-time/no-output behavior,
run it in the background and poll logs. If that is still impossible, run:

```bash
FULL=0 OUT="$PWD/gpt-pro-results/v052-gui-field-beta-fast" bash scripts/gpt_pro_v052_gui_beta_audit.sh
```

Do not mark full regression green unless `FULL=1` completes.

## Manual Isolation Commands

Use these for attribution:

```bash
go version
go build -trimpath -ldflags "-s -w" -o .gpt-pro-build/agentlink ./cmd/agentlink
go build -trimpath -ldflags "-s -w" -o .gpt-pro-build/agentlink-rescue ./cmd/agentlink-rescue
.gpt-pro-build/agentlink version
.gpt-pro-build/agentlink doctor --json
.gpt-pro-build/agentlink guided rescue --target auto --dry-run --json
.gpt-pro-build/agentlink support bundle --json
go test ./internal/genome ./internal/cli ./internal/guided ./internal/supportbundle -count=1
go vet ./...
go test ./... -count=1 -timeout 35m
```

On macOS only:

```bash
swift build -c release --package-path gui/CactusAgentLinkRescue
bash scripts/dogfood_gui_core.sh
dist/Cactus\ AgentLink\ Rescue.app/Contents/MacOS/CactusAgentLinkRescue --selftest-gui
```

Do not classify macOS GUI runtime as failed only because a Linux cloud sandbox
cannot compile or run SwiftUI/AppKit. Classify that surface as a cloud
limitation unless the package claims Linux GUI support.

## Expected Evidence

Minimum evidence:

```text
source tree manifest says sourceTreeDirtyAtPackageTime=false
no model weights, runtime binaries, release archives, node_modules, .build, or >50MB files in package
CHECKSUMS.txt passes
agentlink version == agentlink 0.5.1
V0521 report exists and states controlled field beta, not public release
StatusDashboardView contains Check & Plan Rescue, Run Field Beta Check, Export Support Bundle, no-sudo wording
AppState contains runBetaReadinessCheck
runBetaReadinessCheck invokes guided rescue with --dry-run, not --yes
runBetaReadinessCheck invokes support bundle
runGuarded serializes GUI commands
CommandRunner redacts /Users/<name> paths
AgentlinkClient --selftest-gui emits betaReadiness
betaReadiness has mainActionDryRunOnly=true, noSudoInGUI=true, adminRepairsUseTerminalTicket=true
scripts/dogfood_gui_core.sh asserts betaReadiness.ready and safety fields
docs/gui-dogfood/v0.5.2/after/beta-readiness-dashboard.jpg exists and contains no literal /Users/jack or /Users/bowei string
v0.5.1 FTS/privacy/phase audit remains pass
go vet ./... passes
FULL=1: go test ./... passes
```

Safety expectations:

```text
sudo executed? no
agentlink rescue --yes executed? no
guided rescue --yes executed? no
real network mutation executed? no
networksetup mutation executed? no
route mutation executed? no
ifconfig mutation executed? no
launchctl mutation executed? no
pf mutation executed? no
VPN mutation executed? no
Keychain extraction executed? no
model weights present? no
runtime binaries present? no
live secrets found? no
```

## Audit Questions

Answer directly:

1. Does the GUI present a normal-user rescue dashboard rather than requiring
   developer pages?
2. Is the primary GUI path dry-run-only?
3. Does the beta check actually run the required CLI surfaces?
4. Does `--selftest-gui` expose machine-readable beta readiness?
5. Does the package prevent or gate overlapping GUI command runs?
6. Does support-bundle evidence avoid leaking local home paths in GUI proof?
7. Does the existing v0.5.1 corpus/FTS/privacy/gate line still pass?
8. What surfaces remain inconclusive because GPT Pro cloud is Linux?
9. What would still block public release?

## Failure Classification

Use these labels:

```text
product failure
harness failure
cloud environment limitation
stale binary contamination
documentation/package issue
inconclusive
```

## Report Format

Return:

```text
AgentLink v0.5.2 GUI Field Beta GPT Pro Audit Result

Package:
SHA256:
Host:
Go:
Swift/macOS GUI runtime available? yes/no

Verdict:
- GUI field beta audit: PASS | FAIL | INCOMPLETE
- v0.5.1 regression carry-forward: PASS | FAIL | NOT RUN | INCONCLUSIVE
- public release readiness: PASS | FAIL

Suite results:
- checksum:
- forbidden artifact scan:
- build agentlink:
- CLI guided dry-run:
- CLI support bundle:
- static GUI beta contract:
- GUI selftest/dogfood:
- v0.5.1 audit:
- go vet:
- full go test:

Findings:
- product failures:
- harness failures:
- cloud limitations:
- stale binary risks:
- documentation/package issues:
- inconclusive surfaces:

Safety:
- sudo executed? no
- real network mutation executed? no
- model/runtime/archive files present? no
- secrets found? no

Recommendation:
- accept controlled GUI field beta | fix required | rerun required
```
