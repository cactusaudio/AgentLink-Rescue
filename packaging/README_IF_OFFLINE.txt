Cactus AgentLink Rescue 0.5.x

Current line: v0.5.2 GUI field beta with shared CLI core version
`agentlink 0.5.1`.

1. Put this folder anywhere, for example Downloads.
2. Double-click agentlink.command.
   If you are using the GUI package, double-click Cactus AgentLink Rescue.app instead.
3. If macOS blocks the launcher or binary because of Gatekeeper/quarantine, open Terminal and run:
   xattr -cr "/path/to/Cactus-AgentLink-Rescue"
   ./bin/agentlink package doctor --json
   ./bin/agentlink package repair --yes --json
   /bin/bash "/path/to/rescue.sh" diagnose
   sudo /bin/bash "/path/to/rescue.sh" safe
4. For last-resort clean baseline or deep reset:
   sudo ./bin/agentlink rescue --level clean-baseline --yes
   sudo /bin/bash "/path/to/rescue.sh" deep
5. To rollback, use agentlink if the binary works:
   ./bin/agentlink restore last
   sudo ./bin/agentlink rollback --last
   sudo ./bin/agentlink rollback --id <restore-point-id>
   If the binary does not work:
   sudo /bin/bash "/path/to/rescue.sh" rollback "/Library/Application Support/Cactus AgentLink Rescue/restore-points/<restore-point-id>"

Rescue Orchestrator:
- Analyze only:
  ./bin/agentlink orchestrator rescue --target auto --dry-run --json
- Clash/TUN repair ticket:
  ./bin/agentlink ticket create --type clash-tun-fix --json
- Targeted Clash/TUN Terminal repair:
  sudo ./bin/agentlink rescue --level tun --yes
- Verify after repair or restart:
  ./bin/agentlink verify network --json
- The GUI does not run sudo. Privileged repair uses Terminal tickets.
- Last resort, only after targeted repair/restart/support review:
  sudo ./bin/agentlink rescue --level clean-baseline --yes

Manual restore structure:
- Backed-up files are under restore-point/files/<original absolute path>.
- Quarantined items are under restore-point/quarantine/.
- rescue.sh rollback uses the printed backup path and quarantine.log when available.

Do not paste admin passwords into commands. Let sudo show its normal prompt.

Core vs Brain package:
- If you only need network rescue, Core is enough.
- Core package has no Gemma model and no llama.cpp runtime. It still runs doctor, recipe repair, network rescue, reports, and rollback.
- Brain package includes the local Gemma GGUF and llama.cpp runtime. Use Brain package on an offline Mac if you need Gemma planner mode.
- If using Core and you want Brain mode, run:
  ./bin/agentlink brain fetch
  or from a source checkout:
  ./scripts/fetch_brain_assets.sh
- If the Mac is already offline, Core cannot download the model. Use the Brain package instead.
- Source checkouts keep large assets out of git. Rebuilds can use:
  ~/CactusLocalAgent/.asset-cache/AgentLink-Rescue
  or:
  AGENTLINK_ASSET_CACHE=/path/to/cache ./scripts/package_brain.sh

MacBook Field Rescue package:
- Use Cactus-AgentLink-Rescue-v0.5.1-macbook-field-gui-proxykit.zip when copying to a MacBook whose internet breaks after Clash Verge TUN mode.
- Unzip it, then double-click RUN-FIRST.command.
- The app opens in MacBook Network Rescue mode with one primary Fix My Connection flow. Clash/TUN is checked as one possible cause; targeted repair uses Terminal tickets when applicable.
- The GUI does not run sudo and does not enable Clash proxy/TUN automatically.
- If the app is blocked, run:
  xattr -cr "Cactus MacBook Network Rescue"

Brain package offline checks:
  ./bin/agentlink brain doctor
  ./bin/agentlink brain selftest --json
  ./bin/agentlink brain plan --target path --json
  ./bin/agentlink repair --auto --brain --target path --dry-run

