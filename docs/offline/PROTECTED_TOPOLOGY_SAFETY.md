# Protected Topology Safety

AgentLink must not make a machine "cleaner" by damaging an intentional local
network. Dante/AES67 audio VLANs, NDI/video VLANs, ATEM/PTZ camera control,
Art-Net/sACN lighting, NAS/iSCSI storage links, Thunderbolt/direct links,
lab/PLC control LANs, VM bridges, USB management adapters, and MDM/VPN policy
surfaces are treated as repair boundaries.

## Product Rule

Protected topology is not automatically the root cause. It is a mutation
boundary:

1. Detect protected constraints first.
2. Classify root cause candidates separately.
3. Mark red herrings such as TUN/fake-IP, bridge interfaces, or DNS symptoms
   only after route ownership is proven.
4. Filter every mutating repair through `topology.repairCorridor`.
5. If the corridor forbids mutation, AgentLink emits snapshots, support bundle,
   and incident-ticket guidance instead of running repair.
6. Model, GUI, and terminal-ticket surfaces cannot relax the deterministic
   corridor.

## Data Model

`DiagnosticReport.topology` is the stable, model-facing topology surface.

Important fields:

- `interfaces[]`: per-interface role, evidence, protected flag, and policy.
- `protectedConstraints[]`: protected boundary classes such as
  `PROTECTED_AUDIO_VLAN` or `PROTECTED_POLICY_NETWORK`.
- `rootCauseCandidates[]`: separate suspected causes such as
  `DEFAULT_ROUTE_OWNERSHIP_STALE`.
- `redHerrings[]`: present but non-authoritative signals such as fake-IP TUN or
  VM bridges.
- `repairCorridor`: `mutationAllowed`, `allowedNext`, and `globalForbidden`.

Raw command text is not a trusted topology model. Raw evidence may help only
when it is bound to the same interface. A detached "Dante" string elsewhere in
a report must not protect or condemn `en0`.

## Live Detection

Default diagnosis now honors topology without requiring `DiagnosticReport.Raw`:

- explicit report topology from fixtures/support bundles;
- user/admin protected-topology manifests:
  - `~/.config/agentlink/protected-topology.json`
  - `~/Library/Application Support/Cactus AgentLink Rescue/protected-topology.json`
  - `/Library/Application Support/Cactus AgentLink Rescue/protected-topology.json`
- MDM/profile evidence as a policy-owned repair boundary;
- weak read-only DNS-SD service discovery as a red-herring signal, not as
  mutation permission.

Example manifest:

```json
{
  "schemaVersion": 1,
  "interfaces": [
    {
      "name": "en0",
      "roles": ["protected_media", "local_control"],
      "roleEvidence": ["Dante", "AES67", "MTRX", "Cisco VLAN2"],
      "protected": true,
      "protectionPolicy": {
        "forbiddenActions": [
          "dhcp_renew",
          "set_dhcp",
          "set_dns",
          "route_flush",
          "ifconfig_down",
          "proxy_reset",
          "switch_vlan_mutation"
        ],
        "allowedActions": ["snapshot", "scoped_probe", "incident_report"]
      }
    }
  ]
}
```

## Repair Corridor

When any protected constraint is active, broad mutation is refused by default.
Allowed next steps are read-only:

- route snapshot;
- network snapshot;
- support bundle;
- incident report / terminal ticket.

Forbidden actions include:

- DHCP renew or `setdhcp` on protected surfaces;
- DNS/proxy reset on protected services;
- route flush;
- TUN cleanup before route ownership proof;
- clean-baseline reset;
- interface down;
- switch/VLAN mutation.

This is intentionally conservative. A future scoped repair can be added only
after it proves the target action touches a non-protected service/interface and
survives the protected-topology fixture gate.

## Fixture Gate

`docs/testing/AGENTLINK_SANDBOX_MAZES.md` now includes m13-m25 protected
topology cases:

- m13/m14: Dante and AES67 audio route traps;
- m15-m23: video, camera, lighting, storage, direct-link, lab, bridge, MDM, and
  USB-management protected surfaces;
- m24: control case where TUN repair is still allowed with no protected
  topology;
- m25: false-positive guard for unbound raw audio text.

Run:

```bash
bash scripts/run_sandbox_mazes.sh
```

The protected topology product claim requires all maze fixtures to pass and the
graph for m13-without-Raw to stay on the protected repair corridor.
