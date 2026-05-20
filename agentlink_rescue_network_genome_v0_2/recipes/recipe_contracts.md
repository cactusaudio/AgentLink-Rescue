# AgentLink Rescue v0.2 Recipe Contracts

The recipe layer is intentionally conservative. v0.2A expands failure knowledge; v0.2B compiles topology. It is not a blind auto-fixer.

## Contract

1. **Read-only snapshot first.** Every guided run starts with collectors.
2. **No recipe without verifier.** A suggested change must name the exact verifier that should improve.
3. **No mutating recipe without rollback.** The previous value/location/rule/profile must be captured.
4. **AoIP is manual-only.** Dante/AES67/RAVENNA/PTP/switch/QoS/VLAN changes are never executed automatically.
5. **Policy-managed systems are report-only.** MDM/VPN/EDR/profile-locked settings produce an admin-ready report.

## v0.5 implementation hint

Represent recipes as state transitions:

```yaml
precondition_observations: []
change: []
verifier: []
rollback: []
risk: low|medium|high|never_auto
owner: system_proxy|shell_env|vpn_app|clash_core|tool_config|switch|audio_controller
```

The runtime should refuse any transition where `verifier` or `rollback` is absent, except pure read-only reports.
