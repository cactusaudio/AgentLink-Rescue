# Read This First - GPT Pro Stress Package

This package is for GPT Pro cloud fixture/read-only/dry-run stress testing of Cactus AgentLink Rescue.

Run this from the extracted package root:

```bash
cd AgentLink-Rescue
FULL=1 bash scripts/gpt_pro_stress_test.sh
```

The result directory is:

```text
AgentLink-Rescue/gpt-pro-results/<timestamp>/
```

The key result file is:

```text
SUMMARY.json
```

## Hard Safety Rules

Do not run:

```text
sudo
agentlink rescue --yes
agentlink guided rescue --yes
real networksetup mutation
real route mutation
real ifconfig mutation
real launchctl mutation
rm -rf outside a temporary sandbox
```

Allowed GPT Pro cloud surfaces are fixture, read-only, and dry-run only:

```text
agentlink version
agentlink selftest
agentlink doctor
agentlink manifest
agentlink chaos list
agentlink chaos run --fixture ...
agentlink diagnose --json
agentlink diagnose-graph --from ...
agentlink recipe list
agentlink recipe inspect ...
agentlink recipe run ... --dry-run
agentlink support bundle
agentlink package doctor
agentlink package repair --dry-run
agentlink journal list/inspect/recover without --yes
agentlink readiness
agentlink dev doctor
agentlink rollback --dry-run
agentlink last-good list/inspect
agentlink brain doctor
```

On non-macOS cloud hosts, mutating commands should return `unsupported platform`. That is expected for real rescue/mutation surfaces and a failure only if it happens on fixture/read-only/dry-run surfaces.

## What A Pass Means

A passing GPT Pro cloud run proves:

- package integrity,
- source buildability in cloud,
- fixture/read-only/dry-run CLI behavior,
- sandbox maze behavior,
- deterministic chaos/adversarial gates,
- no model weights or runtime binaries in the package.

It does not prove:

- real macOS network mutation,
- macOS GUI behavior,
- signed/notarized distribution,
- installer DMG behavior,
- Gemma/Qwen/model behavior.

Those are separate macOS/model release surfaces.

## Full Guide

Use the detailed guide here:

```text
docs/testing/GPT_PRO_PRERELEASE_STRESS_GUIDE.md
```

