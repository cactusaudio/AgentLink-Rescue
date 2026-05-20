# GUI Dogfood

v0.5.1 focuses on native GUI dogfood hardening.

## v0.5.2 Field Beta Landing

The main GUI entry is now the `Rescue` dashboard, not the developer pages.
The first action remains `Check & Plan Rescue`, which runs a guided rescue
dry-run and does not execute sudo or network mutation from the GUI.

The dashboard also exposes `Run Field Beta Check`. This check verifies:

- package doctor;
- repair journal recovery state;
- CLI selftest;
- read-only diagnosis;
- guided rescue dry-run;
- readiness doctor;
- support-bundle export;
- GUI safety boundary: main rescue is dry-run-only and privileged repair stays
  in Terminal tickets.

Current screenshot evidence:

- `docs/gui-dogfood/v0.5.2/after/beta-readiness-dashboard.jpg`

This is a controlled field beta proof, not a public release proof. It does not
claim notarization, signed distribution, real mutating network repair, or GUI
behavior on another Mac.

Recommended checks:

1. Build and package Core GUI.
2. Build and package Brain GUI.
3. Launch Core GUI from an extracted folder with no Brain assets.
4. Confirm Core GUI shows Brain assets missing as a warning, not a fatal error.
5. Launch Brain GUI from Downloads or another path with spaces.
6. Run Doctor, Brain Doctor, Brain Selftest, Brain Plan, and Auto Repair Dry-Run.
7. Confirm Plan and Dry Run pages do not show stale output before their own commands run.
8. Confirm Reports auto-loads the latest session on entry.
9. Confirm Execute is disabled until a matching dry-run succeeds.
10. Confirm changing target invalidates the previous dry-run execution state.
11. Confirm Network Rescue Advanced only presents copyable sudo commands.
12. Confirm Settings shows package-local paths and redaction enabled.

Screenshot evidence for v0.5.1 lives under:

- `docs/gui-dogfood/v0.5.1/before/`
- `docs/gui-dogfood/v0.5.1/after/`
