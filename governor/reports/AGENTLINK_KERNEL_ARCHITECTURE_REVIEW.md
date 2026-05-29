# AgentLink Rescue — Kernel Architecture Review

Date: 2026-05-29
Reviewer: Claude Opus 4.8 (1M), advisor role. Static source + import-graph + binary behavioral spot-check; `go` not on PATH so no recompile.
Repo: `/Users/jack/CactusLocalAgent/AgentLink-Rescue` HEAD `01ced16` (post-v0.5.2-gui-field-beta-alpha hardening).
Scope question: *Is the kernel's structure / design / topology already optimal, and has it reached "distill ALL of a top model's local-network-repair capability and topologically optimize it to the extreme"?*
Method note: two breadth agents mapped the genome wiring and the input-surface/coverage matrix; I read `classify.go`, `orchestrator.go` in full and corrected one agent error (the orchestrator **does** import `brain` — Gemma is wired, just heavily caged).

---

## 0. Verdict

**No, not optimal — and not because of poor execution, but because the project completed *distillation* without completing *integration into the execution kernel*.**

- The knowledge was extracted: `agentlink_rescue_network_genome_v0_2/` holds ~300 failure cards, a diagnosis DAG, a 15-layer ontology, 8 indexes, and 4 knowledge volumes (macOS / proxy-VPN / dev-connectivity / pro-audio-Dante).
- The smart structures were built: Go packages `diagnosisgraph` (confidence/edge-weighted DAG), `planner` (PlannerDecision contract + validator), `brain` (Gemma llama.cpp backend).
- **But the execution kernel consumes almost none of it.** The "Fix My Connection" orchestrator path can autonomously execute exactly **one** recipe (`macos-clash-tun-force-repair`). The actual diagnostic brain is **613 lines of hand-coded `if` statements** in `classify.go`. The genome, the `diagnosisgraph` package, and the `planner` package are **not imported by the rescue pipeline at all** — they hang off a parallel advisory CLI surface. The one LLM that *is* wired (Gemma) is boxed to 2 recipes, can never override a deterministic safety-critical plan, and can never introduce a fact (`hasValidatedContradictoryFacts` is hardcoded `return false`).

So against the aspiration: **"提炼" ≈ 70% done (rich genome exists), "极致拓扑进内核" ≈ 20% done.** The distilled intelligence is sitting next to the binary as documentation and an advisory lookup, not inside the decision loop that fixes machines.

This is the same pattern flagged in the Qwen harvest report (`QWEN_GEM_FREE_RUN_CHINA_NETWORK_RESCUE_CAPABILITY_HARVEST.md` §12): the bottleneck is the kernel's thin ingest + thin repair surface, not the available knowledge.

---

## 1. What the runtime kernel actually is

Real execution path of `agentlink orchestrator rescue` / GUI "Fix My Connection" (`orchestrator.go:33` `Run`):

```
diagnose.Engine.Run        ← real macOS commands (scutil/networksetup/ifconfig/route/netstat/ping/curl/git/npm/brew/profiles/systemextensionsctl)
   → classify.Apply         ← 613 LOC hand-coded if-rules → 29 flat class strings, no confidence
   → DiagnoseTun            ← TUN heuristic
   → [protected topology?]  → refuse, support bundle              (good, safe)
   → brain.Available?       → gemmaSupervise (llama-cli, RescuePlanDecision JSON)
        → arbitratePlans    ← deterministic ALWAYS wins safety-critical; Gemma can't add facts
   → ValidatePlan
   → decision.Intent==repair AND SelectedRecipe=="macos-clash-tun-force-repair"?
        → repair.Run(LevelTun)   ← the ONLY auto-executed repair
     else → manual_action_required / support bundle / terminal ticket / restart gate
```

Three structural facts from the code:

1. **One-recipe autonomy.** `orchestrator.go:175`: any selected recipe other than `macos-clash-tun-force-repair` → `manual_action_required`. `:194`: the execute call is hardcoded `repair.LevelTun`. The deterministic `supervise()` (`:413`) only ever emits: TUN-repair, OK-report, or `manual_action UNKNOWN`. There is **no orchestrator branch** for proxy-dirty, DNS, dev-tool, residue, MDM, captive-portal, or protected-route-rehoming — even though `classify` detects several of them.

