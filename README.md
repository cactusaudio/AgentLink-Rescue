# Cactus AgentLink Rescue

Cactus AgentLink Rescue is an offline, portable, reversible macOS rescue tool for restoring the broken path between a Mac and AI agents such as Codex, Claude, GitHub, and API endpoints.

v0.2 is Core only: deterministic CLI engine plus deterministic recipe runtime, no GUI, no local LLM runtime, no privileged helper, no daemon, no telemetry, and no network-based rule updates.

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
- Validates planner JSON without executing model-generated shell.

## What It Does Not Do

- No GUI.
- No local LLM.
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

## Package

```bash
./scripts/package.sh
```

The output folder is:

```text
dist/Cactus-AgentLink-Rescue/
dist/Cactus-AgentLink-Rescue-v0.2.2.zip
```

It can be copied to Downloads and launched with `agentlink.command`.

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
- Some provider failures may be outside local repair scope.
- The fallback `rescue.sh` is intentionally simpler than the Go engine and should be used only when the binary is blocked.
