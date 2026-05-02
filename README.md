# Cactus AgentLink Rescue

Cactus AgentLink Rescue is an offline, portable, reversible macOS rescue tool for restoring the broken path between a Mac and AI agents such as Codex, Claude, GitHub, and API endpoints.

v0.4 adds a native macOS GUI shell. The GUI does not reimplement rescue logic: it displays status, runs package-local `agentlink` CLI commands through argv arrays, and shows reports, dry-runs, rollback state, and copyable sudo commands. Qwen remains a bounded planner only: facts go to Qwen, Qwen returns PlannerDecision JSON, the validator checks it, and the deterministic recipe runner executes only registered rollback-capable recipes. There is still no telemetry, daemon, privileged helper, SMAppService, RAG, queue, remote mutation, or arbitrary shell execution.

## Product Thesis

Remote AI agents cannot help when the local machine is offline, trapped behind stale proxy settings, missing routes, broken DNS, or blocked agent endpoints. AgentLink Rescue is the local fallback layer: it diagnoses the Mac, classifies the failure, applies bounded repairs, and keeps every mutating action reversible.

This is not a generic network reset tool and not a cleanup app.

## What It Does

- Runs non-mutating diagnostics without root.
- Classifies local agent-link failures with deterministic rules.
- Clears system proxy, DNS, DHCP, and IPv6 settings in safe rescue.
- Creates a clean macOS network location in standard rescue.
- Optionally quarantines known Clash-family helper residue after confirmation.
- Backs up and quarantines selected SystemConfiguration plists in deep rescue.
- Creates restore points before mutating actions.
- Logs every command with redacted stdout/stderr summaries.
- Rolls back restore points.
- Packages as a copyable folder with `agentlink.command` and `rescue.sh`.
- Runs bounded JSON recipes with snapshot-first mutation and verifier-driven repair.
- Runs optional local Qwen planning through `llama-cli`.
- Validates Brain planner JSON before dry-run or execution.
- Provides a native macOS GUI rescue console as a thin wrapper around the CLI.

## What It Does Not Do

- No GUI-owned repair logic.
- No model-generated shell execution.
- No Qwen executor surface.
- No permanent privileged helper.
- No SMAppService.
- No automatic notarization.
- No MDM/profile modification.
- No automatic removal of detect-only VPN or security tools.
- No persistent daemon.
- No telemetry.
- No cloud sync.
- No network-based rule updates.
- No MTU changes.
- No RAG or queue.
- No remote SSH/tmux mutation.
- No automatic paid API smoke tests.

## Safety Model

- Diagnose is non-mutating and does not require root.
- Mutating rescue runs create restore points under:
  `/Library/Application Support/Cactus AgentLink Rescue/restore-points/`
- Third-party residue is quarantined, not deleted.
- System paths `/System`, `/bin`, `/usr/bin`, and `/usr/sbin` are never modified.
- Admin passwords are never accepted as arguments, stdin, or logs.
- `sudo` uses the normal macOS password prompt.
- Credentials in URLs, proxy env vars, git/npm/brew configs, and command logs are redacted.
- Private keys, Wi-Fi passwords, browser cookies, and shell history are not collected.

## CLI Usage

```bash
agentlink diagnose [--json] [--verbose]
agentlink doctor [--json]
agentlink recipe list [--json]
agentlink recipe inspect <id> [--json]
agentlink recipe run <id> [--dry-run] [--yes] [--json] [--param key=value]
agentlink repair --target path|proxy|codex|keys [--dry-run] [--yes] [--json]
agentlink brain doctor [--json]
agentlink brain fetch [--model qwen3-4b-instruct-2507-q4km] [--runtime llama.cpp]
agentlink brain selftest [--json]
agentlink brain prompt --text "..." [--json]
agentlink brain plan --target path|proxy|codex|keys|network [--json]
agentlink repair --auto --brain --target path|proxy|codex|keys|network [--dry-run] [--yes] [--online] [--json]
agentlink planner validate <decision.json>
agentlink report --for-human --latest
agentlink report --for-codex --latest
agentlink classify [--json]
agentlink rescue [--level safe|standard|deep] [--yes] [--dry-run] [--json]
agentlink rollback [--last | --id RESTORE_POINT_ID] [--dry-run] [--json]
agentlink report [--latest | --id REPORT_ID] [--json]
agentlink selftest
agentlink version
```

Examples:

```bash
./bin/agentlink diagnose --json
./bin/agentlink rescue --level safe --dry-run
sudo ./bin/agentlink rescue --level standard
sudo ./bin/agentlink rescue --level deep --yes
sudo ./bin/agentlink rollback --last
```

## Repair Levels

`safe` clears proxy state, DNS servers, search domains, DHCP, IPv6 automatic mode, renews DHCP on hardware devices, and flushes DNS cache.

`standard` includes safe rescue, creates and switches to a clean network location, detects new hardware, flushes routes, renews DHCP again, and restarts Wi-Fi when a Wi-Fi service exists.

`deep` requires `--yes`. It includes safe and standard repair, quarantines eligible known residue, backs up selected network configuration plists, removes them by moving them into the restore point quarantine, and recommends reboot.

