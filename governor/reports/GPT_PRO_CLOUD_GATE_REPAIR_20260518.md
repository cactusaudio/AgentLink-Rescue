# GPT Pro Cloud Gate Repair - 2026-05-18

## Verdict

The GPT Pro failure report was valid. The prior stress package claimed Linux cloud fixture/read-only pressure could run, but the CLI blocked every non-`version`/`selftest` command on non-Darwin before the fixture kernel could execute.

This was a package/cloud-gate contract failure, not evidence that the macOS AgentLink rescue kernel failed.

## Root Cause

`internal/cli/cli.go` had a top-level non-Darwin guard:

```go
if cmd != "version" && cmd != "selftest" && !system.IsDarwin() {
    fmt.Fprintln(stderr, "unsupported platform: agentlink supports macOS only")
    return 60
}
```

That incorrectly blocked pure fixture and dry-run surfaces such as:

- `chaos list`
- `chaos run --fixture`
- `diagnose-graph --from`
- `recipe list`
- `recipe run --dry-run`
- `support bundle`
- v0300/v0320/v0330 chaos matrix shell-outs

## Fixes

- Replaced the broad non-Darwin guard with a command-specific cloud whitelist.
- Allowed only fixture, read-only, and dry-run surfaces on non-Darwin.
- Continued to block mutating rescue/repair/restore/proxy/keychain/installer/system-verification commands on non-Darwin.
- Forced `recipe run <id>` without `--yes` into dry-run mode on non-Darwin.
- Added `journal recover --dry` compatibility for the cloud chaos harness.
- Skipped `/bin/zsh` mutation/rollback tests on non-Darwin; those remain covered by macOS release gates.
- Added CLI unit tests for allowed and denied non-Darwin cloud surfaces.
- Added `scripts/package_gpt_pro_stress.sh` so future packages exclude model/runtime/archive artifacts and generate `CHECKSUMS.txt` without self-checksum failure.
- Updated `docs/testing/GPT_PRO_PRERELEASE_STRESS_GUIDE.md` with precise cloud contract limits.

## Verification

Run on Mac Studio from `/Users/jack/CactusLocalAgent/AgentLink-Rescue`:

```text
go test ./... -count=1 -timeout 30m
go vet ./...
bash scripts/build.sh
REPORT_PATH=/tmp/agentlink-sandbox-mazes-fixed.json bash scripts/run_sandbox_mazes.sh
FULL=0 OUT=/tmp/agentlink-gpt-pro-fixed-smoke bash scripts/gpt_pro_stress_test.sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /tmp/agentlink-linux-amd64-fixed ./cmd/agentlink
bash scripts/package_gpt_pro_stress.sh
cd <package-root> && shasum -a 256 -c CHECKSUMS.txt
FULL=0 OUT=/tmp/agentlink-gpt-pro-package-062655-smoke bash scripts/gpt_pro_stress_test.sh
```

Observed:

- `go test ./...`: pass, including `internal/chaos` in `748.081s`.
- `go vet ./...`: pass.
- Sandbox mazes: `12/12`.
- Source-tree GPT Pro smoke: completed through `06-v0330-rc-chaos`, no `unsupported platform` in logs.
- Packaged-copy GPT Pro smoke for the final zip staging tree: completed through `06-v0330-rc-chaos`, no `unsupported platform` in logs.
- Linux/amd64 cross-build: pass.
- Package checksum verification: pass.
- Zip integrity: pass.
- Forbidden package artifacts: none found.

## New Package

```text
/Users/jack/CactusLocalAgent/.asset-cache/AgentLink-Rescue/release-artifacts/gpt-pro-stress-lite-20260518T062655Z/agentlink-gpt-pro-prerelease-stress-lite-20260518T062655Z.zip
```

SHA256:

```text
0a01ccd9cd20a05e458ffb8e05b01cb5181cb27bf7e3007616ea09e0d8c54b52
```

## Remaining Truth Boundary

This package proves the cloud fixture/read-only/dry-run stress contract. It still does not prove:

- real macOS network mutation,
- real macOS GUI behavior,
- signed/notarized distribution,
- Gemma or Qwen model behavior.

Those remain separate macOS/model release surfaces.
