# OpenCode Gemma Lab

## Purpose

This lab tested whether Gemma 4 E4B IT Q4_K_M served by local llama.cpp can be a useful OpenCode harness for AgentLink-adjacent work: report analysis, safe repair planning, sandbox coding, and future Local Toolkit probes.

The answer from this run is negative for first-class product use: the raw local Gemma endpoint works, and OpenCode can resolve the local provider config, but real `opencode run` attempts are not usable yet. OpenCode's default agent prompt exceeds AgentLink's current 8k server context and remains too slow to complete useful tasks within bounded local-agent timeouts even after a manual 16k llama-server restart.

## Setup

- Branch: `lab/gemma-opencode-flywheel`
- Baseline HEAD: `85294b4de2b5a32f37ac672909da9c1f5481b071`
- Lab root: `/Users/bowei/CactusLocalAgent/.labs/gemma-opencode`
- OpenCode: `1.14.33`
- Model: `gemma-4-e4b-it-q4km`
- Model file: `/Users/bowei/CactusLocalAgent/AgentLink-Rescue/dist/Cactus-AgentLink-Rescue/assets/models/gemma-4-E4B-it-Q4_K_M.gguf`
- Model SHA-256 observed: `dff0ffba4c90b4082d70214d53ce9504a28d4d8d998276dcb3b8881a656c742a`
- Runtime: `/Users/bowei/CactusLocalAgent/AgentLink-Rescue/dist/Cactus-AgentLink-Rescue/assets/runtimes/llama.cpp/arm64/llama-server`
- Provider config: `/Users/bowei/CactusLocalAgent/.labs/gemma-opencode/workspaces/opencode-agentlink-sandbox/opencode.json`

The source checkout `./bin/agentlink brain doctor --json` reported missing user-cache Brain assets. The packaged Brain build under `dist/Cactus-AgentLink-Rescue` did verify package-local model/runtime availability.

## Architecture

OpenCode should stay outside the AgentLink Rescue execution kernel. Its plausible role is local coding, report analysis, dry-run orchestration, and future Local Toolkit maintenance. The deterministic rescue kernel, recipes, verifier, rollback, restart gate, and Terminal Repair Tickets remain the only system repair execution surfaces.

The lab used disposable workspaces under `/Users/bowei/CactusLocalAgent/.labs/gemma-opencode/workspaces`. The live repo was not used as a mutation target except for this research report.

## Safety Boundaries

Mandatory boundaries were preserved:

- No sudo commands executed.
- No `networksetup`, `route`, `ifconfig`, `launchctl`, `killall`, or other network rescue mutations executed through OpenCode.
- No real API keys or paid online model calls used.
- No merge, tag, or commit was made.
- OpenCode attempts were run only in disposable workspaces.

OpenCode's default `build` agent exposes broad tools including `bash`, `edit`, and `write` unless constrained. That is not acceptable as the AgentLink rescue executor boundary. Any product bridge must install a restricted agent/plugin policy before letting Gemma operate near AgentLink reports or local tools.

## OpenCode Provider Status

Status: `configured_but_not_operational`

Evidence:

- `curl http://127.0.0.1:8080/v1/models` succeeded against local llama-server.
- Direct `/v1/chat/completions` with model alias `gemma-4-e4b-it` returned `OK`.
- `opencode debug config` resolved the sandbox `agentlink-gemma` provider.
- `opencode models agentlink-gemma --verbose` listed both configured local model ids.

Failure:

- AgentLink's packaged `brain server start --port 8080` hardcodes `-c 8192`.
- A trivial `opencode run` exceeded that context with about 10k prompt tokens and entered repeated compaction/error behavior.
- A manual 16k llama-server restart avoided immediate context overflow but did not produce useful task output within 20-second bounded attempts; a prior unbounded smoke ran for over two minutes with no emitted output and had to be terminated.

## Plugin Hook Status

Status: `scaffold_available`; static policy check `policy_verified`; real OpenCode plugin load not verified.

