# AgentLink — Vision & Operator Judgment

> **The human layer. Read this before the mechanics.** `AGENTS.md` and
> `docs/offline/AGENTLINK_CONSTITUTION.md` tell you *how* the system works;
> `HANDOFF.md` tells you *where it is now*. This file tells you **why it exists
> and what must not change.** Operator / owner: **Bowei** — he owns vision and the
> verdict. Agents (Claude, Codex, GPT-Pro) own mechanism and adversarial audit;
> they do not own direction.

## Why this exists (the human problem)

Remote AI agents — Codex, Claude, the API — are useless the moment the local
machine can't reach them. A stale Clash/mihomo TUN, a dead localhost proxy, a
hijacked default route, polluted DNS, an audio VLAN that captured the route: any
one of these takes the whole agent-driven workflow offline — and the agent that
could fix it is exactly the thing you can no longer reach.

AgentLink is the floor under that workflow: the **local fallback layer** that runs
when nothing else can — diagnose the Mac, classify the failure, apply a bounded
reversible repair, get back online. It is built by someone who lives this failure
mode (several Macs, constant proxy churn, real Dante / audio VLANs), not as a
theoretical "network tool."

It is **not** a generic network-reset utility and **not** a cleanup app. It is a
rescue floor for a human — developer or not — whose machine is down and who needs
to be back online with one action and **zero new damage**.

## Where it sits

AgentLink is the connectivity-rescue ("恢复通路") module of **Cactus Local Agent** —
the offline, task-specific small-model agent line. Two siblings: `AgentLink-Rescue`
(macOS) and `AgentLink-for-Win` (Windows). The line's bet: a small local model,
tightly harnessed to a deterministic core, doing **one hard job reliably** — not a
general chatbot.

## North-star (愿景)

A stressed human gets back online **fast, offline, and reversibly**:

- **One button.** A non-developer can rescue their own machine; an expert can drop to the CLI.
- **Fully offline.** No network, no Homebrew / Python / npm at runtime — it has to work *on the broken machine itself*.
- **Portable.** Carry it on a USB stick to a dead Mac and rescue that one too.
- **Never worse.** Every mutation is snapshotted and reversible. If it can't help, it fails clean and reports.

## Settled judgment calls — preserve, do not re-litigate

These are the operator's decisions. They are *why* the Constitution reads the way
it does. An incoming agent may **extend** them; it may not quietly **reverse** them.

1. **Connectivity-first.** Getting online is the first imperative. The `nuclear`
   level **deliberately overrides** the protected-topology (audio VLAN) refusal — a
   conscious reversal of the earlier "protected-refusal-is-sacred" stance: a human
   who is offline outranks a preserved VLAN. The graduated tiers still respect
   topology; `nuclear` is the explicit, snapshotted, operator-sanctioned escape hatch.
2. **The deterministic core owns truth; the model is shadow-only.** Gemma / Qwen
   explain and advise. They never execute, never gate a release, never introduce a
   fact. Rules, recipes, the verifier, rollback, and the restart gate own behavior.
   *Verifier owns truth* — truth lives in verifiers and frozen acceptance criteria,
   never in a confident model or a green report.
3. **Reversibility is sacred.** No unmanaged mutation. Every mutating repair carries
   a verifier and a rollback, or it is blocked. Failure must be clean.
4. **Offline & portable are hard constraints, not features.** If a change needs the
   network or a package manager at runtime, it is wrong by construction.
5. **用约束打败约束 — beat constraints with constraints.** When something is unsafe or
   drifting, the fix is a *tighter counter-constraint*, not a wider blast radius and
   not a "temporary" convenience exception. Do not weaken a gate to make a case pass.
6. **Recipes / cards are the product.** Bounded, auditable, card-backed repairs — not
   prose, not a chatbot, not invented fixes. Every diagnosis cites a card; every
   repair passes the gate.

## Anti-downgrade (the most common way an agent drifts)

The recurring failure mode is **shrinking the ambition to fit the easy
implementation** — turning a platform into a single runtime, a hard rescue into a
happy-path demo, a judgment call into a TODO. The operator's nouns —
*connectivity-first, 主权/sovereignty, 提纯/distillation, deterministic-core* — are
**entry points into the real problem, not the problem itself**. If a task feels like
it is getting smaller as you implement it, stop: you are probably downsizing the
vision. Surface it; do not silently narrow it.

## Roles & who owns truth

- **Bowei** owns vision, direction, and the verdict. He does not write code — he
  steers, judges, and corrects. A follow-up message from him is a *route correction*,
  not proof you were wrong to proceed. Default to acting on reversible work; stop and
  ask only at real, irreversible boundaries.
- **Agents** (Claude / Codex / GPT-Pro) own mechanism, implementation, and
  *adversarial audit of each other*. No agent owns direction, and no agent declares
  its own success.
- **Truth** is owned by verifiers, frozen tasks, acceptance criteria, evidence, and
  user intent — never by the harness, a passing CI line, or a confident explanation.
  A fixed runner or a prettier report is not proof of capability. **产出越漂亮越可疑.**

## For the incoming agent — start here

Read order: **this file → `HANDOFF.md` →
`governor/reports/AGENTLINK_KERNEL_OPUS_REFACTOR_SCORECARD.md` (current state) →
`AGENTS.md` + `docs/offline/AGENTLINK_CONSTITUTION.md` (the contract).**

Without an explicit operator decision you may **not**: reverse a settled judgment
call above, weaken a safety gate, downsize the vision, or present a model's output
as truth. When in doubt, preserve reversibility and surface the question.
