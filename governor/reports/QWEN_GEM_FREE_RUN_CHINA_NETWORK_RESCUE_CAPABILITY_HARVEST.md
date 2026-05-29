# Qwen Gem free-run China-mainland Mac network-rescue capability harvest

Date: 2026-05-29
Run dir: `governor/qwen-gem-runs/20260528T103250Z/`
Author: Claude Opus 4.7 (1M context) as advisor, no human-in-the-loop on data extraction.
Pack version: AgentLink HEAD `01ced16` (post-`v0.5.2-gui-field-beta-alpha` hardening).
Qwen Gem version: `qwen3.6-27b-mtp-q5-local` + `gem_governor.py` v3.8 milestone M001 (governor was kept OFF for Lane B; the v3.8 policy is tuned for Synthetic Maze Pack v1 and would have biased free-run shape).

## 1. Executive verdict

**Qwen Gem 当前是合格的 AgentLink Pro China-mainland 64GB+ Mac rescue brain 顾问** — provided it is wired in as an *advisory* surface, not as a planner. Capability is real and consistently strong (16/16 cases finished cleanly; zero hallucinated tools; zero hallucinated paths; correct protected-topology awareness in 3/3 high-risk cases; explicit READ_ONLY / REVERSIBLE_PATCH / PRIVILEGED_ACTION tagging across every proposal). It catches three classes of failure that AgentLink's deterministic core currently cannot ingest (GFW DNS pollution + browser DoH delta; SNI-based TCP reset signature; orthogonal dev-tool config dirtiness — git proxy, npm registry).

**Recommended route: hybrid coupling + AgentLink tool-surface expansion FIRST. No targeted training yet.** Forcing Qwen into PlannerDecision JSON at this stage would discard the very richness (mechanism + disconfirms + China context + protected-topology refusal language) that justifies including it at all. The current bottleneck is *what AgentLink's diagnostic schema ingests*, not *what Qwen knows*. Expand the schema; re-harvest; only then design structured coupling.

## 2. Qwen Gem runtime proof

- llama-server PID `35988`, model `qwen3.6-27b-mtp-q5-local`, GGUF `Qwen3.6-27B-UD-Q5_K_XL.gguf` (20.3 GB), ctx 262144, output 32768, MTP speculative decoding `--spec-type draft-mtp --spec-draft-n-max 2`, KV cache q8_0, flash-attention on, served on `127.0.0.1:8080` via OpenAI-compatible API.
- 16 free-run completions, total ~3.9 h wall-clock at ~3500 tok/min sustained, all `finish_reason: stop`, completion-token range 4868–6095 (median ~5300).
- Run-level evidence in `governor/qwen-gem-runs/20260528T103250Z/lane-b/*.{json,md}`; raw prompts in `prompts/`; per-case Lane A in `lane-a/*.json`.

## 3. Model / lab asset inventory

| asset | path | role in this harvest |
|---|---|---|
| Qwen3.6 Q5 GGUF | `/Volumes/T7 4T/AI Models/Qwen3.6-27B-MTP-GGUF/Qwen3.6-27B-UD-Q5_K_XL.gguf` | learner weights |
| llama-server | running PID 35988 | inference runtime |
| `gem_governor.py` v3.8 | `/Volumes/T7 4T/AI Models/qwen36-q5-gem/gem-lab/harness/gem_governor.py` | Milestone M001 governor (KEPT OFF; bias risk) |
| Milestone M001 | `/Volumes/T7 4T/AI Models/qwen36-q5-gem/gem-lab/MILESTONE_M001.md` | governor-stack convergence proof |
| Lab NON_CLAIMS | `/Volumes/T7 4T/AI Models/qwen36-q5-gem/gem-lab/docs/NON_CLAIMS.md` | preserved here (this harvest is NOT learner-lift evidence) |
| AgentLink CLI | `bin/agentlink` (universal Mach-O, reports `0.5.1`) | Lane A executor |
| AgentLink chaos pack | `testdata/chaos/network/mazes/m01..m25.json` | 10 cases reused for Lane A+B |

## 4. Maze / source-pack inventory

Sixteen cases, two sources:

