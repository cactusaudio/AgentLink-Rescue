# AgentLink Rescue v0.2 Rollback Templates

## System network settings

- Save Network Location name and service order before changes.
- Save per-service DNS/search domains/proxy settings before changes.
- Restore exact previous values, not generic defaults, after failed verifier.

## Shell/app environment

- Store previous env values in report.
- Use session-only unsets before changing shell dotfiles.
- For LaunchAgent/GUI environment, document owning file/process before mutation.

## Tool configs

- Use `git config --show-origin`, `npm config list`, Docker config snapshots, Homebrew env files.
- Prefer commenting/renaming stale config only after backup.

## VPN/Clash/TUN

- Restore proxy/DNS/route snapshot after disabling tunnel.
- Do not edit route table manually when owning app can restore state.
- Keep provider/rule/profile diff for rollback.

## AoIP

- Screenshot controller/switch state before changes.
- Revert one variable at a time.
- Never auto-change clock, sample rate, VLAN, IGMP, QoS, or routing matrix.
