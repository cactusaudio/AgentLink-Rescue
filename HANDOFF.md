# AgentLink Rescue — Engineer / Agent Handoff

Read this FIRST to take over cold. This is the **working monorepo** for the macOS
connectivity-rescue tool. The top-level `README.md` body documents the *pre-Opus*
v0.5.2 GUI field-beta base; this handoff + the two governor reports below are the
authoritative picture of the current **Opus kernel refactor**:

- `governor/reports/AGENTLINK_KERNEL_OPUS_REFACTOR_SCORECARD.md` — what the Opus refactor delivered, with cited test names + the routed-class coverage matrix. **The authoritative status doc.**
- `governor/reports/AGENTLINK_KERNEL_ARCHITECTURE_REVIEW.md` — the pre-refactor review that scoped the work (distillation 70% / topology 20% → both completed).

## What this is

The macOS connectivity-rescue tool (offline, portable, reversible), module
`cactus-agentlink-rescue` (Go 1.22), with the **Opus kernel refactor** applied.
It restores the broken path between a Mac and remote AI agents (Codex / Claude /
GitHub / API endpoints): diagnoses the Mac, classifies the failure, applies
bounded reversible repairs, and keeps every mutating action rollback-able. Gemma 4
(and the 64GB Qwen advisory) are explanation/shadow-only; deterministic rules,
recipes, verifier, rollback, and a restart gate own all release behavior.

## Repo layout & branches (current, as of 2026-05-30)

- **This repo** (`cactusaudio/AgentLink-Rescue`) is the working monorepo. Branches:
  **`agentlink-rescue-by-opus`** (live — HEAD carries the Opus refactor + nuclear
  button) and **`main`** (trunk, behind the live branch). Stale merged branches
  (`feat/distill-claude-into-kernel`, `lab/gemma-opencode-flywheel`,
  `sprint/v0.3.0-agentlink`, `from-mac/v0.5.1-core-reliability`) and the
  `AgentLink-Rescue-v0.5.1-from-mac` worktree were pruned 2026-05-30; **all their
  commits survive in `agentlink-rescue-by-opus`** (they were fully merged).
- `cactusaudio/AgentLink-Rescue-by-Opus` = a clean 2-commit **snapshot** of the live
  branch (the public / handoff face; a frozen export that will drift as this branch
  advances).
- `cactusaudio/AgentLink-for-Win` = the from-scratch **Windows port** (separate module
  `agentlink-for-win`; VM-verified on Win11 ARM64).

## What the Opus refactor added (vs the `01ced16` v0.5.2 base)

1. **Topology wired into the kernel.** `orchestrator.Run` now runs
   `diagnose → classify → diagnosisgraph.Build (ranked primary class + confidence)`
   and routes `recipe → planner → orchestrator → verifier → safety`:
   `planner.Plan` (registry-driven, the distilled judgment) → `planner.ValidateDecision`
   → `recipe.Run` (config recipes) **or** `repair.Run(level)` (network-state tiers).
2. **No orphan classes / no dead routing keys.** 6 dead `classify` constants resolved;
   `classify.AllClasses()` is the canonical taxonomy and every class routes
   deterministically (proven by `TestPlanNoOrphanClassAcrossTaxonomy`).
3. **Genome as source of truth (P0.1, via bridge not codegen).** `internal/genomekernel`
   anchors each `classify` class to the genome's 15-layer ontology (`ClassLayer`) + a
   drift-guard test (`TestEveryClassCoveredByGenome`); orchestrator attaches a read-only
   `Report.GenomeAdvisory`.
4. **Deeper distilled judgment.** `planner.Decision` carries an escalation ladder
   (safe→standard→deep→nuclear) and multi-fault co-faults.
5. **Verification rigor (P5).** verifier-delta harness + the 5 previously-zero-test
   packages (facts/interference/networkverify/readiness/verify) now covered.