2. **Genome / diagnosisgraph / planner are out of the loop.** `orchestrator.go:3-19` imports `brain, classify, command, diagnose, networkverify, repair, restartgate, system, ticket`. It does **not** import `genome`, `diagnosisgraph`, or `planner`. The genome is read only by `cli/genome.go` (a symptom-lookup advisory: `agentlink diagnose --symptom "..."`). `diagnosisgraph` is read only by the genome CLI. `planner` is used only by `agentlink planner validate <file>` (an offline JSON validator). None touch the repair decision.

3. **The LLM is wired but architecturally near-mute.** Gemma *is* called (`gemmaSupervise` → `backend.Generate`, `orchestrator.go:256-310`) and there is a real arbitration layer (`arbitratePlans` :321). But: allowed recipes = 2; `hasValidatedContradictoryFacts()` permanently returns `false` (:406-411, "v0.5.0 does not let Gemma introduce new facts"); safety-critical always retains deterministic (:337-352). Net authority of the model = optional commentary + (only when deterministic is non-repair or <0.55 confidence) picking a repair the deterministic planner already could have. It cannot expand the repair surface.

---

## 2. Coverage data (evidence-cited)

### 2a. Failure-class → recipe coverage
`classify.go` defines **29 classes**; `Classify()` can emit **23**; **6 are dead constants** never `add()`-ed: `NETWORK_EXTENSION_SESSION_STALE`, `CLASH_CLEAN_REINSTALL_REQUIRED`, `RESTART_GATE_REQUIRED`, `NETWORK_LOCATION_SUSPECTED`, `SYSCONFIG_SUSPECTED`, `UNKNOWN`.

Of the 13 recipes, only these map to network classes:

| recipe | classes | risk |
|---|---|---|
| macos-clash-tun-force-repair | CLASH_TUN_ACTIVE_OR_STALE, NETWORK_EXTENSION_SESSION_STALE†, TUN_ROUTE_OWNERSHIP_SUSPECTED, AIRDROP_DISCOVERY_DEGRADED | privileged |
| macos-clean-network-baseline-reset | RESTART_GATE_REQUIRED†, CLASH_CLEAN_REINSTALL_REQUIRED†, NO_DEFAULT_ROUTE, RAW_IP_UNREACHABLE, HTTPS_FAIL | privileged |
| npm-git-proxy-conflict-repair | GIT_PROXY_DIRTY, NPM_PROXY_DIRTY | reversible |
| proxy-clean-stale-env | USER_PROXY_DIRTY | reversible |

† = dead routing key (the class is never emitted, so the recipe can never be selected via it).

**Classified-but-no-repair (ORPHAN) classes — 22 total**, including high-frequency real cases:
- `SYSTEM_PROXY_DIRTY` — the single most common mainland case (dead Clash leaves system proxy pointing at 127.0.0.1:7890). **No recipe.** Qwen harvest cn02/m02.
- `DNS_FAIL`, `GATEWAY_UNREACHABLE`, `LINK_LOCAL_ONLY`, `NO_DHCP_LEASE`, `NO_ACTIVE_INTERFACE`, `BREW_PROXY_DIRTY`
- `KNOWN_AGENT_RESIDUE`, `NETWORK_EXTENSION_SUSPECTED`, `MDM_PROFILE_SUSPECTED`
- `GENERAL_INTERNET_OK_AGENT_ENDPOINT_BLOCKED` (the api.openai/anthropic block — Qwen harvest m09/cn05)
- `PROTECTED_AUDIO_VLAN_ROUTE_TRAP`, `PROTECTED_TOPOLOGY_CONSTRAINT` (correctly refuse-only by design — acceptable as "no mutation", but no *rehome-internet-to-safe-interface* recipe exists either, which the Qwen run showed is the actual fix for m13/cn04)

