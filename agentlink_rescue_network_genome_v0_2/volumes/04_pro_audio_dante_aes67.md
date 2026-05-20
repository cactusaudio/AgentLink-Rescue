# Volume 04 — Dante / RAVENNA / AES67 / Pro-Audio IP Networking Failure Space

Pro-audio IP networking is not just ordinary internet troubleshooting. A network can have valid IP reachability and still fail audio transport because clocking, multicast, packet timing, QoS, or device configuration is wrong.

## Separate the planes

1. Physical/link: cabling, switch ports, link speed, NIC choice.
2. IP addressing: same subnet or routable path, static IP/DHCP expectation.
3. Discovery: Dante Controller visibility, mDNS/Bonjour, vendor discovery, static discovery.
4. Clocking: Dante PTP, AES67 PTPv2, grandmaster/leader, domain, BMCA behavior.
5. Stream description: channel count, sample rate, bit depth, packet time, SDP.
6. Media transport: RTP multicast/unicast, buffer/latency, packet loss.
7. Switch behavior: IGMP snooping, querier, multicast flooding/drop, QoS, EEE, VLANs.
8. Application mapping: subscriptions, device names, transmit/receive flows, driver selection.

## Dante-specific clusters

### Devices invisible

Potential causes:

- Mac using wrong NIC.
- Dante Controller bound to wrong interface.
- Devices on a different subnet/VLAN.
- Wi-Fi selected instead of wired Dante network.
- mDNS/discovery blocked.
- Primary/secondary ports mixed.

### Devices visible but no audio

Potential causes:

- Subscription not made or failed.
- Sample rate mismatch.
- Clock not synchronized.
- Latency too low for network path.
- Receiver/transmitter channel mapping wrong.
- Device muted due to clock instability.

### Clock unstable

Potential causes:

- PTP blocked or filtered.
- Multiple unintended leaders.
- Switch behavior disturbing timing.
- EEE/green ethernet causing jitter.
- VLAN/QoS misconfiguration.

## AES67/RAVENNA-specific clusters

### Stream visible but cannot subscribe

Potential causes:

- SDP incompatible or missing.
- PTP domain/profile mismatch.
- Sample rate/packet time/channel count mismatch.
- Multicast route/IGMP issue.
- Device supports AES67 nominally but not the required mode/firmware.

### Audio dropouts/glitches

Potential causes:

- Multicast flooding.
- IGMP snooping without a querier or wrong querier behavior.
- QoS absent on congested links.
- EEE enabled.
- Packet time/buffer too aggressive.
- Clock not locked.

## Runtime stance

AgentLink Rescue should never auto-change pro-audio switch configuration. It should:

1. Detect likely layer.
2. Provide exact observation checklist.
3. Preserve Dante/AES67 network settings.
4. Ask user to export Dante Controller logs/screenshots if available.
5. Offer manual steps, not one-click mutation, for clocking, switch, VLAN, QoS, and multicast.

## Pro-audio snapshot checklist

```text
Mac NIC used for audio:
IP/subnet/gateway of audio NIC:
Wi-Fi enabled during audio session? yes/no:
Dante Controller selected interface:
Visible devices:
Sample rates:
Clock leader / preferred leader:
Clock status / sync / mute status:
Primary/secondary network wiring:
Switch model:
Managed/unmanaged:
IGMP snooping:
IGMP querier:
EEE/green ethernet:
QoS trust/DSCP:
VLANs:
AES67 mode enabled:
PTPv2 enabled:
PTP domain:
SDP source:
```
