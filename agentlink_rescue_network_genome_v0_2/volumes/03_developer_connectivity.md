# Volume 03 — Developer Connectivity Failure Space

Developer tools are not equivalent to browser reachability. A Mac can browse the web while Git, npm, Homebrew, Docker, package managers, and API clients fail.

## Tool-specific surfaces

### curl

curl is the baseline because it exposes direct/proxy/TLS/DNS differences quickly:

```bash
curl -vI https://example.com
curl -vI --noproxy '*' https://example.com
curl -vI --proxy http://127.0.0.1:7890 https://example.com
```

### Git

Git can read proxy config from global/system/local config, environment variables, remote-specific settings, and credential helpers. A stale global proxy is a common failure.

### npm

npm has its own config chain. `proxy`, `https-proxy`, `registry`, and certificate options can diverge from shell/browser state.

### Homebrew

Homebrew commonly relies on proxy-related environment variables. Failure may come from GitHub reachability, bottle download reachability, DNS, proxy, or TLS.

### Docker

Docker has several planes:

- Docker Desktop proxy mode.
- Docker CLI config in `~/.docker/config.json`.
- Build-time proxy args.
- Container runtime environment.
- Host reachability from container.

## Common developer failure signatures

### Browser works, `curl` fails

Often system proxy vs env proxy mismatch, TLS trust, or DNS difference.

### `curl` works, Git fails

Often Git-specific proxy config, remote URL protocol, credentials, corporate TLS interception, or SSH config.

### Git works, npm fails

Often npm registry, npm proxy, npm CA, or scoped registry config.

### Host works, Docker container fails

Often container DNS, missing proxy env, wrong localhost assumption, Docker Desktop proxy mode mismatch, or corporate CA not injected into container.

### API works in browser, local agent fails

Often env proxy mismatch, API base URL override, DNS, TLS trust, IPv6 failure, or local runtime sandbox not inheriting env.

## Diagnostic principle

Every developer connectivity report should classify the failure into:

```text
DNS
TCP reachability
proxy reachability
TLS trust
HTTP status/auth
tool-specific config
provider outage
```