6. **Connectivity-first nuclear button.** `repair.LevelNuclear` — maximal reset that
   **overrides the protected-topology boundary** (operator directive: getting online
   outranks preserving an audio VLAN), with a static-IP route-rebuild fallback.
   `agentlink rescue --level nuclear --yes`.

## Architecture (key packages)

| package | role |
|---|---|
| `internal/orchestrator` | `Run` = the kernel loop (diagnose→graph→supervise/planner→dispatch); `reversibleRepairPlan`/`runLevelRepair`, `connectivityBroken`, `genomeAdvisory`. |
| `internal/diagnose` | macOS collectors (scutil/networksetup/ifconfig/route/...) → `DiagnosticReport`; `DiagnoseTun`, topology analysis. |
| `internal/classify` | the 26-class taxonomy + `Classify` + `AllClasses` + `RecommendedRepairLevel`. |
| `internal/diagnosisgraph` | `Build` → ranked primary class + confidence + classRoute. |
| `internal/planner` | `Plan` (class→recipe/level, distilled), `ValidateDecision` (registry+verifier+rollback gates), `Decision`. |
| `internal/recipe` | registry + `RecipesForClass` (reverse index) + `Run` (config-patch executor). |
| `internal/genomekernel` | `ClassLayer` bridge + `CoverageForClass`/`AdvisoryForClass`. |
| `internal/repair` | level engine (safe/standard/tun/clean-baseline/deep/**nuclear**), `buildNuclearActions`, static-IP fallback, network snapshot + rollback. |
| `internal/verifier` | verifiers ("own truth"); recipe.Run runs them + rolls back on failure. |
| `agentlink_rescue_network_genome_v0_2/` | the 300-card / 15-layer genome corpus (knowledge asset, SSOT per `AGENTS.md`). |

## Build & test

```
go build ./...                                                  # green
go test -ldflags=-linkmode=external -timeout 180s ./internal/<pkg>/   # per-package, green
go vet ./...
```

**macOS 26 + go1.22 caveat:** `go test` binaries from the internal linker abort
("missing LC_UUID"); use `-ldflags=-linkmode=external` (Apple ld writes LC_UUID).
`go build` is unaffected. Toolchain: `~/sdk/go1.22.12`. Run build/test **sharded per
package, foreground** (the genome/brain packages are heavier; chaos shard ~14 min).

## Current state

`go build ./...` green; every test-bearing package green; `go vet` clean. The two
named chains fire end-to-end with tests; the nuclear button is implemented (unit +
logic tested). The macOS nuclear path is logic/unit-verified; the live
break→reset→online loop was verified on the **Windows** port in a VM, not against a
deliberately-broken live macOS host this session.

## Known gaps / decisions (for the next agent)

- **README.md body is the pre-Opus v0.5.2 base** (a banner at its top now redirects to
  this handoff + the scorecard). Update it further or rely on this handoff.
- **P0.1 is a bridge, not codegen.** Genome↔classify share identifiers via
  `genomekernel.ClassLayer` + a drift test; classify is NOT generated from the 300
  cards (full codegen flagged for a second opinion — it risks regressing the tuned
  classifier).
- **Protected-topology refusal is now overridable** by `--level nuclear`
  (connectivity-first operator directive). The graduated tiers still respect it.
- **GUI not updated** for the nuclear button / Opus routing (the GUI is the v0.5.2
  field-beta surface; the Opus work is in the kernel/CLI).
- **Biggest open backlog = P2 ingest-schema expansion** (capture DNS-returned IPs +
  HTTP/TLS error strings → `DNS_POISON_SUSPECTED` / `SNI_RESET_SUSPECTED`) to unlock
  the China-mainland failure classes the 64GB Qwen harvest exposed
  (`governor/reports/QWEN_GEM_FREE_RUN_*`). Schema first, no targeted model training yet.
- **Hard release blocker = signing/notarization** — 0 valid code-signing identities on
  the build Mac (`docs/signing-notarization.md`); portable beta is ad-hoc signed only.
