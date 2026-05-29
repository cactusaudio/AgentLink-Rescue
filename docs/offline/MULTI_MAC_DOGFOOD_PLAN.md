# Multi-Mac Dogfood Plan

This is a manual checklist for v0.5.2 field-beta portability. It is not a release blocker for every local build, but it defines the next real-world confidence surface.

## Machines

- Apple Silicon 16 GB
- Apple Silicon 64 GB or larger
- Intel Mac
- macOS 13, 14, 15, and current beta when available
- Admin account and non-admin account

## Package Shapes

- Core CLI package
- Core GUI package
- Brain GUI package
- ProxyKit package
- MacBook Field Rescue package
- Package copied to Downloads
- Package copied to a path with spaces
- Package opened after quarantine/xattr
- Portable field-beta folder with top-level `Cactus AgentLink Rescue.app`
- Portable zip verified by `VERIFY-CHECKSUMS.command`

## Host Variants

- No Homebrew
- No Node/npm
- Tailscale installed
- Clash/Clash Verge/ClashX/Mihomo installed
- Surge or WARP installed
- USB Ethernet active
- Wi-Fi only
- AirDrop receive degraded

## Read-Only Pass

Run on every machine:

```bash
./bin/agentlink package doctor --json
./bin/agentlink doctor --json
./bin/agentlink readiness doctor --json
./bin/agentlink support bundle --json
./bin/agentlink orchestrator rescue --target auto --dry-run --json
./bin/agentlink guided rescue --target auto --dry-run --json
./bin/agentlink package repair --dry-run --json
```

GUI checks:

- App launches.
- Main view has one Fix button.
- Developer Mode is hidden by default.
- Package health badge is visible.
- No mutation happens without explicit consent.
- No GUI sudo/password prompt appears.
- `Run Field Beta Check` returns ready or ready-with-warnings without mutation.
- Support bundle export succeeds and does not expose raw secrets.
- The app and embedded CLI report universal binaries on both Intel and Apple Silicon Macs.

## Mutating Pass

Only run on expendable test machines or after a manual checkpoint:

- `rescue --level safe --yes`
- `rescue --level tun --yes` only with high-confidence Clash/Mihomo TUN fixture or real condition
- `rollback --id` for the restore point
- `journal recover --json` after simulated interruption

Do not run clean-baseline, standard-system-reset, or deep on a primary workstation unless Bowei explicitly approves that test.