## Build

Runtime does not require Homebrew, Python, npm, pip, or network access. Building requires Go on the build machine.

```bash
./scripts/build.sh
```

Cross-build examples:

```bash
GOOS=darwin GOARCH=arm64 ./scripts/build.sh
GOOS=darwin GOARCH=amd64 ./scripts/build.sh
```

If `lipo` is available, `scripts/build.sh` creates a universal `bin/agentlink` when no explicit `GOARCH` is set.

## Test

```bash
go test ./...
./bin/agentlink selftest
./bin/agentlink rescue --level safe --dry-run
./bin/agentlink rescue --level standard --dry-run
```

Brain dogfood, when assets are present:

```bash
./scripts/dogfood_brain.sh
./scripts/dogfood_runtime_assets.sh
./scripts/dogfood_runtime_core.sh
./scripts/dogfood_runtime_brain.sh
```

## Package

```bash
./scripts/package.sh
```

The output folder is:

```text
dist/Cactus-AgentLink-Rescue/
dist/Cactus-AgentLink-Rescue-v0.4.0-core.zip
```

It can be copied to Downloads and launched with `agentlink.command`.

To build the optional Brain package after fetching the model/runtime:

```bash
./scripts/fetch_brain_assets.sh
./scripts/package_brain.sh
```

Brain package output:

```text
dist/Cactus-AgentLink-Rescue-v0.4.0-brain-qwen3-4b-q4km.zip
```

Native GUI packages:

```bash
./scripts/package_gui_core.sh
./scripts/package_gui_brain.sh
```

GUI package outputs:

```text
dist/Cactus-AgentLink-Rescue-v0.4.0-core-gui.zip
dist/Cactus-AgentLink-Rescue-v0.4.0-brain-gui-qwen3-4b-q4km.zip
```

The GUI app embeds the CLI package under:

```text
Cactus AgentLink Rescue.app/Contents/Resources/agentlink/
```

The GUI uses the embedded `bin/agentlink`; it does not use a system `agentlink` from `PATH` unless a developer override is set. It uses `Process` with argument arrays, not shell command strings.

### Core Package Vs Brain Package

The Core package is the network and recipe runtime. It does not include the Qwen GGUF model or llama.cpp runtime. Core can still run `doctor`, `diagnose`, recipe repairs, network rescue, reports, rollback, and `brain doctor`. In Core, `brain doctor` reports missing Brain assets and prints fetch commands.

The Brain package includes everything in Core plus:

- `assets/models/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf`
- `assets/runtimes/llama.cpp/<arch>/llama-cli`
- required llama.cpp shared libraries
- manifests and licenses

On an offline Mac, use the Brain package if you need local Qwen planning. After unzip, these commands should run without network:

```bash
./bin/agentlink brain doctor
./bin/agentlink brain selftest --json
./bin/agentlink brain plan --target path --json
./bin/agentlink repair --auto --brain --target path --dry-run
```

Core can fetch Brain assets only when the Mac has network access:

```bash
./bin/agentlink brain fetch
```

Qwen is not the executor. It returns PlannerDecision JSON only. The validator rejects unknown recipes, low-confidence repairs, privileged/destructive actions, and invalid JSON. The deterministic runner executes only bundled recipes, and writable repairs still require snapshot and rollback.

GUI dogfood:

```bash
./scripts/dogfood_gui_core.sh
./scripts/dogfood_gui_brain.sh
```

## Reports And Restore Points

User-level reports:

```text
~/Library/Application Support/Cactus AgentLink Rescue/reports/
```

System restore points:

```text
/Library/Application Support/Cactus AgentLink Rescue/restore-points/
```

Each restore point contains:

- `manifest.json`
- `preflight.json`
- `postflight.json`
- `commands.log`
- `files/`
- `quarantine/`
- `human-report.txt`

## Privacy And Logging

AgentLink Rescue logs local diagnostic state needed to classify and repair agent-link failures. It does not collect private keys, Wi-Fi passwords, browser cookies, shell history, or telemetry. Proxy credentials and tokens are redacted before report or command-log output.

## Known Limitations

- macOS only.
- Rules and recipes are static and bundled locally.
- Detect-only VPN, firewall, proxy, and security tools are reported but not automatically removed.
- MDM/profile state is detected but never modified.
- Deep rescue may require reboot before macOS fully rebuilds network configuration.
- Codex DeepSeek provider recipes generate a static config template with current `name`, `base_url`, `env_key`, and model fields; an explicit online smoke test is required before treating the provider as operational.
- DeepSeek Chat Completions compatibility and Codex Responses wire protocol may not be equivalent; do not claim runtime compatibility without a smoke test.
- Brain mode requires the Qwen GGUF and llama.cpp runtime. Core package remains functional without them.
- Qwen planning can recommend only registered recipes; it cannot invoke network rescue safe/standard/deep automatically.
- Some provider failures may be outside local repair scope.
- The fallback `rescue.sh` is intentionally simpler than the Go engine and should be used only when the binary is blocked.
