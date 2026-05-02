# Cactus AgentLink Rescue GUI

Native macOS shell for the `agentlink` CLI.

The GUI does not implement repair logic. It locates the package-local CLI at:

```text
Cactus AgentLink Rescue.app/Contents/Resources/agentlink/bin/agentlink
```

All operations use `Process` with argv arrays. There is no shell string executor, no password capture, no privileged helper, no daemon, and no telemetry.

Build:

```sh
../../scripts/build_gui.sh
```

Package:

```sh
../../scripts/package_gui_core.sh
../../scripts/package_gui_brain.sh
```

Selftest from a packaged app:

```sh
"Cactus AgentLink Rescue.app/Contents/MacOS/CactusAgentLinkRescue" --selftest-gui
```
