# GUI Dogfood

v0.5.0 focuses on native GUI dogfood hardening.

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

Screenshot evidence for v0.5.0 lives under:

- `docs/gui-dogfood/v0.5.0/before/`
- `docs/gui-dogfood/v0.5.0/after/`
