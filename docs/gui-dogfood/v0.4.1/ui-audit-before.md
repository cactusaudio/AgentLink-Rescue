# Cactus AgentLink Rescue v0.4.1 UI Audit - Before

Source build: v0.4.0 Brain GUI package launched from Downloads.

Screenshots captured with `screencapture`:

- `before/dashboard.png`
- `before/doctor.png`
- `before/brain.png`
- `before/plan.png`
- `before/dry-run.png`
- `before/rescue.png`
- `before/rollback.png`
- `before/reports.png`
- `before/settings.png`

Findings:

- Dashboard showed `Recommended: unknown` while Doctor JSON contained `recommendedRepairLevel: none` in the nested network diagnostic.
- Doctor reused the last dry-run raw output before its own Run Doctor action completed.
- Plan reused the last dry-run raw output on entry, making the page look like a plan had already run.
- Dry Run reused stale output until the user clicked Run Dry-Run.
- Reports showed an older cached session until `Load Latest Reports` was clicked.
- Raw JSON blocks were visually dominant on most pages; they should default to collapsed debug detail.
- Brain status correctly detected package-local model/runtime, but paths were too prominent and long.
- Network Rescue Advanced correctly displayed copyable sudo commands and did not execute them.
- Execute became available after a matching dry-run, but the dependency could be stated more clearly.
