# Cactus AgentLink Rescue — Architecture Digest

## What it is

An offline, portable, reversible macOS rescue tool for "restoring the broken path between a Mac and AI agents such as Codex, Claude, GitHub, and API endpoints" (`README.md`). It targets the failure where remote help is unreachable by definition: the Mac is offline, behind stale proxy settings, a missing route, broken DNS, or Clash/Mihomo TUN residue.

Shape: a Go 1.22 module (`go.mod`) built to a single `bin/agentlink` binary (`cmd/agentlink/main.go`), plus a SwiftUI app (`gui/CactusAgentLinkRescue/`) that shells out to it. `README.md`: "Runtime does not require Homebrew, Python, npm, pip, or network access. Building requires Go on the build machine." Counted on `main` (`ls`/`grep`/`wc`): 33 packages under `internal/`, 109 `.go` files (15,504 lines including tests), 12 recipes in `recipes/`, 31 verifier IDs in `internal/verifier/registry.go`, 27 failure-class constants in `internal/classify/classify.go`, 21 Swift files, 26 design docs in `docs/offline/`.

## Topology

Stages map onto packages almost one-to-one.

- **Diagnose** — `internal/diagnose` (engine, parsers, `tun.go`) and `internal/facts`. `README.md`: "Diagnose is non-mutating and does not require root."
- **Classify** — `internal/classify` turns the report into deterministic classes (`OK`, `NO_DEFAULT_ROUTE`, `SYSTEM_PROXY_DIRTY`, `CLASH_TUN_ACTIVE_OR_STALE`, …).
- **Plan** — `internal/planner` defines the `Decision` struct (`schema.go`); `internal/brain` drives the local Gemma model; `internal/orchestrator` carries a deterministic `supervise()` that needs no model (it imports `brain` for the optional path, but `supervise()` itself reads only the diagnose and TUN reports).
- **Validated recipe** — `internal/recipe` (schema, loader, registry, params, dry-run, runner) executes only bundled JSON from `recipes/`.
- **Repair** — `internal/repair`: levels `safe`, `tun`, `standard`, `standard-system-reset`, `deep`.
- **Verify** — `internal/verifier`, `internal/networkverify`, `internal/verify` (in-binary selftest).
- **Rollback** — `internal/snapshot` (restore points) and `internal/rollback`.
- Supporting: `internal/safety` (redaction), `internal/system` (path allowlist), `internal/ticket`, `internal/restartgate`, `internal/supportbundle`, `internal/session`, `internal/report`.

External commands go through `internal/command`: `exec.CommandContext(ctx, path, args...)` with a 10 s default timeout (`runner.go`) — a path plus an argument slice, never a shell string. The GUI is a thin shell over it: `CommandRunner.swift` builds a `Process` with `process.arguments = args`, and `README.md` states it "uses the embedded `bin/agentlink`; it does not use a system `agentlink` from `PATH` unless a developer override is set." It runs no privileged work — `AgentlinkClient.swift`'s `copyableTerminalCommand(_ args:, sudo:)` renders a `sudo ./bin/agentlink …` string for the user to paste.

## The model is a planner, not an executor

`README.md`: "Gemma is not the executor. It returns PlannerDecision JSON only." Model output is parsed (`internal/brain/json_extract.go`) and must pass `planner.ValidateDecision` (`internal/planner/validate.go`), which rejects a wrong `schemaVersion`, confidence outside 0–1, an unknown intent, a repair below 0.55 confidence, an unregistered recipe id, params failing `recipe.ResolveParams`, a risk exceeding the recipe's own, a writable recipe with no rollback ("writable recipe lacks rollback"), unknown verifiers or fallbacks, and secret-shaped text in explanation or evidence. A second gate, `brain.ValidateAutoExecution` (`internal/brain/policy.go`), refuses `privileged_action` and `destructive_action` in auto mode and requires `--yes` for `reversible_patch`, `--online` for `network_action`. On validation failure `internal/brain/planner.go` records the error in `ValidationErrors` and returns; there is no re-prompt.

Bad outputs are pinned as fixtures under `internal/brain/testdata/`: `gemma_output_with_text_around_json.txt` (prose wrapping the JSON), `gemma_planner_low_confidence.json` (0.30), against `gemma_planner_valid.json`; `internal/brain/brain_test.go` loads these three (lines 147, 213, 200/224). Two further fixtures, `gemma_output_invalid_json.txt` ("This is not JSON.") and `gemma_planner_unknown_recipe.json` (`"id": "does-not-exist"`), are present but not referenced by any test.

## Why not just reset the network?

`README.md` line 11: "This is not a generic network reset tool and not a cleanup app." A macOS network reset is one destructive act with no undo; it discards the proxy/VPN/TUN topology a user built on purpose. The repo's order is the opposite, and each piece is in the code.

