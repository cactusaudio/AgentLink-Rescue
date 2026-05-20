# AoIP manual snapshot checklist — Dante / RAVENNA / AES67

v0.2 keeps professional-audio network changes manual-only. Collect evidence before changing clock, sample rate, VLAN, IGMP, QoS, or switch configuration.

## Capture

- Mac interface used by controller / virtual soundcard / DAW.
- IP, subnet, link speed, switch port, VLAN for each AoIP device.
- Dante Controller screenshots: Routing matrix, Device Info, Clock Status, Network Status, Device Config/sample rate.
- AES67/RAVENNA: SDP, SAP/NMOS/discovery status, PTP profile/domain, multicast group, packet time, sample rate, channel count.
- Switch: IGMP snooping, querier, QoS/DSCP queue mapping, EEE/green ethernet, storm-control, VLAN trunk/access membership, port errors/drops.
- Whether VPN/Clash/TUN/Wi-Fi is active on the Mac during the test.

## Do not auto-change

- Clock leader/grandmaster.
- PTP domain/profile/version.
- Sample rate / pull-up/down.
- VLAN trunk/access membership.
- IGMP snooping/querier.
- QoS/DSCP queue mapping.
- Primary/secondary Dante wiring.

## Minimal verifier sequence

1. Device visibility stable on intended interface.
2. Clock lock stable for observation window.
3. Subscription status healthy.
4. Media counters stable under expected load.
5. Audio passes cleanly after one variable change at a time.
