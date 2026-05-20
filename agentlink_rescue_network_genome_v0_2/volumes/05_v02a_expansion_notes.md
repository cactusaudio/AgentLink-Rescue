# v0.2A Expansion Notes

v0.2A expands the seed from 34 to 300 cards. The intent is not encyclopedic prose; it is build-time exploration compressed into structured failure cards.

## Coverage added

- macOS physical/link, service order, Network Location, DHCP/static IP, route, DNS.
- System proxy, PAC, shell env proxy, NO_PROXY, local proxy listener, auth, PAC cache, proxy chain.
- VPN, Network Extension, utun, kill-switch, split tunnel, Clash/Verge/mihomo TUN, fake-IP, DNS, bypass/rules.
- Developer/agent tools: curl, Git, GitHub CLI, npm/pnpm/Yarn/Bun, pip/uv, Homebrew, Docker/BuildKit/Colima, Ollama, VS Code/Electron, Node/Python/Rust/Go/Java, OpenAI/Codex/Anthropic.
- TLS/trust: corporate CA, certifi, SNI, OCSP/CRL, mTLS, custom CA env.
- Pro audio: mDNS/SAP/NMOS/RAVENNA discovery, Dante visibility, DDM, PTPv1/PTPv2, clock domains, RTP/SDP/channel/sample/packet-time/QoS/IGMP/VLAN/latency.
- External-provider and policy/MDM failure classes.

## Product consequence

The runtime can now route the same vague symptom — “network broken” — into multiple planes without pretending they are the same failure.