### 2b. Dead input fields (collected, never read by classify)
90+ DiagnosticReport fields; **20+ never consumed**: all `HostInfo` (arch/macOS ver/user), `NetworkInfo.CurrentLocation`, `Services`, `DNSSummary.Nameservers/Search/Raw`, `ProxySummary.Exceptions/Raw`, `DefaultRoute.Raw`, and on every `ProbeResult`: `Status/HTTPStatus/Error/DurationMS/RedactedInfo`. Consequence: the kernel cannot see *which IP DNS returned* (no cn01 DNS-poison detection) or *the HTTP status / TLS error string* (no m09 403-vs-RST or cn05 SNI-reset distinction). This is exactly the schema gap the Qwen advisory exploited.

### 2c. Package test coverage
19,283 src LOC / 7,283 test LOC (~38%). Zero-test packages: `facts`, `interference`, `networkverify`, `readiness`, `verify`. `verifier` has 565 src / 30 test LOC (thin given it "owns truth" per Constitution rule 12). `cli` 2,974 src / 152 test.

---

## 3. What is genuinely good (don't regress these)

The safety topology is strong and should be preserved through every change below:
- Constitution + Core Reliability invariants are real in code: dry-run default (orchestrator `mode()`), Terminal-ticket instead of GUI sudo (`:185-192`), auto-rollback on postflight-worsening (`:209-211`), restart-gate when NetworkExtension state won't release (`:214-219`), durable transaction journal, 0600/0700 private artifacts.
- Protected-topology refusal is correct and early (`:81-88`), and the Gemma arbiter cannot relax it (`isProtectedBoundaryPlan` :332-336). The Qwen harvest independently confirmed 3/3 protected cases handled.
- The deterministic-wins-safety-critical arbitration is the right default for a field tool.
- The genome itself is high quality as a knowledge asset (300 cards, ontology, volumes) — the problem is wiring, not content.

A capability expansion that breaks any of these is a net loss (per `feedback_dont_over_govern_specialist_agent` / `feedback_listen_dont_hide_behind_metrics`).

---

## 4. All improvement items (prioritized)

### P0 — Topology: wire the distilled brain into the kernel
The headline gap. Until this is closed the aspiration is unmet regardless of other work.

1. **P0.1 Make the genome the source of truth for classification, or delete the duplication.** Today `classify.go` (29 hand-coded classes) and the genome (300 `MAC-*/DEV-*/AUDIO-*` cards, 15-layer ontology) are two unsynced taxonomies with zero shared identifiers. Either (a) generate `classify` rules/classes from the genome cards (codegen, single source), or (b) have the runtime load the genome DAG and traverse it. Current state guarantees drift: a card added to the genome never reaches the repairer.
2. **P0.2 Wire `diagnosisgraph` (the Go DAG with confidence/edges) into the pipeline.** It already exists with the right shape and is only consumed by the genome CLI. Replace/augment the flat `classify` emission with DAG traversal so the kernel produces ranked hypotheses with confidence, not a flat unordered class set. This is the structural prerequisite for "topology", which currently means "order of if-statements".
3. **P0.3 Replace the hardcoded single-recipe execute gate with class→recipe routing.** `orchestrator.go:175` should look up *the recipe registry* by emitted class, not string-equal one recipe. The registry + `planner.ValidateDecision` already exist; the orchestrator just doesn't call them. This unlocks every recipe that already passes validation without weakening the validator/verifier gates.

