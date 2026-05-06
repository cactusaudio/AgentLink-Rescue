# Robustness Matrix

v0.5.1 release readiness is measured by invariants, not by optimistic command output.

## Core

- `package doctor` runs quickly and is read-only.
- `doctor` succeeds without Brain assets.
- Dry-run does not mutate files, routes, proxy, DNS, or app support state.
- Support bundle export succeeds or fails with a clear error.
- GUI and CLI command runners use timeouts and capped output.

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

## UX

- Main path remains one button.
- Developer Mode contains expert controls.
- Disabled actions state their reason.
- Admin repair uses Terminal tickets, not GUI password capture.
- Incomplete transactions appear as recovery work, not as hidden stale state.