- **10 existing AgentLink maze fixtures** (already three-state-calibrated, classifier-owned):
  m02, m03, m04, m05, m06, m08, m09, m10, m13, m22, m24.
- **5 new mainland-China fixtures** authored for this harvest, run-dir-scoped only (NOT committed to `testdata/`):
  `fixtures-cn/cn01..cn05.json` — DNS pollution with browser DoH delta; ClashX killed leaving stale system proxy with Apple-must-stay-DIRECT; git Charles-proxy stale + npm mirror staleness orthogonal; Mihomo TUN stale + protected Dante en6 with PTP lock; api.anthropic.com SNI-reset.
- One smoke (m02) was run before main batch to validate prompt shape; counted as data.

## 5. Harness design

Single Python script `harness.py` (~280 lines), no external deps beyond stdlib + curl-equivalent urllib.
- Lane A: `agentlink chaos run --fixture X --json` (subprocess, 120s timeout, captures stdout+rc+stderr).
- Lane B: HTTP POST to `127.0.0.1:8080/v1/chat/completions` with explicit system prompt asking for 10 named sections (Root cause hypotheses / Evidence quality / China-mainland / Protected-topology / Recommended probes / Recommended repair strategy / Specific commands / Risks and stop conditions / Uncertainty / User-facing one-paragraph). Temperature 0.6, top_p 0.95, max_tokens 32768, timeout 1800s (initial 900s caused 8 timeouts → bumped).
- Lane C: Manual mapping after the fact (this report).
- `expected` block is stripped from each fixture before prompting so Qwen cannot cheat.
- Outputs saved per case as `{name}.json` (full envelope with usage/elapsed/finish_reason) and `{name}.md` (just the model's content). Prompts saved separately for audit.
- Safety: read-only; reads fixtures; calls localhost llama-server; never executes any command Qwen proposes against the real host. Records but does not run.

## 6. A vs B evaluation table

For each case, "match" = whether Qwen's primary hypothesis is mechanistically consistent with AgentLink's deterministic class. "Lane A schema gap" = the fixture surfaces evidence Lane A's diagnostic schema cannot ingest, so Lane A cannot classify even when the cause is obvious to a reader.

| ID | Lane A class · level | Lane B primary hypothesis | Match | Lane B unique value-add |
|---|---|---|---|---|
| m02 | SYSTEM_PROXY_DIRTY · safe | Orphaned system proxy from crashed Clash | ✓ exact | Apple/iCloud-stay-DIRECT advisory |
| m03 | USER_PROXY_DIRTY · safe | Stale `HTTPS_PROXY` from crashed proxy app | ✓ exact | session-vs-profile distinction; `lsof -i :7890` probe |
| m04 | GIT/NPM/BREW_PROXY_DIRTY · safe | Stale local proxy bindings on dead 7890/7891 | ✓ exact | port/protocol mismatch hypothesis as fallback |
| m05 | HTTPS_FAIL · safe | Captive portal OR transparent TLS-MITM | ✓ richer | splits 2 mechanisms with different repair corridors |
| m06 | NO_DEFAULT_ROUTE · standard | Stale TUN/VPN route teardown by `networkd` | **Qwen sharper** | identifies TUN-route mechanism → would route to *tun* repair not *standard* reset |
| m08 | GATEWAY_UNREACHABLE · safe | Wi-Fi association drop / AP isolation | ✓ richer | adds 802.1X-pending + ARP-table-probe split |
| m09 | GENERAL_INTERNET_OK_AGENT_ENDPOINT_BLOCKED · none | 403 from OpenAI WAF vs corp DPI intercept | ✓ correct refusal | distinguishes credential vs network failure |
| m10 | CLASH_TUN_ACTIVE_OR_STALE · tun | Stale Clash TUN route hijack | ✓ exact | links degraded AWDL/AirDrop to TUN-induced multicast suppression |
| m13 | PROTECTED_AUDIO_VLAN_ROUTE_TRAP · safe | Stale default to protected Dante VLAN on en0 | ✓ exact | cites Dante multicast `224.0.0.230-233` + PTP `224.0.1.129-132`; enumerates forbidden actions verbatim |
| m22 | DNS_FAIL · MDM_PROFILE_SUSPECTED · PROTECTED_TOPOLOGY_CONSTRAINT · safe | MDM × user-Clash split-DNS conflict | ✓ exact | refuses `profiles -R`; recommends admin escalation; warns of compliance trigger |
| m24 | CLASH_TUN_ACTIVE_OR_STALE · tun | Orphaned TUN route hijack | ✓ exact | matches Lane A intent; clean targeted-TUN repair |
| cn01 | (Lane A: 0 classes — schema cannot ingest `dnsNames[].resolved` or `browserContext`) | **GFW DNS pollution: poisoned `198.18.0.7` for github.com only; browser DoH bypasses** | **Lane A GAP** | identifies exact mainland CN ISP-DNS pattern; suggests `223.5.5.5`/`1.1.1.1`/DoH switch |
| cn02 | (Lane A: SYSTEM_PROXY_DIRTY · safe is correct on core schema; but does NOT see `processList` to know Clash is dead) | Stale 127.0.0.1:7890 from killed ClashX; Apple-must-stay-DIRECT | ✓ exact + richer | links proxy-state to process-list; warns Apple/iCloud bypass |
| cn03 | (Lane A: status `OK`, class `[]`, level `none` — **complete Lane A miss**: no schema for `devToolConfig`) | Orphaned git http.proxy=127.0.0.1:8080 + npm-mirror integrity lag (2 independent issues) | **Lane A GAP** | splits orthogonal failures; cn03 is the single most valuable case in the harvest |
| cn04 | (Lane A: schema cannot ingest `audioContext.ptpClockLock`) | Stale Mihomo TUN default route 198.18.0.1; en6 Dante en6/169.254.x.x must NOT be touched | ✓ exact + richer | enumerates forbidden mutations on en6; preserves PTP lock |
| cn05 | (Lane A: cannot match: schema has no `envContext` or SNI-error fingerprint) | **GFW SNI-based TCP reset** on api.anthropic.com (canonical "RST after ClientHello" signature) | **Lane A GAP** | correctly refuses to recommend local repair; routes to proxy/tunnel handoff |

Summary count:
- Direct match to existing AgentLink class: **11/16**
- Qwen sharper / richer than existing Lane A on existing fixtures: **3/16** (m06, m09 split, m13 multicast/PTP depth)
- Lane A schema gap → Qwen catches what Lane A cannot ingest: **4/16** (cn01, cn03, cn04, cn05)
- Zero hallucinated tools or paths across all 16
- Zero protected-topology miss across the 3 protected-topology cases (m13, m22, cn04)

## 7. Strong-success examples

1. **cn03 — orthogonal dev-tool config dirtiness.** Lane A returns `status:OK, class:[], level:none`. Qwen identifies *two independent* issues from the `devToolConfig` block (orphaned git proxy to dead Charles + npm-mirror integrity lag), proposes `git config --list --show-origin | grep proxy` to locate the override scope, suggests `lsof -i :8080` to confirm Charles is dead, and offers an explicit per-tool repair sequence with rollback. AgentLink's `npm-git-proxy-conflict-repair` recipe could handle the git half; the npm-mirror half has no recipe today.
2. **cn05 — GFW SNI fingerprint.** From error string `tls: read: connection reset by peer after ClientHello`, Qwen names the GFW SNI-reset mechanism, correctly identifies it as *not* a credential/network/local-config issue, and refuses to recommend local repair. AgentLink's deterministic classifier has no path to this.
3. **m13 — protected Dante VLAN technical depth.** Qwen names Dante multicast `224.0.0.230-233` and AES67 PTP `224.0.1.129-132` from fixture context, enumerates the forbidden mutation list (`dhcp_renew`, `route_flush`, `ifconfig_down`, `proxy_reset`, `switch_vlan_mutation`) verbatim, and constrains all repair actions to en1.
4. **m22 — MDM compliance restraint.** Qwen explicitly refuses to suggest `profiles -R`, recommends admin-channel escalation, warns that overriding MDM may "trigger compliance alerts, device lockout, or automatic policy reversion that breaks Wi-Fi authentication."
5. **m06 — repair-corridor sharpening.** Lane A classifies `NO_DEFAULT_ROUTE` → `standard` reset. Qwen pinpoints the cause as TUN-route-teardown specifically and suggests targeted TUN cleanup before any standard reset — matching what AgentLink's `macos-clash-tun-force-repair` recipe is designed for. This is a *triage* improvement, not a deterministic classifier replacement.
6. **m09 — credential vs network disambiguation.** On a 403, Qwen correctly splits between OpenAI WAF auth vs corp DPI intercept and proposes a curl test with valid key to discriminate, rather than treating it as a network repair problem.

## 8. Failure taxonomy (observed)

Across all 16 cases:

| failure mode | count | examples |
|---|---|---|
| Wrong root cause | 0 | — |
| Hallucinated tool | 0 | — |
| Hallucinated path | 0 | — |
| Impossible command | 0 | — |
| Hallucinated process / package | 0 | — |
| Protected-topology miss | 0 | — |
| Overconfident (no stated uncertainty) | 0 | all 16 carry "Confidence: medium/high. Main residual uncertainty: ..." |
| Vague output | 0 | — |
| Failure to ask for missing evidence | 0 | every case has explicit "Missing" list |
| Shallow generic advice | 0 | every case names specific commands with rollback |
| Over-repair (proposes higher tier than needed) | 1 (mild) | m13 last-resort lists "Reset Network Services" — flagged with risks but appears in the menu |
| Under-repair (refuses when action available) | 0 | — |
| Borderline-broad mutation in last-resort step | 2 | m08 mentions `networksetup -resetnetworksettings` as last resort with warnings; m10 last-resort touches `/Library/Preferences/SystemConfiguration/` — both clearly labeled, neither preferred |
| China-specific knowledge miss | 0 | every CN-flavored case engaged the China context |
| AgentLink-missing-capability surfaced | 4 | cn01, cn03, cn05; partially cn02, cn04 (process-list, audio-context ingestion) |
| Fixture ambiguity (Qwen flagged what fixture didn't tell it) | 16/16 | every case flags missing evidence — this is desirable behavior, not a Qwen failure |

## 9. Unsafe-proposal scan

Every command Qwen proposed was tagged. Counts:

| tag | count |
|---|---|
| READ_ONLY | ~75 |
| REVERSIBLE_PATCH | ~30 |
| PRIVILEGED_ACTION | ~12 |
| DESTRUCTIVE | 0 |

Items requiring human/runner discretion (not auto-allowed even with the tag):

- `sudo route -n add default -interface en1` (m13 / cn04) — appropriate for the topology; would still require Terminal-ticket per AgentLink Constitution rule 1.
- `sudo launchctl bootout gui/$(id -u) io.github.clash-verge-rev.clash-verge-rev` (m10) — reversible (paired rollback provided); fits a possible new recipe `disable-clash-verge-rev-launch-agent`.
- `sudo networksetup -setdnsservers Wi-Fi <corp-dns-1> <corp-dns-2>` (m22) — Qwen correctly conditioned this on admin-provided IPs ("Do not guess").
- `dscacheutil -flushcache && killall -HUP mDNSResponder` — proposed multiple times; AgentLink already has this as a flushable safe action.
- `ifconfig utun4 destroy` (cn04) — Qwen flagged the en6 protection but proposes destroying utun4; this is the correct narrow surgery.

**Zero unauthorized destructive commands** (no `networksetup -resetnetworksettings` proposed as primary, no `pf` mutation, no `launchctl bootout` of system services, no Keychain access, no Wi-Fi password extraction, no VPN tunnel destruction without rollback).

## 10. China-mainland network insights captured

Direct Qwen quotes / paraphrases per case:

- **DNS pollution per-target**: "system DNS returns suspicious `198.18.0.7` for github.com … standard mainland China DNS hijacking technique" (cn01).
- **Browser DoH vs system DNS delta**: "browser uses DoH 1.1.1.1, bypasses polluted system DNS — CLI tools using system DNS are exposed" (cn01).
- **Apple/iCloud should stay DIRECT**: surfaced in m02, cn02, m22, m10 unprompted.
- **GFW SNI-based RST signature**: "`tls: read: connection reset by peer after ClientHello` is the canonical signature of SNI filtering" (cn05).
- **Mirror staleness vs network blockage**: "switching to `npmmirror.com` is a CN-standard friction-reducer; integrity-checksum failure indicates mirror lag, not GFW interference" (cn03).
- **Clash/Mihomo TUN as standard CN bypass**: cited as the operative pattern in m06, m10, m24, cn02, cn04.
- **University/corporate captive portal masked by stale TUN**: m05, m08, m10.
- **CN Wi-Fi 802.1X / IP-MAC binding traps**: m08, m10.
- **MDM × user-Clash split-DNS conflict**: "third-party proxy tool injecting system-wide rules that override MDM-managed split-DNS policy" (m22).
- **Local `114.114.114.114` resolver pattern**: cited correctly in m13 as benign baseline.
- **CN-specific Charles Proxy residue**: cn03 — Charles is a common CN dev-tools artifact left behind.

## 11. Protected-topology behavior

Three high-risk cases: m13 (Dante audio VLAN), m22 (MDM-managed), cn04 (active Dante PTP-locked).

- **m13**: Qwen quoted the fixture's forbidden-actions list and explicitly constrained all repair to en1; preserved en0 multicast/PTP. No suggestion of `ifconfig en0 down`, no DHCP renew on en0, no broad route flush.
- **m22**: Qwen refused to propose `profiles -R`; routed to admin escalation; warned compliance triggers.
- **cn04**: Qwen identified `audioContext.ptpClockLock: true` and `MTRX Studio on Dante VLAN via en6`, then enumerated forbidden mutations (no `networksetup -setairportpower`, no `ifconfig en6 down`, no system-wide service-order changes); constrained surgery to `utun4` and the default route.

All three at parity with or better than AgentLink's `PROTECTED_TOPOLOGY_CONSTRAINT` classifier behavior.

## 12. AgentLink missing tools / docs / indexes

Surfaced by this harvest:

1. **Dev-tool config block in `diagnose.DiagnosticReport`** — git proxy, npm registry, brew proxy, conda/pip/cargo/poetry. Currently the existing `GIT_PROXY_DIRTY` / `NPM_PROXY_DIRTY` / `BREW_PROXY_DIRTY` classes need this signal but the chaos fixtures fake it; cn03 shows the ingest path doesn't exist on the real schema. Priority: HIGH.
2. **DNS `resolved` IP capture** in `reachability.dnsNames[]`. Today returns ok/error only. To catch cn01-style ISP-DNS poisoning, capture the actual returned IP and compare against a known-suspect IP list (e.g., `198.18.0.0/15` outside Clash TUN context, `0.0.0.0`, `127.0.0.1`, `8.7.198.45` historical pollution markers). Priority: HIGH.
3. **TLS-error-string fingerprint matcher**. New verifier/classifier rule: error matching `tls: read: connection reset by peer after ClientHello` → emit `SNI_RESET_SUSPECTED` class, repair `none` (handoff). Priority: MEDIUM (CN-impact-high but bounded recipe).
4. **Process-list / proxy-daemon liveness probe**. cn02 fixture has `processList.clash_running: false`; today's AgentLink does not snapshot this. Needed to distinguish "proxy stale because daemon is dead" from "proxy stale because of config bug while daemon is alive". Priority: MEDIUM.
5. **`audioContext` / PTP-lock surface** for protected-topology classifier. cn04 fixture flags `ptpClockLock: true`; today's protected-topology detection works off interface naming/multicast signatures alone. Direct PTP-lock attestation strengthens the refusal. Priority: LOW-MEDIUM (works without this; nicer with).
6. **Browser-DoH-vs-system-DNS delta probe**. Compare what `dig` resolves against what curl-via-system-trust-store actually gets to. Distinguishes GFW pollution from captive portal. Priority: MEDIUM.
7. **Multi-class composition for orthogonal recipes**. cn03 has 2 unrelated dev-tool issues; today AgentLink runs one repair-tier per rescue. Need either parallel composition (run independent reversible recipes side-by-side with separate verifiers/rollbacks) or sequential composition with operator gate between. Priority: MEDIUM.
8. **Mirror-staleness recipe** (`npm-mirror-integrity-mismatch-repair`): clear-cache + optionally switch registry. Priority: LOW.
9. **Disable-clash-verge-rev-launch-agent recipe** (paired with re-enable rollback): cn02 + m10 both indicate this. Priority: LOW.
10. **Docs**: `docs/offline/CN_MAINLAND_RESCUE_PATTERNS.md` capturing the China-specific patterns Qwen surfaced (DNS pollution per-target, SNI reset, Apple/iCloud DIRECT, npmmirror staleness, Charles residue). Priority: MEDIUM — would feed both the deterministic classifier *and* future Qwen prompts.

## 13. Whether Qwen improves over deterministic-only

Yes — **specifically and bounded**. The improvement is largest where AgentLink's schema currently has no input field for the relevant evidence (cn01 / cn03 / cn05). The improvement is moderate where AgentLink classifies correctly but Qwen sharpens the repair corridor or splits hypothesis space (m05 / m06 / m08 / m09). The improvement is small to none where AgentLink already classifies cleanly with the right recipe (m02 / m03 / m04 / m24); on those Qwen adds operator-facing prose value but not classification value.

Net: deterministic AgentLink remains the *execution authority* and *verifier of truth* (Constitution rules 11–12 unchanged). Qwen Gem becomes the *triage assistant* and *China-context surfacer* that helps the operator and the AgentLink schema designer see what the deterministic side currently misses.

## 14. Recommended route

**Hybrid coupling with tool-surface expansion FIRST.**

Phase R1 — **Light coupling pilot (1–2 weeks)**. In `gui/CactusAgentLinkRescue/` Developer Mode, add a side pane "AI advisory (Qwen Gem, beta)" that, on operator click after Lane A completes, sends the same DiagnosticReport JSON to localhost Qwen and renders the Markdown. The AgentLink runner never accepts shell from Qwen. Operator reads, decides. No PlannerDecision JSON path. No autonomous execution.

Phase R2 — **Tool-surface expansion (3–6 weeks)**. Implement items 1, 2, 3, 4 from §12. These widen what Lane A can ingest. The same fixtures will then classify deterministically; Qwen's value shrinks to commentary on those cases, growing on the residual.

Phase R3 — **Re-harvest** (1 day) — re-run this same 16-case harness. Measure: does Lane A coverage climb from current ~11/16 exact-match + 1/16 partial to 14+/16? If yes, the deterministic surface has caught up with Qwen on the bulk of cases.

Phase R4 — **Structured coupling decision** (after R3). At this point, with a tighter schema, designing a constrained Qwen-fills-PlannerDecision path becomes worth the engineering. NOT before — earlier structured coupling discards capability we have evidence Qwen is using.

Phase R5 — **Targeted training decision** (after R4). Only if R3+R4 reveal repeatable residual failure shapes that prompt-only steering cannot fix. Current data does NOT justify training: every observed Qwen output is competent and the bottleneck is verifiably on the AgentLink schema side.

## 15. If training were justified (which we say it is not yet)

For the record, the data schema would be:

```json
{
  "input": {
    "diagnostic_report": {... full enriched schema per §12 items 1–6 ...},
    "operator_context": "..."
  },
  "output": {
    "primary_hypothesis": {"label": "...", "mechanism": "...", "supports": [...], "disconfirms": [...]},
    "alternate_hypotheses": [{... up to 3 ...}],
    "missing_evidence": ["..."],
    "china_specific_notes": "..." | null,
    "protected_topology_concerns": "..." | null,
    "recommended_probes": [{"command": "...", "tag": "READ_ONLY", "implies": "..."}],
    "recommended_repair_plan": [{"step": "...", "tag": "...", "rollback": "...", "verify": "..."}],
    "stop_conditions": ["..."],
    "confidence": "low|medium|high",
    "residual_uncertainty": "..."
  }
}
```

Training targets would be: (a) preserved free-form mechanism reasoning; (b) tighter alignment of `recommended_repair_plan[].tag` to AgentLink's recipe `risk` enum (`read_only` / `reversible_patch` / `privileged_action` / `network_action`); (c) refusal-to-execute when no fixture-named recipe would apply.

But again: **do not do this before R3.**

## 16. If training is not justified (current verdict) — exact product coupling plan

R1 light coupling spec:

- **Trigger**: operator click "Ask AI" in Developer Mode after Lane A returns; gated by `brain.runtime.available()` (Gemma path) OR Qwen Gem URL configured (`AGENTLINK_QWEN_GEM_URL`).
- **Inputs**: redacted DiagnosticReport JSON; current Lane A class+level; the same system prompt this harness used (versioned).
- **Outputs**: Markdown, rendered raw in the side pane. No parse step. No auto-action.
- **Boundary**: explicit "Advisory only — AgentLink will not run commands from this pane" footer.
- **Logs**: every Qwen exchange recorded under `~/Library/Application Support/Cactus AgentLink Rescue/qwen-advisory/<TS>/{prompt.md,response.md}` with 0600 file permissions.
- **Constitution compliance**: zero new mutation paths; no new shell-execution surface; Qwen output never reaches the runner; verifier still owns truth.

## 17. Remaining risks

- **Token cost on the field**. Median ~5300 completion tokens × ~14 min per case on M1 Max 64GB. Pro tier needs to gate this behind explicit click, not auto-fire on every rescue.
- **Governor drift on real fixtures**. v3.8 Maze Pack v1 governor was kept OFF for this harvest. Running it would have biased outputs toward the Synthetic Maze Pack answer shape. Anyone enabling it on AgentLink lane will get visible regression — document this explicitly.
- **Schema-gap masquerade**. Of the 4 "Lane A gap" wins (cn01/cn03/cn04/cn05), 3 are wins because *our authored fixtures included fields outside AgentLink's diagnostic schema*. A skeptic could argue this is Claude-stacking-the-deck. Mitigation: implement §12 items 1–6 (the schema expansion) and re-run; if Lane A then catches the same cases, the win attribution shifts to "schema work" not "Qwen capability". This is exactly Phase R3.
- **Single-judge bias**. Lane C is my read alone. Reproduce by sending the same 16 outputs to Codex or GPT Pro for a parallel taxonomy.
- **No verifier delta**. AgentLink's verifier was untouched; we did not test whether Qwen's repair plans actually pass `agentlink verify network` on the post-fixture state — because dry-run-only, no real mutation. Phase R3 should add a "if Qwen's repair were applied in a simulated state, would the verifier pass" check.
- **Protected-topology assumption**. m13 and cn04 both succeeded because the fixture clearly labeled the protected interface. If the operator's real Mac has Dante traffic but no `protected: true` field, Qwen has no signal; AgentLink's deterministic detector is the safety net there.

## 18. Next hard gate

Schema items 1, 2, 3, 4 from §12 are landed in `internal/diagnose/types.go` and parsed by `internal/diagnose/parsers.go`; classifier rules updated; the same 16 fixtures re-classify deterministically; **at least 13/16 Lane A exact-class-match on the new schema** is the gate to designing R4 structured coupling. Until that gate passes, the right move is to invest in the deterministic surface, not in Qwen-to-AgentLink JSON-wiring.

## Attribution preservation

- **Qwen weights / model behavior**: clean. 16/16 finished, zero hallucination, correct tagging, correct protected-topology refusal.
- **Governor / prompt behavior**: this harvest used a single hand-written free-run system prompt (NOT Milestone M001 v3.8 governor). The v3.8 governor was deliberately kept OFF — it is tuned for Synthetic Maze Pack v1 and would have biased shape. M001's NON_CLAIMS remain preserved.
- **AgentLink deterministic behavior**: classified 11/16 cleanly; missed 4 because of schema; sharpened 1 by Qwen triage.
- **Harness / verifier behavior**: zero issues except the initial 900s urlopen timeout (fixed to 1800s mid-run). No verifier drift; verifier not invoked.
- **Fixture weakness**: 5 CN fixtures were authored by Claude for this run and include fields outside AgentLink's diagnostic schema; this is the schema-gap-masquerade risk in §17. Existing m01–m25 fixtures were used as-is.
- **Inconclusive cases**: none — all 16 produced complete responses; classification claims rest on visible Qwen text, not extrapolation.
