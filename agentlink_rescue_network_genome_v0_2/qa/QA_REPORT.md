# QA Report — AgentLink Rescue Network Genome Pack v0.2

Generated: 2026-05-19 America/Los_Angeles

## Validation

```json
{
  "cards": 300,
  "layers": {
    "L00_physical_link": 12,
    "L02_ip_dhcp": 18,
    "L03_routing": 20,
    "L04_dns": 26,
    "L05_proxy": 32,
    "L06_vpn_ne_tun": 24,
    "L07_firewall_pf": 14,
    "L08_app_connectivity": 42,
    "L09_tls_identity": 16,
    "L10_multicast_discovery": 22,
    "L12_aoip_media": 28,
    "L11_ptp_clock": 18,
    "L13_external_provider": 7,
    "L01_interface_service": 16,
    "L14_user_policy": 5
  },
  "risks": {
    "medium": 82,
    "low": 93,
    "high": 37,
    "never_auto": 81,
    "read_only": 7
  }
}
```

## Completion status

- v0.2A: complete as a 300-card structured corpus.
- v0.2B: complete as topology graph + diagnosis DAG + symptom routes + decision tables + retrieval indexes.

## Counts

- Cards: 300
- Layers: 15
- Source anchors: 16
- Topology layer edges: 23
- Topology card/observation/verifier edges: 3404
- SQLite FTS enabled: True

## Risk distribution

high: 37
low: 93
medium: 82
never_auto: 81
read_only: 7


## Design audit

The biggest remaining weakness is not coverage but **empirical calibration**. v0.2 is a high-quality synthetic/structured seed, not yet a corpus of thousands of real-world incident traces. The next step after Codex implementation should be to feed real snapshots and outcomes back into card scoring, false-positive pruning, and recipe safety gates.

## Hard blockers intentionally preserved

- AoIP switch/PTP/QoS/VLAN/sample-rate changes remain manual-only.
- MDM/corporate policy remains report-only.
- TLS verification bypass remains disallowed as a durable repair.
- pf global flush remains disallowed.
