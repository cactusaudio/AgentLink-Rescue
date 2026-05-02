# GUI Safety Model

Cactus AgentLink Rescue GUI is a native macOS shell around the package-local `agentlink` CLI.

Rules:

1. The GUI never owns repair logic.
2. The GUI never executes model-generated shell.
3. The GUI never captures, stores, or forwards passwords.
4. The GUI never runs `sudo` commands directly.
5. The GUI calls only the package-local `Contents/Resources/agentlink/bin/agentlink`.
6. All command execution uses `Process` with argument arrays, not shell strings.
7. Writable operations must go through the deterministic recipe runner.
8. Writable operations remain snapshot-first and rollback-capable.
9. Verifiers decide repair success; GUI status is display-only.
10. Network rescue safe/standard/deep is shown as copyable Terminal commands.
11. Core GUI remains usable without Brain assets.
12. Brain GUI planning is local and offline when bundled assets are present.

The GUI may show status, command output, reports, dry-runs, copyable commands, and rollback/report paths. It must not bypass `agentlink`.