### P1 — Coverage: give the 22 orphan classes a repair path
4. **P1.1 `SYSTEM_PROXY_DIRTY` repair recipe** (reversible: snapshot current `networksetup -getwebproxy/-getsecurewebproxy`, then `-setwebproxystate off` / `-setsecurewebproxystate off`, rollback restores prior host:port). Highest real-world frequency; today has no recipe. Must keep Apple/iCloud-DIRECT note (Qwen insight).
5. **P1.2 `BREW_PROXY_DIRTY` recipe** (mirror of the npm/git one; trivial extension of `npm-git-proxy-conflict-repair`).
6. **P1.3 `DNS_FAIL` recipe** (reversible: snapshot `networksetup -getdnsservers`, offer non-ISP resolver set 223.5.5.5/1.1.1.1/8.8.8.8 with rollback; flush mDNSResponder). Gate behind "not MDM-managed".
7. **P1.4 Protected-route rehome recipe** for `PROTECTED_AUDIO_VLAN_ROUTE_TRAP` / `PROTECTED_TOPOLOGY_CONSTRAINT`. Today these are refuse-only. The actual fix (Qwen m13/cn04) is *move the default route to the internet-candidate interface while leaving the protected interface untouched* — a bounded, reversible `route` change scoped to the non-protected interface. This is the single biggest capability win for the pro-audio segment and the genome already has the topology to do it safely.
8. **P1.5 `GENERAL_INTERNET_OK_AGENT_ENDPOINT_BLOCKED` handler** — not a mutation; a structured *report* intent (today it falls to generic manual_action). Classify already detects it; emit a dedicated "upstream block, escalate to proxy/operator" decision (Qwen m09/cn05).
9. **P1.6 Resolve the 6 dead constants**: either emit them (`RESTART_GATE_REQUIRED`, `CLASH_CLEAN_REINSTALL_REQUIRED`, `NETWORK_EXTENSION_SESSION_STALE`, `NETWORK_LOCATION_SUSPECTED`, `SYSCONFIG_SUSPECTED`) from real detectors, or delete them and their dead recipe routing keys. Right now two privileged recipes route off classes that can never fire.

### P2 — Ingest surface: stop discarding evidence
10. **P2.1 Capture DNS-returned IPs.** Add resolved-IP to `DNSSummary.Resolution` / `ReachabilityInfo.DNSNames` and a classify rule comparing against a suspect-IP set (198.18.0.0/15 outside TUN context, 0.0.0.0, 127.0.0.1, known-pollution markers) → new `DNS_POISON_SUSPECTED` class. Unlocks cn01.
11. **P2.2 Read `ProbeResult.HTTPStatus` + `Error`.** Add classify rules: 403/401 → auth-vs-network split; `connection reset ... after ClientHello` → new `SNI_RESET_SUSPECTED` class (repair=none/handoff). Unlocks m09/cn05. Fields are *already collected* — pure classify work.
12. **P2.3 Dev-tool config block depth.** `UserConfig` has git/npm/brew proxy; add per-scope origin (global/system/repo) and stale-vs-live discrimination (probe whether the proxy port is listening). Unlocks the cn03 orthogonal-failure case. Add cargo/pip/conda/poetry while here.
13. **P2.4 Process/daemon liveness.** Snapshot whether clash/mihomo/surge/v2ray is actually listening (distinguishes "proxy stale because daemon dead" from "config bug, daemon alive"). Today inferred indirectly. Unlocks cleaner cn02.

### P3 — Multi-fault + composition
14. **P3.1 Parallel reversible composition.** cn03 had two independent faults (git proxy + npm mirror). The orchestrator runs one repair-tier per rescue. Allow independent reversible recipes to run side-by-side, each with its own verifier + rollback, with an operator gate between privileged ones.
15. **P3.2 Per-hypothesis confidence in the report.** Once P0.2 lands, surface ranked (class, confidence, evidence) to the GUI/report so the operator sees triage, not a flat list.

