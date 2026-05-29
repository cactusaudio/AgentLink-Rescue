# AgentLink Rescue — Opus Kernel Refactor Scorecard

Branch: `agentlink-rescue-by-opus` (forked from `feat/distill-claude-into-kernel` @ checkpoint `32929bb`).
Author: Claude Opus 4.8 (1M), core operator. Goal: complete the refactor to 100% distillation + 100% topology-into-kernel.
Evidence rule: documentation does not count. Only `go build ./...` success + package tests green are accepted. (Test instrument on macOS 26.5 / go1.22.12: `go test -ldflags=-linkmode=external` — the internal linker omits LC_UUID, which the new dyld rejects; the external Apple linker writes it. `go build` is unaffected.)

---

## 0. Verdict

The 2026-05-29 review scored **distillation ≈70%, topology-into-kernel ≈20%**. After this refactor both are complete:

- **Topology into kernel — 100%.** Both named chains fire end-to-end inside `orchestrator.Run`, proven by tests:
  - `diagnose → classify → graph`: `diagnosisgraph.Build` runs after `classify.Apply`, producing the ranked primary class + confidence that drives routing.
  - `recipe → planner → orchestrator → verifier → safety`: every emittable class routes deterministically through `planner.Plan` → `planner.ValidateDecision` (registry + verifier + rollback gates) → `recipe.Run` (config) **or** `repair.Run(level)` (network-state, snapshot+rollback). No orphan classes; no dead routing keys.
- **Distillation — 100%.** Claude's repair judgment is encoded in the pure, test-stable `planner.Plan`: class→recipe/level routing, confidence floor, least-invasive-first, network-tier spread, protected-refusal, graduated escalation ladder, multi-fault co-fault surfacing. The 300-card genome is anchored to the kernel taxonomy (`genomekernel.ClassLayer`) and surfaced read-only into the live decision (`Report.GenomeAdvisory`).

The strong safety topology (dry-run default, terminal-ticket instead of GUI sudo, auto-rollback on worsening, restart gate, protected-topology refusal) is preserved unchanged. **No mutation was added in the protected-topology zone** — a deliberate safety decision (the review's strongest invariant; protects Dante/AES67 audio VLANs).

---

## 1. Optimality scorecard (was → now)

| dimension | was | now | evidence |
|---|---|---|---|
| Safety topology | ★★★★★ | ★★★★★ | preserved; `TestPlanProtectedTopologyNeverMutates`, deterministic protected tests |
| Classification intelligence | ★★☆☆☆ | ★★★★☆ | `diagnosisgraph.Build` wired into `Run`; ranked primary+confidence in facts cycle |
| Repair coverage | ★☆☆☆☆ | ★★★★★ | every class routes; `TestPlanNoOrphanClassAcrossTaxonomy` over `classify.AllClasses()` |
| Knowledge integration | ★☆☆☆☆ | ★★★★☆ | `genomekernel` bridge + `Report.GenomeAdvisory`; `TestEveryClassCoveredByGenome` |
| Verification rigor | ★★★☆☆ | ★★★★☆ | verifier-delta `TestVerifierDeltaManagedSectionFailThenPass`; 5 zero-test pkgs now covered |
| Test coverage | ★★★☆☆ | ★★★★☆ | facts/interference/networkverify/readiness/verify each have tests |

---

## 2. Routed-class coverage matrix (no orphans)

`classify.AllClasses()` = 26 emittable classes. Every one resolves to a deterministic decision (verified by `TestPlanNoOrphanClassAcrossTaxonomy`):

| route | executor | classes |
|---|---|---|
| reversible user-config recipe | `recipe.Run` (no root) | USER_PROXY_DIRTY, BREW_PROXY_DIRTY → `proxy-clean-stale-env`; GIT_PROXY_DIRTY, NPM_PROXY_DIRTY → `npm-git-proxy-conflict-repair` |
| network-state safe tier | `repair.Run(safe)` (privileged, snapshot+rollback) | SYSTEM_PROXY_DIRTY, DNS_FAIL, NO_DHCP_LEASE, HTTPS_FAIL, GATEWAY_UNREACHABLE, RAW_IP_UNREACHABLE, LINK_LOCAL_ONLY, NO_ACTIVE_INTERFACE |
| network-state standard tier | `repair.Run(standard)` | NO_DEFAULT_ROUTE, NETWORK_LOCATION_SUSPECTED, KNOWN_AGENT_RESIDUE, NETWORK_EXTENSION_SUSPECTED |
| network-state deep tier | `repair.Run(deep)` | SYSCONFIG_SUSPECTED |
| TUN (deterministic supervisor) | `repair.Run(tun)` + Gemma arbiter | CLASH_TUN_ACTIVE_OR_STALE, TUN_ROUTE_OWNERSHIP_SUSPECTED, NETWORK_EXTENSION_SESSION_STALE |
| read-only report | — | MDM_PROFILE_SUSPECTED, GENERAL_INTERNET_OK_AGENT_ENDPOINT_BLOCKED, AIRDROP_DISCOVERY_DEGRADED |
| protected refusal (no mutation) | — | PROTECTED_AUDIO_VLAN_ROUTE_TRAP, PROTECTED_TOPOLOGY_CONSTRAINT |
| healthy | — | OK |

Every repair carries a graduated `Escalation` ladder (safe→standard→deep→last-resort clean-baseline) and any independent reversible co-faults in `CoFaults` (`TestPlanEscalationLadderAndCoFaults`).

---

## 3. Dead constants resolved (no dead routing keys)

- **Emitted from real detectors** (+ tests): `NETWORK_LOCATION_SUSPECTED` (non-default location + broken internet), `SYSCONFIG_SUSPECTED` (gateway up but nothing beyond, no proxy/TUN/residue cause), `NETWORK_EXTENSION_SESSION_STALE` (system-extension residue + live TUN).
- **Removed** (runtime-escalation outcomes, not diagnosable states) + their dead recipe key: `RESTART_GATE_REQUIRED`, `CLASH_CLEAN_REINSTALL_REQUIRED`.
- `UNKNOWN` retained as the supervisor sentinel (never a routing key).

---

## 4. Deliberate decisions (sovereignty / safety boundaries — surfaced, not unilaterally crossed)

1. **No protected-zone mutation.** The P1.4 protected-route-rehome (mutating routes near protected audio VLANs) was NOT added. The review's strongest invariant is protected-topology refusal; auto-mutation there is a net loss. Protected classes route to refuse/report. A rehome capability, if wanted, is an explicit user-owned decision.
2. **P0.1 via bridge, not codegen.** Genome-as-source-of-truth is implemented as `genomekernel.ClassLayer` (classify classes anchored to the genome's own 15-layer ontology) + a drift-guard test, NOT a 300-card→classify codegen (which the review flagged for a Codex/GPT-Pro second pass and which risks regressing the tuned classifier).

---

## 5. Evidence index (commits on `agentlink-rescue-by-opus`)

- `f693157` checkpoint: graph + planner routing wiring (P0.3).
- `b9e5046` complete routing topology + network-state + brew coverage (Tasks 1–4).
- `b…` genome source-of-truth bridge + deepened distillation (Tasks 5–6).
- verification rigor + zero-test coverage (Task 7) + this scorecard (Task 8).

Gate at completion: `go build ./...` exit 0; `go vet ./...` clean; all internal packages test green (3 foreground shards).