Brain mode is planner-only:
- Gemma may explain or shadow-plan rescue decisions, but current v0.5.x field-beta behavior is deterministic.
- The deterministic runner executes only local recipes.
- Gemma never executes arbitrary shell.
- For Network/TUN rescue, deterministic failure classes and validated recipes own the repair path; Gemma output is not a gated supervisor.

Installer Center:
- Installer Center uses official sources only.
- Codex CLI: @openai/codex or Homebrew codex.
- Claude Code CLI: @anthropic-ai/claude-code.
- Gemini CLI: @google/gemini-cli or Homebrew gemini-cli.
- Codex App opens the official OpenAI page.
- Clash Verge Rev can be opened from a cached DMG when using a ProxyKit package.
- AgentLink does not enable proxy/TUN automatically and does not collect passwords.
- OpenCode Bridge is experimental. Lab work showed OpenCode + Gemma is not yet a usable first-class harness. It is optional config/template support only and is not the system rescue executor.

Offline Readiness / Last-Good / Support Bundle:
  ./bin/agentlink readiness doctor
  ./bin/agentlink dev doctor
  ./bin/agentlink last-good save
  ./bin/agentlink support bundle
- These are limited to AgentLink rescue readiness, known-good AI-tool config, and redacted handoff evidence.
- They do not index repos, manage project dependency caches, run tests, or generate patches.
- Last-Good restore only writes known AI config paths and preserves original permissions when known, defaulting to 0600.
- Support Bundle is redacted but may include local paths, tool presence, network/proxy status, readiness reports, and latest session metadata.

Brain Chat Sandbox:
- Hidden/advanced local Gemma chat for explanations only.
- It cannot run commands, change settings, or repair the machine.
- Use the main Fix flow or Developer Mode Expert Console for actual repair flows.

GUI mode is a shell:
- The GUI calls the bundled ./bin/agentlink.
- The GUI does not collect sudo passwords.
- Normal mode shows one primary Fix button, health, last result, support bundle, and Settings.
- Developer Mode exposes Doctor, Brain, Installer Center, Readiness, Rescue levels, Rollback, OpenCode status, and raw logs.
- Network rescue safe/tun/standard/clean-baseline/standard-system-reset/deep commands are shown for Terminal copy/paste inside Developer Mode.
- The GUI does not implement its own repair engine.

v0.5.x also includes offline recipe and Brain docs:
- docs/offline/FIELD_BETA_STATUS.md
- docs/offline/AGENTLINK_CONSTITUTION.md
- docs/offline/RECIPE_AUTHORING.md
- docs/offline/PLANNER_CONTRACT.md
- docs/offline/REPORT_FORMAT.md
- docs/offline/GEMMA_BRAIN_PACK.md
- docs/offline/GEMMA_PLANNER_PROMPT.md
- docs/offline/BRAIN_RUNTIME_POLICY.md
- docs/offline/INSTALLER_CENTER.md
- docs/offline/PROXY_RECOVERY_KIT.md
- docs/offline/BRAIN_CHAT_SANDBOX.md
- docs/offline/OFFLINE_READINESS_CENTER.md
- docs/offline/LAST_GOOD_PROFILES.md
- docs/offline/SUPPORT_BUNDLE.md
- docs/offline/MACBOOK_FIELD_RESCUE.md
- docs/offline/CLASH_TUN_RESCUE.md
- docs/offline/TERMINAL_REPAIR_TICKETS.md
- docs/offline/RESTART_GATE.md
- docs/offline/RESCUE_ORCHESTRATOR.md
- docs/offline/OPENCODE_BRIDGE.md
- docs/offline/CORE_RELIABILITY_MODEL.md
- docs/offline/ROBUSTNESS_MATRIX.md
- docs/offline/MULTI_MAC_DOGFOOD_PLAN.md