### P4 — Model authority (ties to the 64GB Qwen brain decision)
16. **P4.1 Decide Gemma's real role.** Today Gemma is wired but architecturally muted (can't add facts, 2 recipes). Either (a) keep it honest commentary-only and *say so* in docs (stop implying model-driven), or (b) let validated model output expand the *probe* set (read-only, can't hurt) — a safe first widening that the current `hasValidatedContradictoryFacts==false` blocks entirely.
17. **P4.2 Advisory pane, not planner (from the Qwen harvest).** Wire the 64GB Qwen Gem as a read-only advisory side-surface (harvest report §16 R1), distinct from Gemma-the-arbiter. Do NOT route either model into execution before P2 schema expansion + verifier-delta testing.
18. **P4.3 Genome-grounded model prompts.** When a model *is* consulted, feed it the relevant genome cards (retrieval the genome CLI already does) instead of just the flat facts payload. This is the only place the 300 cards would touch the live decision.

### P5 — Verification + hygiene
19. **P5.1 Thicken `verifier`** (565/30 test LOC). It "owns truth" (Constitution 12) but is the least-tested critical package. Add per-verifier three-state tests (pristine-fail / repaired-pass / wrong-route-fail) mirroring the chaos maze discipline.
20. **P5.2 Verifier-delta harness.** No test currently proves a *repair actually flips its verifier* on a simulated post-state. Add: apply recipe to a fixture-mutated state, assert verifier transitions fail→pass. (Same gap named in the Qwen harvest §17.)
21. **P5.3 Cover the zero-test packages**: `facts`, `interference`, `networkverify`, `readiness`, `verify`.
22. **P5.4 Expand chaos mazes** per the doc backlog: Bluetooth PAN, 802.1X campus, same-subnet multi-home, captive-portal edge, plus sibling fixtures where symptom is identical but correct corridor differs (the discriminating-probe cases) — and the 5 China fixtures from the Qwen harvest (cn01-05) once their schema fields (P2) exist.
23. **P5.5 Genome→runtime consistency test.** Once P0.1 lands, a CI check that every genome card class has a classify path and (where mutation is intended) a recipe — preventing re-divergence.

---

## 5. Optimality scorecard

| dimension | state | note |
|---|---|---|
| Safety topology | ★★★★★ | dry-run/ticket/rollback/restart-gate/protected-refusal all real |
| Diagnostic ingest breadth | ★★☆☆☆ | 90+ fields but 20+ dead; can't see DNS-IP/HTTP-status |
| Classification intelligence | ★★☆☆☆ | 613 LOC flat if-rules; no confidence, no DAG, ignores genome |
| Repair coverage | ★☆☆☆☆ | autonomous = 1 recipe; 22/29 classes orphan |
| Knowledge integration | ★☆☆☆☆ | 300-card genome + DAG + planner all out of the loop |
| Model leverage | ★★☆☆☆ | Gemma wired but muted; 64GB Qwen not connected |
| Verification rigor | ★★★☆☆ | strong gates, thin verifier tests, no verifier-delta |
| Test coverage | ★★★☆☆ | 38% overall, 5 zero-test pkgs |

"Distill ALL top-model capability + extreme topology": **distillation strong, topology-into-kernel weak.** Aspiration not yet met.

---

## 6. Recommended sequencing

1. **P2.1–P2.4 first** (ingest) — cheapest, pure additive, unlocks the China cases, and is the prerequisite that makes model advisory honest (the Qwen wins were schema-gap wins).
2. **P1.1–P1.6** (recipes for orphan classes) — direct capability, each reversible + verifier-gated, no architecture risk.
3. **P0.3** (class→recipe routing) — flips the one-recipe gate to use the registry; medium risk, high payoff; gate behind existing validator.
4. **P0.1–P0.2** (genome/diagnosisgraph as source of truth) — the real architectural fix; do after coverage exists so the DAG has something to route to.
5. **P5.1–P5.5** (verification) — in lockstep with each capability add, not after.
6. **P4** (model role) — last; only after P2 + verifier-delta exist, per the Qwen harvest hard gate (≥13/16 Lane-A class match on enriched schema before structured coupling).

---

## 7. Attribution / confidence

- **Read in full**: `classify.go`, `orchestrator.go`. **Mapped by agents**: genome wiring, input-surface, coverage matrix.
- **Corrected**: a breadth agent claimed "orchestrator imports none of brain/planner" — false; `orchestrator.go:10` imports `brain` and calls it. Gemma *is* wired (caged). The "out of the loop" finding applies to `genome`, `diagnosisgraph`, `planner` — confirmed by import grep.
- **Inferred, not run**: `go test ./...` (go not on PATH). FIELD_BETA_STATUS reports the suite + chaos green + ~15.6M fuzz mutations no-panic as of the hardening commit; chaos behavioral suite per user already re-run this session. I personally confirmed `chaos run m02` classifies `SYSTEM_PROXY_DIRTY/safe` correctly.
- **Not a correctness critique**: on the cases it *does* cover, the kernel is correct and safe. Every item above is about *coverage breadth* and *knowledge-into-kernel integration*, not bugs in existing paths.
- Single-reviewer; recommend a Codex/GPT-Pro second pass on P0.1–P0.3 (the architectural items) before committing to the genome-as-source-of-truth refactor.
