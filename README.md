# Cactus AgentLink Rescue

Cactus AgentLink Rescue is an offline, portable, reversible macOS rescue tool for restoring the broken path between a Mac and AI agents such as Codex, Claude, GitHub, and API endpoints.

Current line: v0.5.2 GUI field beta, hardened at commit `01ced16` (`Harden AgentLink field beta`). The shared CLI core still reports `agentlink 0.5.1`; v0.5.2 is the GUI/audit/portable-beta product line. The current verified portable bundle is a universal macOS app (`arm64` + `x86_64`) with dry-run-first GUI rescue, support-bundle export, tighter private file permissions, symlink-safe support-bundle zipping, config-mode preservation, and fuzz coverage for redaction, planner decisions, recipe decoding, and diagnosis-graph JSON input.

This is still a controlled field beta, not a notarized public release. Gemma 4 E4B-it Q4_K_M remains explanation/shadow-only for rescue decisions; deterministic rules, policy validation, recipes, verifier, rollback, and restart gate own release behavior. There is still no telemetry, daemon, privileged helper, SMAppService, RAG, queue, remote mutation, sudo execution in the GUI, or arbitrary shell execution.

## Product Thesis

Remote AI agents cannot help when the local machine is offline, trapped behind stale proxy settings, missing routes, broken DNS, or blocked agent endpoints. AgentLink Rescue is the local fallback layer: it diagnoses the Mac, classifies the failure, applies bounded repairs, and keeps every mutating action reversible.

This is not a generic network reset tool and not a cleanup app.

## Current Verified Artifact

Latest local portable beta:

```text
/Volumes/CTS Dark/Cactus-AgentLink-Rescue-v0.5.2-field-beta-hardened-portable-universal-20260524T012027Z/
/Volumes/CTS Dark/Cactus-AgentLink-Rescue-v0.5.2-field-beta-hardened-portable-universal-20260524T012027Z.zip
SHA256 e9347c39c50c42920240cadca55f3a28f49079561822604919f8dda81caa17eb
```

Verified properties:

- top-level double-clickable `Cactus AgentLink Rescue.app`;
- fallback `Open AgentLink Rescue.command`;
- `VERIFY-CHECKSUMS.command`, `CHECKSUMS.txt`, `PACKAGE-MANIFEST.json`, and `README-FIRST.txt`;
- app binary and embedded `bin/agentlink` are universal Mach-O (`arm64` + `x86_64`);
- ad-hoc codesign verification passes;
- no model weights, GGUF, llama runtime, DMG installer, `node_modules`, build metadata, or Finder metadata in the portable package;
- `--selftest-gui` reports `ready_for_field_beta`;
- embedded CLI `guided rescue --target auto --dry-run --json` runs dry-run only;
- actual `open` launch from the external USB path was verified and cleaned up.

## What It Does

- Runs non-mutating diagnostics without root.
- Classifies local agent-link failures with deterministic rules.
- Clears system proxy, DNS, DHCP, and IPv6 settings in safe rescue.
- Runs targeted Clash/Mihomo TUN repair before broad network reset when TUN signatures are detected.
- Keeps standard/deep rescue as fallback layers, not the first path for Clash/TUN residue.
- Optionally quarantines known Clash-family helper residue after confirmation.
- Backs up and quarantines selected SystemConfiguration plists in deep rescue.
- Creates restore points before mutating actions.
- Logs every command with redacted stdout/stderr summaries.
- Rolls back restore points.
- Packages as a copyable folder with `agentlink.command` and `rescue.sh`.
- Runs bounded JSON recipes with snapshot-first mutation and verifier-driven repair.
- Runs optional local Gemma 4 E4B planning through `llama-cli`.
- Validates Brain planner JSON before dry-run or execution.
- Preserves protected local topologies such as Dante/AES67, NDI/video, ATEM/PTZ, Art-Net/sACN, NAS/iSCSI, lab/PLC, VM bridge, USB management, and MDM/policy-owned networks by turning them into deterministic repair-corridor boundaries.
- Runs Guided Rescue as a bounded CLI workflow for non-developer users.
- Runs the v0.5 Rescue Orchestrator for network/TUN failures with Terminal repair tickets.
- Provides a native macOS GUI rescue operator as a thin wrapper around the CLI: one main Fix button, clear diagnosis/progress, support bundle export, and Developer Mode for expert tools.
- Provides package doctor/repair, transaction journal recovery, and safe-mode diagnostics so copied packages can fail cleanly or repair their local wrapper state.
- Provides Installer Center for Codex CLI, Codex App, Claude Code CLI, Gemini CLI, and Clash Verge Rev recovery from official sources.
- Can cache a Clash Verge Rev DMG in the Brain GUI ProxyKit package without enabling proxy/TUN automatically.
- Provides a hidden local Brain Chat Sandbox for explanation only; it cannot execute commands.
- Checks offline readiness for AgentLink rescue scope only.
- Saves and restores Last-Good AI-tool profiles with a user snapshot.
- Exports a redacted Support Bundle for Codex or a human helper.
- Checks Dev Essentials needed by AI CLI installers without managing project dependencies.
- Produces a MacBook Field Rescue package for copy-to-another-Mac Clash/TUN recovery.
- Can start a local `llama-server` on `127.0.0.1` for controlled local Gemma work; OpenCode Bridge remains experimental and not part of rescue execution.

