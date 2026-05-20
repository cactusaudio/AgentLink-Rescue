# Volume 02 — Proxy / VPN / Clash Verge Failure Space

## The proxy alignment problem

On macOS, browser, Terminal, Git, npm, Docker, Homebrew, and local AI tools may all observe different proxy states. AgentLink Rescue should treat proxy as a multi-plane object:

```text
system proxy
PAC / auto proxy
shell env proxy
launchctl env proxy
Git proxy
npm proxy
Docker Desktop proxy
Docker CLI/container proxy
app config proxy
Clash/mihomo local listener
TUN route/DNS state
```

## Proxy state grammar

A proxy state is healthy only if all of the following are known:

- Is a proxy expected?
- Which plane owns it?
- Which host/port is configured?
- Is the listener alive?
- Does the target traffic need proxy or direct?
- Are LAN/localhost excluded?
- Does the app inherit environment variables?
- Is DNS supposed to resolve before or inside proxy?

## Clash Verge / TUN special cases

Clash-like clients introduce at least three modes:

1. System proxy mode: apps that respect macOS proxy settings route through local listener.
2. Environment proxy mode: CLI tools route through variables such as `http_proxy`, `https_proxy`, `all_proxy`.
3. TUN mode: routing and DNS can be modified below the application level.

Failure patterns:

- System proxy points to a dead local port after Clash exits.
- TUN creates route capture but DNS is not correctly handled.
- Fake-IP/DNS mode resolves domains into synthetic IPs that confuse direct LAN logic.
- LAN bypass is missing, so local services disappear.
- VPN and Clash both attempt to own routing/DNS.

## VPN + proxy second-order failures

The hardest failures are not VPN or proxy alone but inconsistent composition:

- VPN gives internal DNS; proxy sends DNS outside the tunnel.
- TUN captures traffic; system proxy still points to local port.
- `NO_PROXY` excludes hostname but not IP range.
- Git/npm/brew inherit stale env vars while browser uses PAC.
- Docker containers cannot reach host proxy because `127.0.0.1` inside container means the container itself, not the Mac host.

## Required discriminators

```bash
scutil --proxy
env | grep -i proxy
launchctl getenv HTTP_PROXY
launchctl getenv HTTPS_PROXY
lsof -nP -iTCP -sTCP:LISTEN | egrep '7890|7897|7899|9090|clash|mihomo|verge'
netstat -rn | grep -E 'default|utun|tun|tap'
ifconfig | grep -E '^utun|^en|status:|inet '
```

App-specific:

```bash
git config --global --get http.proxy
git config --global --get https.proxy
npm config get proxy
npm config get https-proxy
cat ~/.docker/config.json 2>/dev/null
```

## Runtime stance

The UI should not ask "turn proxy off?" directly. It should say:

```text
Your system proxy points to 127.0.0.1:7890, but no process is listening there.
This explains why Safari/CLI/app traffic may hang.
Recommended repair: disable stale system HTTP/HTTPS/SOCKS proxy for active services.
Rollback: restore previous proxy snapshot.
```
