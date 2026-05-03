Cactus AgentLink Rescue 0.4.5

1. Put this folder anywhere, for example Downloads.
2. Double-click agentlink.command.
   If you are using the GUI package, double-click Cactus AgentLink Rescue.app instead.
3. If macOS blocks the launcher or binary because of Gatekeeper/quarantine, open Terminal and run:
   xattr -cr "/path/to/Cactus-AgentLink-Rescue"
   /bin/bash "/path/to/rescue.sh" diagnose
   sudo /bin/bash "/path/to/rescue.sh" safe
4. For deep reset:
   sudo /bin/bash "/path/to/rescue.sh" deep
5. To rollback, use agentlink if the binary works:
   ./bin/agentlink restore last
   sudo ./bin/agentlink rollback --last
   sudo ./bin/agentlink rollback --id <restore-point-id>
   If the binary does not work:
   sudo /bin/bash "/path/to/rescue.sh" rollback "/Library/Application Support/Cactus AgentLink Rescue/restore-points/<restore-point-id>"

Guided Rescue:
- Analyze only:
  ./bin/agentlink guided rescue --target auto --dry-run
- Reversible user-level repair only:
  ./bin/agentlink guided rescue --target auto --yes
- Guided Rescue does not run sudo or network standard/deep rescue automatically.

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
- Use Cactus-AgentLink-Rescue-v0.4.5-macbook-field-gui-proxykit.zip when copying to a MacBook whose internet breaks after Clash Verge TUN mode.
- Unzip it, then double-click RUN-FIRST.command.
- The app opens in MacBook Network Rescue mode and shows Analyze Network plus copyable Safe/Standard/Deep Terminal commands.
- The GUI does not run sudo and does not enable Clash proxy/TUN automatically.
- If the app is blocked, run:
  xattr -cr "Cactus MacBook Network Rescue"

Brain package offline checks:
  ./bin/agentlink brain doctor
  ./bin/agentlink brain selftest --json
  ./bin/agentlink brain plan --target path --json
  ./bin/agentlink repair --auto --brain --target path --dry-run

Brain mode is planner-only:
- Gemma outputs PlannerDecision JSON.
- The deterministic runner executes only local recipes.
- Gemma never executes arbitrary shell.

Installer Center:
- Installer Center uses official sources only.
- Codex CLI: @openai/codex or Homebrew codex.
- Claude Code CLI: @anthropic-ai/claude-code.
- Gemini CLI: @google/gemini-cli or Homebrew gemini-cli.
- Codex App opens the official OpenAI page.
- Clash Verge Rev can be opened from a cached DMG when using a ProxyKit package.
- AgentLink does not enable proxy/TUN automatically and does not collect passwords.

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
- Use Guided Rescue or Expert Console for actual repair flows.

GUI mode is a shell:
- The GUI calls the bundled ./bin/agentlink.
- The GUI does not collect sudo passwords.
- Network rescue safe/standard/deep commands are shown for Terminal copy/paste.
- The GUI does not implement its own repair engine.

v0.4.5 also includes offline recipe and Brain docs:
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