## What It Does Not Do

- No GUI-owned repair logic.
- No model-generated shell execution.
- No model executor surface.
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
- No unofficial installer mirrors.
- No automatic proxy/TUN enablement.
- No Brain Chat command execution.
- No repo indexing, dependency cache manager, test runner, code patch engine, or local docs RAG inside AgentLink.

## Safety Model

- Diagnose is non-mutating and does not require root.
- Mutating rescue runs create restore points under:
  `/Library/Application Support/Cactus AgentLink Rescue/restore-points/`
- Third-party residue is quarantined, not deleted.
- System paths `/System`, `/bin`, `/usr/bin`, and `/usr/sbin` are never modified.
- Admin passwords are never accepted as arguments, stdin, or logs.
- `sudo` uses the normal macOS password prompt.
- Credentials in URLs, proxy env vars, git/npm/brew configs, and command logs are redacted.
- Command execution has timeouts and capped stdout/stderr so the GUI cannot hang indefinitely on large output.
- Mutating repairs write a durable journal under the user Application Support directory before mutation and update it through checkpoint, mutation, verification, rollback, and final status.
- Protected topology constraints force report-only behavior until route/interface ownership is proven; models and GUI surfaces cannot relax that boundary.
- Private keys, Wi-Fi passwords, browser cookies, and shell history are not collected.

## CLI Usage

```bash
agentlink diagnose [--json] [--verbose]
agentlink doctor [--json]
agentlink readiness doctor [--json]
agentlink dev doctor [--json]
agentlink last-good save [--name NAME] [--json]
agentlink last-good list [--json]
agentlink last-good inspect <id> [--json]
agentlink last-good restore [--last | --id ID] [--yes] [--json]
agentlink support bundle [--output PATH] [--json]
agentlink package doctor|repair [--package-root PATH] [--dry-run] [--yes] [--json]
agentlink journal list|inspect|recover [--yes] [--package-root PATH] [--json]
agentlink chaos list|run [--root PATH] [--fixture PATH] [--json]
agentlink --safe-mode doctor|support bundle|package doctor|journal list|journal recover
agentlink recipe list [--json]
agentlink recipe inspect <id> [--json]
agentlink recipe run <id> [--dry-run] [--yes] [--json] [--param key=value]
agentlink repair --target path|proxy|codex|keys [--dry-run] [--yes] [--json]
agentlink brain doctor [--json]
agentlink brain fetch [--model gemma-4-e4b-it-q4km] [--runtime llama.cpp]
agentlink brain selftest [--json]
agentlink brain prompt --text "..." [--json]
agentlink brain chat --prompt "..." [--json]
agentlink brain plan --target path|proxy|codex|keys|network [--json]
agentlink brain rescue-plan --target network|clash-tun [--json]
agentlink brain server start|stop|status|verify [--port 8080] [--json]
agentlink repair --auto --brain --target path|proxy|codex|keys|network [--dry-run] [--yes] [--online] [--json]
agentlink orchestrator rescue [--target auto|network|clash-tun|proxy|codex|keys|readiness] [--dry-run] [--yes] [--json]
agentlink guided rescue [--target auto|path|proxy|codex|keys|network] [--dry-run] [--yes] [--json]  # deprecated v0.4 compatibility alias
agentlink diagnose tun [--json]
agentlink verify network|airdrop [--json]
agentlink restart-gate prepare|verify [--json]
agentlink ticket create --type clash-tun-fix|rollback|verify-network [--id ID] [--json]
agentlink installer list [--json]
agentlink installer doctor [--json]
agentlink installer inspect <id> [--json]
agentlink installer dry-run <id> [--json]
agentlink installer install <id> --yes [--json]
agentlink installer verify <id> [--json]
agentlink installer open <id> [--json]
agentlink opencode doctor|install|configure-local-gemma|install-plugin|verify [--dry-run] [--yes] [--json]
agentlink planner validate <decision.json>
agentlink report --for-human --latest
agentlink report --for-codex --latest
agentlink classify [--json]
agentlink rescue [--level safe|tun|standard|clean-baseline|standard-system-reset|deep] [--yes] [--dry-run] [--json]
agentlink rollback [--last | --id RESTORE_POINT_ID] [--dry-run] [--json]
agentlink report [--latest | --id REPORT_ID] [--json]
agentlink selftest
agentlink version
```

