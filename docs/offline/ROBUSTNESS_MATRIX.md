# Robustness Matrix

v0.5.2 field-beta readiness is measured by invariants, not by optimistic command output.

## Core

- `package doctor` runs quickly and is read-only.
- `doctor` succeeds without Brain assets.
- Dry-run does not mutate files, routes, proxy, DNS, or app support state.
- Support bundle export succeeds or fails with a clear error.
- GUI and CLI command runners use timeouts and capped output.
- Private diagnostic files default to `0600`.
- Private AgentLink-owned diagnostic directories default to `0700`.
- Config-file patching preserves an existing mode instead of widening it.

## Network

- High-confidence Clash/Mihomo TUN signatures recommend targeted `tun` repair, not `standard`.
- `standard` does not run broad system reset.
- `standard-system-reset` and `deep` remain explicit fallback levels.
- Postflight worsening triggers rollback.
- Suspicious TUN gateways such as `198.18.0.1`, `198.18.0.0/15`, or `fdfe:dcba:9876::1` are never restored as default route.

## Rollback

- `rollback --id` loads persisted network snapshot evidence when present.
- Corrupted restore points fail cleanly.
- Partial rollback is reported as partial; it is not called success.
- Running rollback a second time should return already-restored/no-op style behavior where possible.

## Package

- Core zip does not include GGUF model files or llama runtime.
- Brain packages include model/runtime only when explicitly built with assets.
- ProxyKit packages include Clash DMG only when the asset is present.
- Zips contain no Finder metadata.
- Source tree does not include dist zips, runtime binaries, GGUF, or DMG assets.
- Support-bundle zip creation skips symlinks and non-regular files.
- Portable field-beta packages include checksum verification and a fallback launcher.
- Portable GUI and embedded CLI binaries are universal (`arm64` + `x86_64`).

## UX

- Main path remains one button.
- Developer Mode contains expert controls.
- Disabled actions state their reason.
- Admin repair uses Terminal tickets, not GUI password capture.
- Incomplete transactions appear as recovery work, not as hidden stale state.

## Latest Pressure Surface

Post-alpha hardening added four fuzz harnesses:

- `internal/safety/redact_fuzz_test.go`
- `internal/planner/fuzz_test.go`
- `internal/recipe/fuzz_test.go`
- `internal/diagnosisgraph/fuzz_test.go`

The latest pressure run reported roughly 15.6M fuzz mutations across those
harnesses, 64-way diagnosis concurrency, race tests, chaos fixtures, and the
v0.5.2 GUI field-beta audit green. Remaining gosec findings are classified in
the hardening notes as intentional design or static-analysis false positives.
