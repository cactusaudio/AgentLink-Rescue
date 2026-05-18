# GPT Pro Cloud Gate Pass - 2026-05-18T07:49:25Z

## Verdict

Package `agentlink-gpt-pro-prerelease-stress-lite-20260518T062655Z.zip` passed the declared GPT Pro cloud fixture/read-only/dry-run stress gate.

This validates the cloud package contract only. It does not validate real macOS mutation, GUI behavior, signing/notarization, installer DMG behavior, or Gemma/Qwen/model behavior.

## Package

```text
SHA256: 0a01ccd9cd20a05e458ffb8e05b01cb5181cb27bf7e3007616ea09e0d8c54b52
PACKAGE_MANIFEST stamp: 20260518T062655Z
sourceCommit: d8d97609be3ad1dc934c429506da44956c940fb1
sourceTreeDirtyAtPackageTime: true
```

GPT Pro command:

```bash
FULL=1 OUT=/mnt/data/agentlink_gpt_pro_new_results_full_20260518T0702Z \
  bash scripts/gpt_pro_stress_test.sh
```

Driver exit:

```text
0
```

## Suite Results

```text
00-version: PASS
01-chaos-list: PASS
02-sandbox-mazes: PASS (12/12)
03-chaos-fixtures: PASS
04-core-units: PASS
05-v0300-v0320-chaos: PASS (83.739s)
06-v0330-rc-chaos: PASS (1292.487s)
07-full-go-test: PASS (internal/chaos 1376.343s)
08-go-vet: PASS
```

The previous `unsupported platform: agentlink supports macOS only` blocker did not recur on fixture/read-only/dry-run surfaces.

## Safety

GPT Pro reported:

```text
sudo executed? no
agentlink rescue --yes executed? no
agentlink guided rescue --yes executed? no
real networksetup mutation executed? no
real route mutation executed? no
real ifconfig mutation executed? no
real launchctl mutation executed? no
rm -rf outside temp sandbox? no
model weights present? no
live secrets found? no
residual processes? no
18080/18081 listeners? no
```

## Follow-Up Cleanup

- Added root `README_FIRST_FOR_GPT_PRO.md` for future packages.
- Confirmed repo copy of `governor/reports/GPT_PRO_CLOUD_GATE_REPAIR_20260518.md` points at the final `20260518T062655Z` package and `0a01...` SHA. The stale metadata existed in the already-uploaded zip because the report was updated after that zip was generated.