Examples:

```bash
./bin/agentlink diagnose --json
./bin/agentlink package doctor --json
./bin/agentlink journal recover --json
./bin/agentlink rescue --level safe --dry-run
./bin/agentlink chaos run --fixture testdata/chaos/network/clash-tun-19818.json --json
./bin/agentlink orchestrator rescue --target auto --dry-run --json
./bin/agentlink orchestrator rescue --target clash-tun --dry-run --json
sudo ./bin/agentlink rescue --level tun --yes
sudo ./bin/agentlink rescue --level clean-baseline --yes
sudo ./bin/agentlink rescue --level standard-system-reset --yes
sudo ./bin/agentlink rescue --level deep --yes
./bin/agentlink ticket create --type clash-tun-fix --json
sudo ./bin/agentlink rollback --last
```

## Repair Levels

`safe` clears proxy state, DNS servers, search domains, DHCP, IPv6 automatic mode, renews DHCP on hardware devices, and flushes DNS cache.

`tun` is the preferred privileged repair when Clash/Mihomo TUN residue is detected. It stops Clash-family runtime, quarantines known residue, kickstarts NetworkExtension daemons, downs only stale matching `utun` interfaces, rebuilds active Wi-Fi DHCP/DNS/default route, refreshes mDNSResponder, and restores AWDL/sharingd for AirDrop discovery.

`standard` includes safe rescue and a lower-impact Wi-Fi refresh by default. The old broad clean-location/route-flush behavior is demoted to `standard-system-reset`.

`clean-baseline` requires `--yes`. It is a last-resort path for reaching a clean local online baseline after targeted repair, rollback, restart-gate verification, or support review shows no safer remaining action. It stops known interference runtimes, clears proxy/DNS/search/DHCP state, renews the active service, rebuilds the default route only from a safe current DHCP router, refreshes DNS/AWDL, and still does not enable Clash proxy/TUN.

`standard-system-reset` requires `--yes`. It is the explicit broad clean-location / route-flush / all-service DHCP reset path and is not the default for Clash/TUN signatures.

`deep` requires `--yes`. It includes safe and standard repair, quarantines eligible known residue, backs up selected network configuration plists, removes them by moving them into the restore point quarantine, and recommends reboot.

Protected topology note: if AgentLink detects a protected topology constraint, mutating rescue levels are refused and the next action is route/network snapshot, support bundle, or incident report. See `docs/offline/PROTECTED_TOPOLOGY_SAFETY.md`.

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
go test ./... -count=1 -timeout 35m
go vet ./...
staticcheck ./...
./bin/agentlink selftest
./bin/agentlink rescue --level safe --dry-run
./bin/agentlink rescue --level standard --dry-run
```

The v0.5.2 hardening pass additionally ran 64-way concurrency, chaos, race, audit, and fuzz pressure. Four fuzz harnesses live under:

```text
internal/safety/redact_fuzz_test.go
internal/planner/fuzz_test.go
internal/recipe/fuzz_test.go
internal/diagnosisgraph/fuzz_test.go
```

The latest local hardening run reported roughly 15.6M fuzz mutations across those harnesses with no panic or missed seeded secret-redaction contract.

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
dist/Cactus-AgentLink-Rescue-v0.5.1-core.zip
```

