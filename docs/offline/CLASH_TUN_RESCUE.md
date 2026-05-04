# Clash / Mihomo TUN Rescue

AgentLink v0.5.0 treats stale Clash, Clash Verge, ClashX, and Mihomo TUN state as a targeted failure class, not a generic network reset.

High-confidence signatures:

- `utun*` contains `198.18.0.1`.
- `utun*` contains an IPv4 address in `198.18.0.0/15` plus Clash/Mihomo residue.
- `utun*` contains `fdfe:dcba:9876::1`.
- Clash/Mihomo LaunchAgent, LaunchDaemon, privileged helper, or app data residue is present.
- System proxy is clean but raw IP or HTTPS traffic is blackholed.

Preferred command:

```bash
sudo ./bin/agentlink rescue --level tun --yes
```

This targeted repair stops Clash/Mihomo runtime, quarantines known residue, kickstarts macOS NetworkExtension daemons, downs stale matching `utun` interfaces, rebuilds Wi-Fi DHCP/DNS/default route, refreshes mDNSResponder, and restores AWDL/sharingd for AirDrop discovery.

It does not create a clean network location, does not route-flush blindly, does not permanently delete proxy app data, and does not enable proxy or TUN.

