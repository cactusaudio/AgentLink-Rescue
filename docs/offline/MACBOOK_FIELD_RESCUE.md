# MacBook Field Rescue

v0.4.5 can build a copy-to-another-Mac package for Clash Verge TUN-related network breakage:

```bash
./scripts/package_macbook_field_rescue.sh
```

Output:

```text
dist/Cactus-AgentLink-Rescue-v0.4.5-macbook-field-gui-proxykit.zip
```

The package unzips to `Cactus MacBook Network Rescue/` and includes:

- `Cactus AgentLink Rescue.app`
- `RUN-FIRST.command`
- `README-MACBOOK-NETWORK-RESCUE.txt`
- `emergency-terminal-commands.txt`

The app embeds the Brain GUI ProxyKit resources plus `field-mode.json`, so it starts in MacBook Network Rescue mode. It highlights Clash/TUN recovery, runs read-only network analysis, and shows copyable Safe/Standard/Deep Terminal commands.

Safety boundaries:

- GUI does not run sudo.
- GUI does not collect passwords.
- AgentLink does not enable Clash proxy/TUN automatically.
- `agentlink field macbook-network-rescue` is read-only.
- Network standard/deep repair remains explicit Terminal execution.
- Deep repair should be used only after safe/standard fail or a human helper instructs it.
