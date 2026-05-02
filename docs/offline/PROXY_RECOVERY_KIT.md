# Proxy Recovery Kit

Proxy Recovery Kit is the optional cached installer surface for Clash Verge Rev.

AgentLink can open the cached DMG so a user can reinstall a proxy client while offline. AgentLink does not configure Clash Verge Rev, import a subscription, enable system proxy, enable TUN, or launch the app automatically.

## Fetch

```bash
./scripts/fetch_clash_verge_rev.sh
```

The script uses the official upstream GitHub releases API:

```text
https://github.com/clash-verge-rev/clash-verge-rev/releases
```

Downloaded DMGs are ignored by git and can be included in the optional Brain GUI ProxyKit zip.

## Safety

- Official GitHub release assets only.
- Architecture-specific DMG selection.
- SHA256 is computed and recorded in `manifest.lock.json`.
- No silent install.
- No proxy/TUN auto-enable.
- No deletion of existing proxy configuration.
