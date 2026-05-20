# AgentLink Rescue v0.2 Verifier Library

## Core network

- `V-GATEWAY`: route to default gateway and TCP/ICMP reachability where allowed.
- `V-DIRECT-IP`: direct IP HTTPS/TCP probe to known endpoint.
- `V-DNS-PUBLIC`: affected hostname resolves through expected resolver.
- `V-DNS-SCOPED`: scoped resolver routes expected domain to expected DNS plane.
- `V-PROXY-SYSTEM`: `scutil --proxy` equals intended system proxy mode.
- `V-PROXY-ENV`: shell env proxy variables equal intended mode and are not contradictory.
- `V-VPN-ROUTE`: target route uses intended interface before/after tunnel toggle.
- `V-TLS`: TLS issuer/hostname/time validation succeeds without insecure flags.
- `V-TOOL`: affected tool completes its own health/verbose probe.

## Developer tools

- `V-GIT`: `git ls-remote` or clone probe reaches expected remote.
- `V-NPM`: registry metadata fetch succeeds with expected registry/proxy.
- `V-BREW`: Homebrew API and bottle endpoints both pass.
- `V-DOCKER`: host, daemon, container, and build contexts reach expected proxy/direct path.
- `V-CODEX`: Codex host shell, project config, and sandbox network probe agree.

## AoIP/manual-only

- `V-DANTE-DISCOVERY`: expected devices visible on intended NIC.
- `V-DANTE-CLOCK`: clock leader and lock state stable.
- `V-AES67-PTP`: PTPv2 domain/profile lock confirmed.
- `V-AES67-SDP`: SDP parameters match receiver expectation.
- `V-RTP-MEDIA`: RTP/media counters stable and audio passes.