The repo scaffold at `opencode/agentlink-plugin/` exposes read-only/dry-run tools in source:

- `agentlink_doctor`
- `agentlink_readiness`
- `agentlink_support_bundle`
- `agentlink_guided_rescue_dryrun`
- `agentlink_recipe_list`
- `agentlink_verify_network`
- `agentlink_tun_diagnose`

The static policy check confirmed blocked tokens for `sudo`, `networksetup`, `route`, `ifconfig`, `launchctl`, `killall`, and `pkill`, with no forbidden tool names exposed. OpenCode plugin installation was not attempted because `opencode plugin <module>` expects an npm module and mutates project/global config.

## Task Suite

Task definitions were written under `/Users/bowei/CactusLocalAgent/.labs/gemma-opencode/tasks`.

Groups covered:

- A: AgentLink report and diagnosis, including the canonical Clash/TUN residue scenario and standard-rescue-worsened scenario.
- B: Local coding tasks in toy Go and Node workspaces.
- C: Safety boundary refusal tasks for `sudo networksetup`, `route`, and `ifconfig`.
- D: OpenCode provider smoke.
- E: Local Toolkit exploratory repo-doctor concept.

## Results

Summary file: `/Users/bowei/CactusLocalAgent/.labs/gemma-opencode/reports/summary.json`

| Metric | Result |
| --- | ---: |
| Tasks | 9 |
| Attempts | 18 |
| Passed | 0 |
| Failed | 9 |
| Inconclusive | 0 |
| Dominant failure class | OpenCode provider/config failure |

Every corrected task attempt failed before meaningful model output. This means the run did not prove Gemma cannot classify reports or fix code; it proved the current OpenCode harness path is not yet a viable way to ask it.

## Examples

Raw endpoint success:

- `/v1/models` returned the packaged Gemma GGUF model.
- `/v1/chat/completions` returned `OK` for a tiny prompt.

OpenCode failure:

- With AgentLink's default 8k server wrapper, a trivial OpenCode run produced `ContextOverflowError` around 10k prompt tokens.
- With manual 16k context, bounded attempts timed out with empty output.

Harness correction:

- Initial non-sandbox workspace attempts failed with `ProviderModelNotFound` because project-local config existed only in the sandbox workspace.
- The harness was corrected to copy `opencode.json` into each disposable workspace before the final run.

## Failures

Primary failure: OpenCode 1.14.33's default run context is too heavy for the current AgentLink Gemma server profile and too slow for this local model in bounded agent-loop use.

Secondary failures:

- AgentLink `brain server start` needs configurable context for OpenCode bridge use.
- OpenCode plugin scaffold is not packaged as a loadable npm module.
- Default OpenCode agent permissions are too broad for AgentLink safety boundaries unless replaced or constrained.
- The Go verifier could not run because `go` was not available in this shell.

## Promoted Rules

No model behavior rules were promoted. The task outputs never reached a point where a report-diagnosis or safety response rule could be validated through OpenCode.

## Rejected Rules

Rejected as product assumption:

- "AgentLink's current 8k Brain server wrapper is sufficient for OpenCode." It is not sufficient for OpenCode 1.14.33 default prompt behavior.

## Recommendation

Keep OpenCode experimental for v0.5.0/v0.5.1. Do not make it a first-class Cactus Local Agent component yet.

Recommended product changes:

- Harden OpenCode Bridge as a developer-mode experiment, not a rescue-path dependency.
- Add an OpenCode-specific llama-server profile with configurable context and explicit performance expectations.
- Package the AgentLink OpenCode plugin as a real loadable module or document the exact local plugin installation path.
- Define a restricted OpenCode agent that denies dangerous shell/network mutation and exposes only safe AgentLink read-only/dry-run tools.
- Re-run this suite only after D001 provider smoke completes under a practical timeout.

Do not build Local Toolkit Alpha on this OpenCode path until provider smoke and policy loading both pass.
