# Volume 01 — macOS Network Stack Failure Space

## Control planes to separate

macOS local networking is controlled by several overlapping planes:

1. Network services and service order.
2. Interface state and link state.
3. DHCP/IP configuration.
4. Routing table and default route.
5. System DNS and scoped DNS resolvers.
6. System proxy and PAC settings.
7. VPN and Network Extension state.
8. pf/application firewall/content filters.
9. Shell/application-level environment variables.
10. App-specific config files.

AgentLink Rescue must never assume these planes are aligned.

## Universal observation sequence

Read-only first:

```bash
sw_vers
uname -a
networksetup -listallnetworkservices
networksetup -listnetworkserviceorder
scutil --dns
scutil --proxy
scutil --nc list
ifconfig
netstat -rn
route -n get default
```

Connectivity ladder:

```bash
ping -c 2 <gateway>
ping -c 2 1.1.1.1
ping -c 2 8.8.8.8
dscacheutil -q host -a name example.com
curl -I --connect-timeout 5 https://example.com
curl -I --noproxy '*' --connect-timeout 5 https://example.com
```

## Common macOS failure clusters

### Link and service order

Symptoms:

- Wi-Fi says connected but no internet.
- Ethernet connected but ignored.
- USB-C adapter appears but route goes through Wi-Fi.
- Dante network should use Ethernet but Mac selects Wi-Fi for discovery.

Likely causes:

- Wrong network service priority.
- Disabled network service.
- Link speed degraded.
- USB/Thunderbolt adapter not enumerated.
- Wi-Fi and Ethernet are on different networks but route expectation is ambiguous.

### DHCP/IP

Symptoms:

- IP begins with 169.254.
- Gateway missing.
- DNS server present but unreachable.
- LAN devices invisible.

Likely causes:

- DHCP lease failure.
- Wrong VLAN.
- No DHCP server on audio-only network.
- Static IP misconfigured.
- IPv6-only or IPv4-only expectation mismatch.

### Routing

Symptoms:

- Browser works but LAN device unreachable.
- VPN works but local Dante/AES67 disappears.
- CLI tools route via utun while browser goes via proxy.
- Raw IP ping works but domain does not.

Likely causes:

- Default route captured by VPN/TUN.
- Split route missing.
- Service order changed.
- IPv6 preferred path broken.
- Local subnet route shadowed by a broader VPN route.

### DNS

Symptoms:

- `ping 1.1.1.1` works but `example.com` fails.
- One app resolves; another does not.
- VPN domain works only when VPN active.
- Clash fake-ip resolves but LAN hostname fails.

Likely causes:

- Scoped resolver mismatch.
- Stale DNS cache.
- DNS server unreachable from active route.
- TUN/fake-ip DNS interaction.
- Corporate split DNS unavailable.

### Proxy

Symptoms:

- Browser works but Terminal fails.
- Terminal works but browser fails.
- Codex/API/Git/npm/brew fail while Safari works.
- `curl --noproxy '*'` works but normal `curl` fails, or vice versa.

Likely causes:

- Stale system proxy.
- Dead local proxy port.
- Environment variable points to old proxy.
- PAC file used by browser but ignored by CLI.
- `NO_PROXY` missing for LAN/localhost.

### VPN / Network Extension / TUN

Symptoms:

- Internet dies after enabling VPN/TUN.
- LAN dies after enabling VPN.
- DNS changes after enabling Clash Verge TUN.
- Multiple `utun` interfaces appear and default route changes.

Likely causes:

- Kill-switch policy.
- DNS hijack or DNS server substitution.
- Route capture.
- Split tunnel exclusion missing.
- Network Extension stale state.

## Repair design

The default v0.5 repair order should be:

1. Snapshot.
2. Read-only diagnosis.
3. User-level reversible changes.
4. System-level changes only with confirmation.
5. High-risk pro-audio changes never auto-executed.
