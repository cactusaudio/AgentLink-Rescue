# Cactus AgentLink Rescue v0.4.1 UI Audit - After

Date: 2026-05-02

Test object:

- `dist/Cactus AgentLink Rescue.app`
- Embedded `agentlink 0.4.1`
- Brain package with package-local Gemma GGUF and llama.cpp runtime

Screenshots:

- `after/dashboard.png`
- `after/doctor.png`
- `after/brain.png`
- `after/brain-selftest.png`
- `after/plan.png`
- `after/plan-result.png`
- `after/dry-run.png`
- `after/dry-run-result.png`
- `after/rescue.png`
- `after/rescue-advanced.png`
- `after/rollback.png`
- `after/reports.png`
- `after/settings.png`

## Findings

### Fixed

- Dashboard now maps `recommendedRepairLevel: none` to `Recommended: None` with an OK badge instead of `unknown`.
- Plan page initial state is isolated: it shows `Not run` and does not inherit Brain selftest output.
- Dry Run page initial state is isolated: it shows `Not run` and does not inherit Plan output.
- Target changes clear stale Plan/Dry Run state. After switching from `path` to `proxy`, Execute is disabled and the page asks for a matching dry-run.
- Reports auto-loads the latest session on first entry. The latest dry-run session generated during this audit was displayed without pressing Refresh.
- Raw command output is collapsed by default across Doctor, Brain, Plan, Dry Run, Rescue, Rollback, Reports, and Settings.
- Brain page now separates Brain pack status, model, runtime, and selftest status. Model/runtime package-local availability is clear.
- Network Rescue remains copy-only. The Advanced section explicitly says the GUI does not collect passwords and does not run sudo commands.
- Settings shows package root and embedded binary path from the app bundle resource tree.

### Manual smoke results

- Brain Selftest: OK, recipe `api-key-detection-redaction`.
- Brain Plan target `path`: OK, selected `macos-zsh-path-repair`, confidence `0.80`, risk `reversible_patch`.
- Brain Auto Repair Dry Run target `path`: OK, status `dry-run`, planned `append_managed_block_if_missing` for `~/.zshrc`, no changes made.
- Execute dependency: enabled only after matching `path` dry-run; disabled after target changed to `proxy`.
- Reports: latest session auto-loaded and showed the matching dry-run report.

### Remaining v0.4.1 limitations

- The app is unsigned and not notarized.
- Screenshot dogfood is semi-automated; it uses app selftest plus macOS `screencapture` rather than a fully deterministic in-app page navigation harness.
- The GUI does not run sudo operations directly. System network rescue and system rollback remain copyable Terminal commands.
- The GUI is still a thin shell over package-local `agentlink`; all repair semantics live in the CLI.
