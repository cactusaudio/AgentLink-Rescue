# GPT Pro Pre-Release Stress Guide

This package is for final cloud-sandbox pressure testing of Cactus AgentLink Rescue before product release.

It intentionally excludes Gemma, Qwen, GGUF weights, llama.cpp runtimes, installer DMGs, secrets, and any host-specific asset cache.

## What This Package Can Prove

- The AgentLink deterministic diagnosis/classification kernel handles common macOS network failure fixtures.
- The sandbox maze pack exercises common rescue situations without host mutation.
- The executor, transaction, support-bundle, package-health, chaos, and adversarial gates remain green in a clean sandbox.
- The CLI can be built and run from source in a cloud environment.
- The model-facing safety boundary remains testable: fixture and chaos suites must not require `sudo`, real `networksetup`, real `route`, real `ifconfig`, real `launchctl`, or real host repair.

On non-macOS cloud hosts, the CLI intentionally exposes only fixture, read-only, and dry-run surfaces:

```text
version
selftest
doctor
manifest
chaos list/run
diagnose --json
diagnose-graph --from <DiagnosticReport>
recipe list/inspect
recipe run <id> without --yes (forced dry-run on non-Darwin)
support bundle
package doctor
package repair --dry-run
journal list/inspect/recover without --yes
readiness
dev doctor
rollback --dry-run
last-good list/inspect
brain doctor
```

All mutating rescue, repair, rollback, restore, proxy, keychain, installer, and system-verification commands remain macOS-only.

## What This Package Cannot Prove

- It does not prove real macOS GUI behavior in GPT Pro cloud unless the sandbox is a macOS GUI host.
- It does not prove a real Mac network repair, because the cloud sandbox is not Bowei's Mac.
- It does not prove Gemma or Qwen behavior, because model assets are deliberately excluded.
- It does not prove signed/notarized distribution.

Those surfaces require a real macOS host, the GUI app bundle, and/or explicitly provisioned model assets.

## Hard Safety Rules For GPT Pro

Do not run:

```text
sudo
agentlink rescue --yes
agentlink guided rescue --yes
networksetup mutation
route mutation
ifconfig mutation
launchctl mutation
rm -rf outside a temporary sandbox
```

Allowed cloud-sandbox actions:

```text
go test ...
go vet ...
agentlink version
agentlink chaos list
agentlink chaos run --fixture ...
agentlink diagnose-graph --from ...
agentlink recipe run <id> --dry-run
bash scripts/run_sandbox_mazes.sh
bash scripts/gpt_pro_stress_test.sh
```

If GPT Pro discovers a failing fixture or timeout, it should preserve logs and report the exact command, exit code, stderr/stdout, and whether the failure is product, harness, cloud-environment, or stale-binary contamination.

`unsupported platform` on a fixture/read-only/dry-run command is a package/cloud-gate failure. `unsupported platform` on a real mutating command is expected.

## Quick Start

From the extracted package root:

```bash
cd AgentLink-Rescue
bash scripts/gpt_pro_stress_test.sh
```

The short root-level safety entrypoint is:

```text
README_FIRST_FOR_GPT_PRO.md
```

Results are written under:

```text
AgentLink-Rescue/gpt-pro-results/<timestamp>/
```

The key file is:

```text
SUMMARY.json
```

## Required Result Expectations

Minimum acceptable green state:

- `02-sandbox-mazes.json`: `failed == 0`, `passed >= 25`.
- `03-chaos-fixtures.log`: Go test passes.
- `04-core-units.log`: Go test passes.
- `05-v0300-v0320-chaos.log`: Go test passes.
- `06-v0330-rc-chaos.log`: Go test passes.
- If `FULL=1`, `07-full-go-test.log` and `08-go-vet.log` pass.

Known timing notes:

- `TestV0330` can take more than 10 minutes in some environments. The provided script uses `-timeout 30m`.
- macOS `/bin/zsh` mutation/rollback tests are skipped on non-Darwin hosts. They remain covered by the macOS release gates and are not evidence of Linux rescue support.

## Manual Deep Commands

Run only if the quick start fails and more isolation is needed.

```bash
go version
go env GOOS GOARCH
go build -trimpath -ldflags "-s -w" -o /tmp/agentlink ./cmd/agentlink
/tmp/agentlink version
/tmp/agentlink chaos list --root testdata/chaos --json
AGENTLINK_BIN=/tmp/agentlink SKIP_BUILD=1 bash scripts/run_sandbox_mazes.sh
AGENTLINK_BIN=/tmp/agentlink go test ./internal/chaos -run TestChaosFixturesPass -count=1
AGENTLINK_BIN=/tmp/agentlink go test ./internal/chaos -run 'TestV0300|TestV0301|TestV0320' -count=1 -timeout 20m
AGENTLINK_BIN=/tmp/agentlink go test ./internal/chaos -run 'TestV0330' -count=1 -timeout 30m
go test ./internal/packagehealth ./internal/journal ./internal/command ./internal/supportbundle -count=1
go test ./... -count=1 -timeout 30m
go vet ./...
```

## Stale Binary Guard

Always set `AGENTLINK_BIN` to the binary built from the extracted source.

Do not rely on `~/CactusLocalAgent/.asset-cache/AgentLink-Rescue/bin/agentlink` or any host-level binary. Older local binaries previously caused false timeout attribution in the RC stress suite.

The packaged script builds `.gpt-pro-build/agentlink-<goos>-<goarch>` from the extracted source when the binary is absent. The release package should not include a prebuilt `.gpt-pro-build` binary.

## Package Integrity

Run:

```bash
shasum -a 256 -c CHECKSUMS.txt
```

`CHECKSUMS.txt` deliberately does not include itself. A self-checksum failure is a packaging defect, not a product result.

## Report Template For GPT Pro

GPT Pro should return:

```text
AgentLink GPT Pro Pre-Release Stress Result

Package hash:
Host OS/arch:
Go version:

Suites:
- sandbox mazes:
- chaos fixtures:
- core units:
- v0300/v0301/v0320:
- v0330:
- full go test:
- go vet:

Failures:
- command:
- exit code:
- stdout/stderr:
- classification: product | harness | cloud environment | stale binary | inconclusive

Safety:
- sudo executed? no
- real network mutation executed? no
- model weights present? no
- secrets found? no

Recommendation:
- release candidate holds | fix required | rerun required
```

## Included Sandbox Maze Pack

The package includes at least 25 common and protected-topology network situations under:

```text
testdata/chaos/network/mazes/
```

They cover:

- healthy Wi-Fi
- system proxy pollution
- shell proxy pollution
- Git/npm/Brew proxy pollution
- HTTPS-only captive portal / TLS interference
- no default route
- DHCP link-local only
- gateway unreachable
- agent endpoint blocked only
- Clash/Mihomo TUN + AirDrop degraded
- stale Network Extension suspicion
- managed MDM profile suspicion
- protected Dante/AES67 audio route traps
- NDI/video, ATEM/PTZ, Art-Net/sACN, NAS/iSCSI, Thunderbolt/direct-link, lab/PLC, VM bridge, MDM policy, and USB-management protected-topology boundaries
- a TUN-owned control case where targeted TUN repair is still allowed
- an unbound raw-audio false-positive guard

Every maze is fixture-driven and goes through the real classifier. Fixtures do not hand-set classifications.