**Snapshot first.** Mutating runs create a restore point before touching anything (`internal/repair/repair.go` calls `snapshot.NewRestorePointWithPolicy` before the action loop). `internal/snapshot/manifest.go` records per-file owner uid/gid, mode, file type and SHA-256, plus a redacted `commands.log`, `preflight.json` and `postflight.json`.

**Bounded.** `system.IsAllowedMutationPath` (`internal/system/paths.go`) refuses relative paths, `..` traversal, and anything under `/System`, `/bin`, `/sbin`, `/usr`; writes land only inside a named allowlist.

**Reversible on a worse result, automatically.** `criticalWorsened(pre, post)` (`internal/repair/repair.go`) compares preflight to postflight — lost default route, lost raw-IP/DNS/HTTPS reachability, newly dirty proxy, newly stale TUN, AWDL down — and on any of these calls `rp.RestoreAllWithRunner`, status `rolled_back_after_worsening`. The orchestrator says so verbatim: "Postflight was worse than preflight, so AgentLink automatically rolled back." (`internal/orchestrator/orchestrator.go`).

**Quarantine, not delete.** "Third-party residue is quarantined, not deleted" (`README.md`); `snapshot.QuarantinePath` copies the file into `files/` and *moves* the original into `quarantine/`; rollback refuses to overwrite a path that came back on its own.

**Targeted before broad.** `orchestrator.supervise()` picks `macos-clash-tun-force-repair` when TUN signatures are present, with the reason "targeted TUN runtime repair is safer than broad standard network reset"; `README.md` records that "the old broad clean-location/route-flush behavior is demoted to `standard-system-reset`". Detect-only tools (Tailscale, WARP, Little Snitch — `internal/interference`) are reported, never removed.

**Escalation is gated.** `deep` requires `--yes` (`internal/cli/cli.go`); without root the orchestrator creates a Terminal repair ticket instead of trying (`internal/orchestrator/orchestrator.go`, state `PrivilegeGate`); `internal/restartgate` covers state macOS will not release without a restart.

## How truth is enforced

`internal/verifier` holds 31 named checks; a recipe lists which must pass (`recipes/macos-clash-tun-force-repair.json` names `default_route_present`, `raw_ip_ping_ok`, `dns_lookup_ok`, `https_baidu_ok`, `stale_tun_absent_or_down`, `awdl0_up`). Any `fail` sets `StatusVerifierFailed` in `internal/recipe/runner.go`. `docs/offline/AGENTLINK_CONSTITUTION.md` puts it as rule 12: "Verifier owns truth."

Tests: 28 `_test.go` files, 100 `func Test…` (92 under `internal/`, 8 in `scripts/package_test.go`). `internal/command/mock.go` provides a `MockRunner` keyed by the rendered command line, used in 10 test files, so repair paths run without touching the machine; `internal/installer/installer_test.go:94` is a literal case table; `internal/diagnose/parsers_test.go` pins parsers against captured `scutil --proxy` and `route` output (`testdata/scutil_proxy_{clean,dirty}.txt`, `route_default_{ok,missing}.txt`). `agentlink selftest` re-runs a subset inside the shipped binary (`internal/verify/selftest.go`), including URL-credential redaction and restore-point-id generation. GUI changes are dogfooded visually: 29 PNGs under `docs/gui-dogfood/`, `v0.4.1` kept as `before/` + `after/` pairs.

## How to read the repo

`README.md` → `docs/offline/AGENTLINK_CONSTITUTION.md` (12 rules) → `docs/offline/RESCUE_ORCHESTRATOR.md` and `PLANNER_CONTRACT.md`. Then `internal/orchestrator/orchestrator.go` (253 lines, the whole loop), `internal/planner/validate.go` (the gate), `internal/snapshot/manifest.go` (the undo), `recipes/*.json`. `internal/cli/cli.go` is the 1,901-line command surface.

## Non-claims

Among the 20 bullets of `README.md`'s "What It Does Not Do": no GUI-owned repair logic, no model-generated shell execution, no permanent privileged helper, no SMAppService, no MDM/profile modification, no automatic removal of detect-only VPN or security tools, no persistent daemon, no telemetry, no cloud sync, no network-based rule updates, no RAG or queue, no automatic proxy/TUN enablement. "Known Limitations" include: macOS only; rules and recipes are static and bundled; deep rescue may require reboot; Gemma "can recommend only registered recipes"; for the DeepSeek provider recipes "an explicit online smoke test is required before treating the provider as operational".

Repository state: the public repository carries `main` only — 12 commits, tip `85294b4` (2026-05-04, `internal/system/version.go` = 0.5.0), tags `v0.2.2` … `v0.4.5`. This digest describes that tree.

---

_Reviewed line by line against `main` (85294b4) by a second agent on 2026-09-10; eight statements in the first draft were corrected to match the code._
