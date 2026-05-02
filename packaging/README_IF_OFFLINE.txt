Cactus AgentLink Rescue 0.4.0

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

Manual restore structure:
- Backed-up files are under restore-point/files/<original absolute path>.
- Quarantined items are under restore-point/quarantine/.
- rescue.sh rollback uses the printed backup path and quarantine.log when available.

Do not paste admin passwords into commands. Let sudo show its normal prompt.

Core vs Brain package:
- If you only need network rescue, Core is enough.
- Core package has no Qwen model and no llama.cpp runtime. It still runs doctor, recipe repair, network rescue, reports, and rollback.
- Brain package includes the local Qwen GGUF and llama.cpp runtime. Use Brain package on an offline Mac if you need Qwen planner mode.
- If using Core and you want Brain mode, run:
  ./bin/agentlink brain fetch
  or from a source checkout:
  ./scripts/fetch_brain_assets.sh
- If the Mac is already offline, Core cannot download the model. Use the Brain package instead.

Brain package offline checks:
  ./bin/agentlink brain doctor
  ./bin/agentlink brain selftest --json
  ./bin/agentlink brain plan --target path --json
  ./bin/agentlink repair --auto --brain --target path --dry-run

Brain mode is planner-only:
- Qwen outputs PlannerDecision JSON.
- The deterministic runner executes only local recipes.
- Qwen never executes arbitrary shell.

GUI mode is a shell:
- The GUI calls the bundled ./bin/agentlink.
- The GUI does not collect sudo passwords.
- Network rescue safe/standard/deep commands are shown for Terminal copy/paste.
- The GUI does not implement its own repair engine.

v0.3 also includes offline recipe and Brain docs:
- docs/offline/AGENTLINK_CONSTITUTION.md
- docs/offline/RECIPE_AUTHORING.md
- docs/offline/PLANNER_CONTRACT.md
- docs/offline/REPORT_FORMAT.md
- docs/offline/QWEN_BRAIN_PACK.md
- docs/offline/QWEN_PLANNER_PROMPT.md
- docs/offline/BRAIN_RUNTIME_POLICY.md
