# GUI Screenshot Audit

v0.5.x uses screenshot-driven review for the native macOS GUI. v0.5.2 adds field-beta readiness and portable-package launch proof on top of the earlier screenshot pass.

Audit flow:

1. Launch the packaged GUI app.
2. Capture Dashboard, Doctor, Brain, Plan, Dry Run, Rescue, Rollback, Reports, and Settings.
3. Record layout/state issues in `ui-audit-before.md`.
4. Apply targeted GUI hardening changes.
5. Capture the same pages again under `after/`.
6. Record remaining issues and fixed items in `ui-audit-after.md`.

The audit checks:

- Status cards match CLI JSON.
- Raw command output does not appear on unrelated pages.
- Reports open on the latest available session.
- Brain missing assets are nonfatal in Core GUI.
- Brain package-local model/runtime are clear in Brain GUI.
- Execute actions remain gated by a successful matching dry-run.
- Sudo/system rescue actions are copyable commands only.
