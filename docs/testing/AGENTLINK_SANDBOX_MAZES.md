# AgentLink Sandbox Mazes

This pack models common macOS network-rescue situations without mutating the host.
Each maze is a fixture-driven `DiagnosticReport` consumed by the real AgentLink classifier and TUN detector through `agentlink chaos run`.

## Rules

- No `sudo`, `networksetup`, `route`, `ifconfig`, `launchctl`, or real host repair is executed.
- Fixtures must not hand-set classifications. `classify.Apply` owns the classes.
- Each maze should assert both the primary failure class and the high-level repair tier when possible.
- TUN fixtures may additionally assert `DiagnoseTun(...).RecommendedRepair == "tun"`.
- If a scenario is policy-owned, MDM-owned, or endpoint-blocked, the expected result must not imply local system mutation.

## Run

```bash
bash scripts/run_sandbox_mazes.sh
```

Default report:

```text
/tmp/agentlink-sandbox-mazes-report.json
```

To write a persistent report:

```bash
REPORT_PATH=/tmp/agentlink-sandbox-mazes-$(date +%Y%m%dT%H%M%S).json \
  bash scripts/run_sandbox_mazes.sh
```

## Maze Catalog

| ID | Situation | Expected class | Expected tier |
| --- | --- | --- | --- |
| `m01-healthy-wifi-baseline` | Healthy Wi-Fi and agent endpoint | `OK` | `none` |
| `m02-system-proxy-dirty-internet-ok` | macOS system proxy polluted, probes still work | `SYSTEM_PROXY_DIRTY` | `safe` |
| `m03-shell-env-proxy-dirty` | shell `HTTPS_PROXY` points at stale local proxy | `USER_PROXY_DIRTY` | `safe` |
| `m04-dev-tool-proxies-dirty` | Git/npm/Homebrew proxy configs polluted | `GIT_PROXY_DIRTY` | `safe` |
| `m05-https-only-captive-portal` | raw IP and DNS work, HTTPS fails | `HTTPS_FAIL` | `safe` |
| `m06-no-default-route-wifi-active` | Wi-Fi has IP but no default route | `NO_DEFAULT_ROUTE` | `standard` |
| `m07-dhcp-link-local-only` | DHCP failed, only 169.254 link-local | `LINK_LOCAL_ONLY` | `standard` |
| `m08-gateway-unreachable-lan` | route exists but gateway/raw IP unreachable | `GATEWAY_UNREACHABLE` | `safe` |
| `m09-agent-endpoint-blocked-only` | general internet OK, AI/tool endpoint blocked | `GENERAL_INTERNET_OK_AGENT_ENDPOINT_BLOCKED` | `none` |
| `m10-clash-tun-airdrop-degraded` | stale Clash/Mihomo TUN plus AWDL degraded | `CLASH_TUN_ACTIVE_OR_STALE` | `tun` |
| `m11-network-extension-stale` | detect-only Network Extension present, internet broken | `NETWORK_EXTENSION_SUSPECTED` | `safe` |
| `m12-mdm-profile-suspected` | managed profile controls network policy | `MDM_PROFILE_SUSPECTED` | `safe` |
| `m13-dante-vlan-no-internet-trap` | default route still points at protected Dante/MTRX audio VLAN while alternate management path exists | `PROTECTED_AUDIO_VLAN_ROUTE_TRAP` | `safe` |
| `m14-aes67-audio-vlan-route-trap` | AES67/RAVENNA protected audio VLAN, no Dante token required | `PROTECTED_AUDIO_VLAN_ROUTE_TRAP` | `safe` |
| `m15-ndi-video-vlan-stale-default` | NDI/video production VLAN is local-only and protected | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m16-atem-camera-control-static-lan` | ATEM/PTZ camera-control LAN uses static addressing | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m17-artnet-lighting-linklocal` | Art-Net/sACN lighting link-local network is intentional | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m18-iscsi-nas-direct-no-internet` | NAS/iSCSI direct storage interface must not be reset | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m19-thunderbolt-bridge-direct-mac` | Thunderbolt Bridge direct Mac link is not internet | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m20-lab-instrument-static-subnet` | lab/PLC static control subnet is protected | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m21-vm-bridge-active-false-internet` | VM bridge active interface is a red herring | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m22-mdm-scoped-dns-proxy` | MDM/corporate scoped DNS/proxy policy blocks local reset | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m23-usb-management-wifi-internet` | USB management adapter protected; Wi-Fi is internet candidate | `PROTECTED_TOPOLOGY_CONSTRAINT` | `safe` |
| `m24-tun-owned-no-protected-topology` | TUN owns the broken route and no protected topology exists | `CLASH_TUN_ACTIVE_OR_STALE` | `tun` |
| `m25-unbound-raw-audio-false-positive` | unrelated raw Dante text must not protect the default route | `DNS_FAIL` | `safe` |

## Expansion Backlog

- Add Bluetooth PAN, airport-roaming, campus 802.1X, same-subnet multi-home ambiguity, and captured portal edge cases.
- Add sibling fixtures where the visible symptom is the same but the correct repair corridor differs.
- Feed the same fixtures into Gemma/Qwen advisory surfaces after deterministic kernel results are frozen.
