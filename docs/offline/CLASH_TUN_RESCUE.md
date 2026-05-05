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

This targeted repair stops Clash/Mihomo runtime, boots out launch items, quarantines known residue, kickstarts macOS NetworkExtension daemons, downs stale matching `utun` interfaces, refreshes active Wi-Fi DHCP/DNS, rebuilds default route only from the current safe DHCP router when needed, refreshes mDNSResponder, and restores AWDL/sharingd for AirDrop discovery.

It never reuses a preflight TUN gateway such as `198.18.0.1` as a DHCP gateway. It does not create a clean network location, does not route-flush blindly, does not permanently delete proxy app data, and does not enable proxy or TUN.

If targeted TUN repair and restart-gate verification still fail, the operator may offer `clean-baseline` as the first last-resort reset:

```bash
sudo ./bin/agentlink rescue --level clean-baseline --yes
```

`clean-baseline` tries to reach a clean local online baseline without enabling Clash proxy/TUN. It is still explicit, terminal-assisted, checkpointed, and user-approved.

`standard-system-reset` and `deep` are broader explicit fallback layers only. They are not the default when Clash/Mihomo TUN signatures are present.
