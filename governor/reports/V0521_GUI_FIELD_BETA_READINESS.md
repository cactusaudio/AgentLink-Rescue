# AgentLink v0.5.2 GUI Field Beta Readiness

Date: 2026-05-21

Status: controlled field beta candidate, not public release, local review diff, not pushed.

Version note: `v0.5.2-gui-field-beta-alpha` is the GUI/audit-package line.
The shared CLI core still reports `agentlink 0.5.1` until the public release
version bump.

## Objective

Make the native GUI usable as the primary AgentLink entry point for controlled
field beta: diagnose, plan, export evidence, and hand off privileged repairs
without GUI sudo or direct network mutation.

## Implemented

- Added a visible `Field Beta Readiness` card to the main `Rescue` dashboard.
- Added a `Run Field Beta Check` button that runs package, journal, CLI
  selftest, read-only diagnosis, guided dry-run, readiness, and support-bundle
  checks.
- Extended `--selftest-gui` with a machine-readable `betaReadiness` object.
- Strengthened `scripts/dogfood_gui_core.sh` to assert:
  - `betaReadiness.ready == true`;
  - main action is dry-run-only;
  - GUI does not run sudo;
  - admin repair remains Terminal-ticket based;
  - guided dry-run and support-bundle export are available.
- Serialized GUI command execution so repeated clicks cannot launch overlapping
  rescue/diagnostic commands.
- Reclassified package repair as a mutating GUI operation for button gating.

## GUI Evidence

Manual app launch and Computer Use inspection confirmed the normal-user
dashboard shows:

- `Check & Plan Rescue`;
- `Run Field Beta Check`;
- `Export Support Bundle`;
- safety badges for local diagnostics, dry-run first, and approval required;
- protected topology status on the live Mac;
- final `Ready for field beta` status after the beta check.

Screenshot:

```text
docs/gui-dogfood/v0.5.2/after/beta-readiness-dashboard.jpg
```

## Verification

```text
swift build -c release --package-path gui/CactusAgentLinkRescue
PASS

bash scripts/dogfood_gui_core.sh
PASS
core GUI zip SHA256:
30b7abbba0ff85c72ed17fb753b8c6764f6127cff646e87d22f1e432937ca230

CactusAgentLinkRescue --selftest-gui
PASS
betaReadiness.ready=true
no literal local home path leaked in GUI selftest JSON

go vet ./...
PASS

go test ./... -count=1 -timeout 35m
PASS
internal/chaos: 840.956s
```

## Safety Boundary

No real host network repair was executed. The GUI beta path runs read-only
diagnosis, dry-run planning, readiness checks, and support-bundle export. Admin
work remains a Terminal-ticket handoff.

The live Mac currently has a protected-topology signal. The GUI correctly keeps
that visible as a safety status and does not convert it into a broad repair
button.

## Not Proven

Still outside this proof:

- notarized public release;
- signed installer/DMG flow;
- GUI behavior on other Macs;
- real mutating network repair on disposable physical network fixtures;
- model-runtime behavior;
- external GPT Pro review of this exact GUI beta diff.

## Recommendation

Accept as controlled GUI field beta candidate after review. Do not call it a
public release until signing/notarization, cross-Mac GUI QA, and real disposable
mutation/rollback proof are complete.
