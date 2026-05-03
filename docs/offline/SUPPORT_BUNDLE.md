# Support Bundle

Support Bundle exports redacted local rescue evidence for Codex or a human helper when the machine is offline or the agent path is broken.

Disclosure shown by the GUI before export:

> This support bundle contains redacted local diagnostics, including tool presence, local paths, network/proxy status, readiness reports, and latest session metadata. It does not include private keys, browser cookies, shell history, Wi-Fi passwords, or full API keys.

Included:

- facts
- support-bundle-manifest.json with included categories and exclusions
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

Support bundles are redacted, but they may still reveal local paths, installed tool presence, network/proxy state, and session metadata. Review before sharing outside the trusted repair flow.

CLI:

```bash
agentlink support bundle --json
agentlink support bundle --output /path/to/bundle.zip --json
```
