# Terminal Repair Tickets

The native GUI never collects passwords and never runs sudo directly.

When a repair needs administrator privileges, AgentLink creates a Terminal repair ticket:

```bash
agentlink ticket create --type clash-tun-fix --json
```

The ticket folder contains executable `.command` files:

- `Run-Clash-TUN-Fix.command`
- `Verify-Network.command`
- optional rollback tickets

The files contain no password, no `sudo -S`, and no hidden helper. Terminal asks for the user's password normally.

