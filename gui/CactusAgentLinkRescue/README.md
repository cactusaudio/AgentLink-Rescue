# Cactus AgentLink Rescue GUI

Native macOS shell for the `agentlink` CLI.

Current status: v0.5.2 GUI field beta. The bundled CLI still reports
`agentlink 0.5.1`.

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

For portable field-beta handoff, build the GUI as a universal binary (`arm64`
+ `x86_64`) and embed the core AgentLink package in:

```text
Cactus AgentLink Rescue.app/Contents/Resources/agentlink/
```

The latest verified portable beta is documented in
`../../docs/offline/FIELD_BETA_STATUS.md`.

Selftest from a packaged app:

```sh
"Cactus AgentLink Rescue.app/Contents/MacOS/CactusAgentLinkRescue" --selftest-gui
```
