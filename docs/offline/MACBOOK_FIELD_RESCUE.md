# MacBook Field Rescue

The v0.5.x line can build a copy-to-another-Mac package for Clash Verge
TUN-related network breakage:

```bash
./scripts/package_macbook_field_rescue.sh
```

Output:

```text
dist/Cactus-AgentLink-Rescue-v0.5.1-macbook-field-gui-proxykit.zip
```

The filename still carries `v0.5.1` because the shared CLI core reports
`agentlink 0.5.1`. The current GUI/audit/portable-beta line is v0.5.2.

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