It can be copied to Downloads and launched with `agentlink.command`. The zip filename remains `v0.5.1-core` because the shared CLI core still reports `agentlink 0.5.1`; the current GUI/audit/portable beta line is v0.5.2.

To build the optional Brain package after fetching the model/runtime:

```bash
./scripts/fetch_brain_assets.sh
./scripts/package_brain.sh
```

Brain package output:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-brain-gemma4-e4b-q4km.zip
```

Native GUI packages:

```bash
./scripts/package_gui_core.sh
./scripts/package_gui_brain.sh
```

GUI package outputs:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-core-gui.zip
dist/Cactus-AgentLink-Rescue-v0.5.1-brain-gui-gemma4-e4b-q4km.zip
```

For a self-contained double-clickable portable beta, assemble the universal GUI binary and core package into a top-level app bundle plus launcher/checksum files. The latest verified portable beta is recorded in the "Current Verified Artifact" section above.

If a Clash Verge Rev DMG has been fetched, `scripts/package_gui_brain.sh` also emits:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-brain-gui-gemma4-e4b-q4km-proxykit.zip
```

For a copy-to-another-Mac Clash/TUN recovery build:

```bash
./scripts/package_macbook_field_rescue.sh
```

Field package output:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-macbook-field-gui-proxykit.zip
```

It unzips to `Cactus MacBook Network Rescue/` with `RUN-FIRST.command`, a short field README, emergency Terminal commands, and a GUI app that starts in MacBook Network Rescue mode. The GUI still does not run sudo, collect passwords, or enable Clash proxy/TUN automatically.

The GUI app embeds the CLI package under:

```text
Cactus AgentLink Rescue.app/Contents/Resources/agentlink/
```

The GUI uses the embedded `bin/agentlink`; it does not use a system `agentlink` from `PATH` unless a developer override is set. It uses `Process` with argument arrays, not shell command strings.

In normal mode the GUI is intentionally narrow: one primary Fix button, current health, last result, support bundle export, and Settings. The Fix button runs the CLI Rescue Orchestrator in analyze/dry-run mode first. If a selected repair needs administrator permission, the GUI shows a Terminal repair ticket instead of asking for a password.

Advanced controls are behind Settings -> Developer Mode. Developer Mode exposes the Expert Console: Doctor, Brain, Readiness, Installer Center, Plan, Dry Run, Rescue levels, Rollback, raw output, and experimental OpenCode Bridge status. Developer Mode is not needed for normal rescue.

### Core Package Vs Brain Package

The Core package is the network and recipe runtime. It does not include the Gemma GGUF model or llama.cpp runtime. Core can still run `doctor`, `diagnose`, recipe repairs, network rescue, reports, rollback, and `brain doctor`. In Core, `brain doctor` reports missing Brain assets and prints fetch commands.

The Brain package includes everything in Core plus:

- `assets/models/gemma-4-E4B-it-Q4_K_M.gguf`
- `assets/runtimes/llama.cpp/<arch>/llama-cli`
- required llama.cpp shared libraries
- manifests and licenses

Gemma 4 E4B-it Q4_K_M is the default Brain model because it is current, edge-friendly, and suitable for 16GB Macs. The model is roughly 5.07GB before package overhead.

On an offline Mac, use the Brain package if you need local Gemma planning. After unzip, these commands should run without network:

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

Source checkouts intentionally do not keep large Brain assets in git. For
reproducible local builds, put the GGUF model, llama.cpp runtime, and optional
Clash Verge Rev DMG cache under:

```text
~/CactusLocalAgent/.asset-cache/AgentLink-Rescue
```

or set:

```bash
export AGENTLINK_ASSET_CACHE="/path/to/AgentLink-Rescue-asset-cache"
```

Brain and ProxyKit package scripts search repo-local assets first, then this
external cache. The ignored payloads are `assets/models/*.gguf`,
`assets/runtimes/llama.cpp/`, `assets/installers/clash-verge-rev/**/*.dmg`,
`bin/`, `dist/`, GUI build outputs, and temporary zip files.

