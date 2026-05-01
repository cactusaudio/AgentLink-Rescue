Cactus AgentLink Rescue 0.2.1

1. Put this folder anywhere, for example Downloads.
2. Double-click agentlink.command.
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

v0.2 also includes offline recipe docs:
- docs/offline/AGENTLINK_CONSTITUTION.md
- docs/offline/RECIPE_AUTHORING.md
- docs/offline/PLANNER_CONTRACT.md
- docs/offline/REPORT_FORMAT.md
