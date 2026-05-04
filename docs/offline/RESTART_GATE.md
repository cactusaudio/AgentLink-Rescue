# Restart Gate

Restart is a structured gate, not a random suggestion.

AgentLink shows Restart Required only after safe non-restart actions have been attempted:

- Clash/Mihomo runtime stop.
- Launch item/helper quarantine.
- NetworkExtension daemon kickstart.
- stale matching `utun` down.
- Wi-Fi DHCP/DNS/default-route repair.
- AWDL/sharingd refresh.
- verification still fails or stale TUN persists.

Before restart, do not reopen Clash, ClashX, Clash Verge, or Mihomo. After restart, run:

```bash
./bin/agentlink restart-gate verify --json
```

