# Support Bundle

Support Bundle exports redacted local rescue evidence for Codex or a human helper when the machine is offline or the agent path is broken.

Included:

- facts
- Offline Readiness report
- Brain doctor
- Installer doctor
- Dev Essentials doctor
- latest session report when available
- latest Codex dispatch when available

Excluded:

- private keys
- Wi-Fi passwords
- browser cookies
- shell history
- unredacted API keys
- repository indexes or project source bundles

CLI:

```bash
agentlink support bundle --json
agentlink support bundle --output /path/to/bundle.zip --json
```