Gemma is not the executor. For rescue orchestration it returns validated `RescuePlanDecision` JSON; for older recipe planning it returns `PlannerDecision` JSON. Reports distinguish `brainAvailable` from `gemmaCalled`, so an installed Brain pack is not reported as model-supervised unless Gemma was actually invoked. The validator rejects unknown recipes, low-confidence repairs, privileged/destructive actions, and invalid JSON. The deterministic runner executes only bundled recipes, and writable repairs still require snapshot and rollback.

GUI dogfood:

```bash
./scripts/dogfood_gui_core.sh
./scripts/dogfood_gui_brain.sh
./scripts/dogfood_gui_screenshots.sh
```

The GUI is intentionally unsigned in this release. If macOS quarantine blocks it, run `xattr -cr "Cactus AgentLink Rescue.app"` on the extracted app. The GUI never asks for an admin password and never runs sudo; privileged network rescue commands are shown for copy/paste into Terminal.

### Installer Center And Proxy Recovery Kit

Installer Center is available from the GUI and CLI:

```bash
./bin/agentlink installer doctor
./bin/agentlink installer dry-run codex-cli
./bin/agentlink installer dry-run claude-code-cli
./bin/agentlink installer dry-run gemini-cli
./bin/agentlink installer open clash-verge-rev
```

Installer Center uses fixed official package names only: `@openai/codex`, `@anthropic-ai/claude-code`, `@google/gemini-cli`, Homebrew `codex`, and Homebrew `gemini-cli`. Codex App opens the official OpenAI page only. Clash Verge Rev uses the upstream GitHub releases cache when present.

Installer dependency checks are per method. If npm is missing but Homebrew is available, Codex CLI and Gemini CLI can still be available through Homebrew. If Homebrew is missing but npm is available, npm-backed methods can still be available. `missing_dependency` means no supported method is currently available.

AgentLink does not pipe curl into shell, use random mirrors, collect passwords, run sudo from the GUI, enable system proxy/TUN, or import proxy profiles automatically.

Fetch the optional Clash Verge Rev cache:

```bash
./scripts/fetch_clash_verge_rev.sh
```

### Experimental OpenCode Bridge

OpenCode Bridge is experimental in v0.5.1. Lab work showed OpenCode 1.14.33 could reach the local Gemma server, but OpenCode + Gemma failed 9/9 tasks across 18 attempts, mostly through provider/config failure and context overflow. AgentLink does not depend on OpenCode, does not claim OpenCode is configured or verified, and does not route system rescue through OpenCode. The packaged plugin is a scaffold only unless a later lab proves real loading.

### Brain Chat Sandbox

The Brain Chat Sandbox is a hidden/advanced local Gemma chat surface for explanation and report summarization. It is available from Settings. Triple-clicking the Dashboard title unlocks Developer Mode, not repair execution. The sandbox cannot execute commands, read files unless text is pasted, or change system settings. For repair, use the main Fix flow or Developer Mode Expert Console.

### Offline Readiness, Last-Good, And Support Bundle

`readiness doctor` checks only AgentLink rescue readiness: network/proxy state, Brain assets, recovery installers, API key presence, and Dev Essentials for AI CLI installers. It does not index repositories, manage dependency caches, run tests, or generate patches.

Last-Good Profiles only restore known AI config paths. Restored permissions preserve the original file mode when available and default to `0600` otherwise.

Support Bundle export is redacted but still includes local diagnostics such as tool presence, local paths, network/proxy status, readiness reports, and latest session metadata. It does not include private keys, browser cookies, shell history, Wi-Fi passwords, or full API keys.

`codex-config-parse-repair` is syntax recovery only. It writes a minimal valid TOML template when needed and does not prove provider/model online compatibility.

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
- Brain mode requires the Gemma 4 E4B GGUF and llama.cpp runtime. Core package remains functional without them.
- Gemma planning can recommend only registered recipes; it cannot invoke network rescue safe/standard/deep automatically.
- Some provider failures may be outside local repair scope.
- The fallback `rescue.sh` is intentionally simpler than the Go engine and should be used only when the binary is blocked.
