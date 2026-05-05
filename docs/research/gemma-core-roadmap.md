# Gemma Core Roadmap

## v0.5.0 Lab Verdict

OpenCode stays experimental.

Evidence:

- OpenCode installed: 1.14.33.
- Gemma local server worked through `/v1/models` and `/v1/chat/completions`.
- OpenCode + Gemma ran 9 tasks / 18 attempts.
- Passed: 0.
- Failed: 9.
- Top failure class: OpenCode provider/config failure.
- OpenCode's default run context overflows AgentLink's 8k server profile and remains too slow even after manual 16k llama-server restart.
- Plugin scaffold status: `scaffold_available`; real plugin loading is not verified.
- No sudo/network rescue commands executed through OpenCode.
- No secrets leaked.

Conclusion: OpenCode is not a usable first-class harness for AgentLink Rescue v0.5.0. AgentLink Rescue does not depend on OpenCode and does not route system rescue through OpenCode.

## Direct Gemma Core

Direct Gemma Core should be trained and evaluated separately from OpenCode.

Target local API:

- `/v1/rescue/plan`
- `/v1/incident/summarize`
- `/v1/verifier/interpret`
- `/v1/support/dispatch`

AgentLink should integrate Gemma Core through a `SupervisorClient`, not through OpenCode. The contract remains:

- Gemma proposes structured decisions.
- Policy validator rejects unsafe or unsupported decisions.
- Deterministic runner executes approved recipes/actions.
- Verifier owns truth.
- Snapshot/rollback owns safety.

## Later Revisit

The OpenCode adapter may be revisited later for Local Toolkit coding tasks, report analysis, or developer workflow experiments. It should not become the system rescue executor.
