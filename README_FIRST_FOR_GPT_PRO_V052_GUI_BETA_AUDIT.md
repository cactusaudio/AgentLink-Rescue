# GPT Pro Audit Entry: AgentLink Rescue v0.5.2 GUI Field Beta

This package audits the AgentLink Rescue v0.5.2 GUI field beta line.

It is an engineering audit package, not a public release package. It does not
prove signing, notarization, installer DMG behavior, model-runtime behavior, or
real mutating macOS network repair.

Read first:

```text
PACKAGE_MANIFEST.json
docs/testing/GPT_PRO_V052_GUI_FIELD_BETA_AUDIT_PROMPT.md
governor/reports/V0521_GUI_FIELD_BETA_READINESS.md
docs/offline/GUI_DOGFOOD.md
```

Main command:

```bash
FULL=1 OUT="$PWD/gpt-pro-results/v052-gui-field-beta" bash scripts/gpt_pro_v052_gui_beta_audit.sh
```

If the cloud tool cannot finish the long full regression in one foreground
invocation, run the same command in the background and poll logs, or run:

```bash
FULL=0 OUT="$PWD/gpt-pro-results/v052-gui-field-beta-fast" bash scripts/gpt_pro_v052_gui_beta_audit.sh
```

Do not call `FULL=0` a full regression pass. It is only a targeted GUI beta
source/runtime audit.

Expected scope:

- Linux/GPT Pro cloud can verify source, package integrity, CLI/runtime phase
  audit, static GUI beta contracts, and Go regression.
- macOS can additionally run `scripts/dogfood_gui_core.sh` and
  `CactusAgentLinkRescue --selftest-gui`.

Hard safety boundary:

```text
Do not run sudo.
Do not run agentlink rescue --yes.
Do not run guided rescue --yes.
Do not mutate networksetup, route, ifconfig, launchctl, pf, VPN, Keychain, or Wi-Fi settings.
Do not install models or runtimes.
Do not add or execute model weights.
```
