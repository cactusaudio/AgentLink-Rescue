# AgentLink Rescue v0.5.1 Phase Review Closure

## Verdict

v0.5.1 phase review is accepted for controlled read-only CLI dogfood.

It is not a public release. Release readiness remains blocked on clean release
packaging, GUI proof, signing/notarization, and real macOS mutation/rollback
proof.

## Completed Items

1. Accepted v0.5.1 phase closure after GPT Pro confirmed the FTS/provenance
   failure is fixed.
2. Added explicit v0.5.1 audit/package entrypoints while retaining v0.5.0
   compatibility aliases:
   - `scripts/gpt_pro_v0501_audit.sh`
   - `scripts/package_gpt_pro_v0501_audit.sh`
   - `README_FIRST_FOR_GPT_PRO_V0501_AUDIT.md`
3. Ran real Mac read-only CLI dogfood with strict privacy.
4. Ran release-hardening checks that are safe without signing credentials or
   real network mutation:
   - FTS/provenance audit
   - strict privacy collector
   - gate/report generation
   - recipe dry-run
   - rollback/journal/repair unit coverage
   - core GUI package build and selftest
   - code-signing identity inspection

## GPT Pro Re-Audit Result

Package SHA256:

```text
101022d417583fd1a7176ec20e937ce0a8e12ee953a4101ecc7e2a2f1fd962e9
```

GPT Pro verdict:

```text
phase audit: PASS
FTS/provenance fix: PASS
full regression: PASS
dogfood readiness: PASS, scoped to controlled read-only CLI dogfood
release readiness: FAIL
```

Important confirmed facts:

```text
cards=300
layers=15
ftsRows=300
ftsIdsUsable=true
ftsNullIds=0
cards_fts returns non-null id/title values
FTS whyMatched provenance is correct
Docker auth leak fixed
strict privacy fixed for audited surfaces
NO_PROXY false positive fixed
symptom_routes.yaml runtime use proven
schema-derived validation proven for included schema shape
```

## Real Mac Read-Only Dogfood

Dogfood directory:

```text
/tmp/agentlink-v051-macos-readonly-dogfood
```

Commands run through a locally built `agentlink-rescue` binary:

```text
agentlink-rescue index rebuild --json
agentlink-rescue index stats --json
agentlink-rescue collect --privacy strict --json
agentlink-rescue features --snapshot ... --json
agentlink-rescue diagnose --symptom "browser works but codex fails" --snapshot ... --top 5 --json
agentlink-rescue gate --card DANTE-CLOCK-001 --json
agentlink-rescue gate --card MAC-PROXY-002 --json
agentlink-rescue report --format markdown ...
agentlink-rescue report --format json ...
```

Observed summary:

```json
{
  "version": "agentlink 0.5.1",
  "ftsRows": 300,
  "ftsIdsUsable": true,
  "ftsNullIds": 0,
  "collectPrivacy": "strict",
  "featuresPrivacy": "strict",
  "featureKeys": 29,
  "topHypotheses": [
    ["CODEX-API-001", 84],
    ["DEV-CURL-001", 80],
    ["MAC-PROXY-002", 71],
    ["DEV-TLS-001", 65],
    ["DEV-CODEX-002", 62]
  ],
  "danteGate": "manual_only",
  "proxyGate": "human_confirmed_allowed",
  "reportMarkdownExists": true,
  "reportJSONExists": true
}
```

Collector warnings were expected and non-fatal:

```text
raw directory is redacted by default
git_proxy.txt: exit status 1
```

Secret-pattern scan across the dogfood output produced zero matching file
paths.

## Dry-Run / Rollback Gate

Safe dry-runs:

```text
recipe run api-key-detection-redaction --dry-run --json
recipe run macos-zsh-path-repair --dry-run --json
```

Both returned `status=dry-run` and `changedFiles=null`.

`macos-clash-tun-force-repair --dry-run` was attempted and refused before
planning with:

```text
precondition failed: command_exists
```

That is a safe fail-closed result on this host, not a repair proof.

Unit coverage:

```text
go test ./internal/repair ./internal/journal ./internal/rollback ./internal/recipe ./internal/orchestrator -count=1
```

Result:

```text
PASS
```

## Release Hardening Status

Completed:

```text
v0.5.1 audit entrypoints
FTS/provenance assertions
strict privacy audit proof
controlled read-only Mac dogfood
dry-run/report/gate proof
rollback/journal/repair unit proof
core GUI package build
core GUI package selftest
GPT Pro re-audit package with clean checksum
```

Core GUI package proof:

```text
scripts/package_gui_core.sh
dist/Cactus-AgentLink-Rescue-v0.5.1-core-gui.zip
sha256 b02b97869794fc4275ee651bd479b4b24d4c4a15182398e5388aa6e6a6b3c803
unzip -tq PASS
CactusAgentLinkRescue --selftest-gui ok=true
doctorExitCode=0
guidedExitCode=0
brainDoctorExitCode=0
packageDoctorExitCode=20
no GGUF model files in core GUI package
no llama.cpp runtime binaries in core GUI package
```

Signing check:

```text
security find-identity -v -p codesigning
0 valid identities found
```

Still not release-ready:

```text
0 valid local code-signing identities found
no notarization was attempted
no real macOS mutating repair/rollback was executed
SQLite/FTS still uses Python sqlite3 subprocesses; correct but not final-performance architecture
scenarioBoost() still contains ranking-layer hand tuning
```

## Safety Boundary

Not run:

```text
sudo
agentlink rescue --yes
agentlink guided rescue --yes
networksetup mutation
route mutation
ifconfig mutation
launchctl mutation
pf mutation
VPN mutation
Keychain extraction
Wi-Fi password extraction
```

## Recommendation

Tag this state as `v0.5.1-phase-review-alpha`.

Next release-candidate sprint should focus only on:

```text
native or long-lived SQLite/FTS access
GUI proof
clean release package from clean committed source
code signing and notarization with explicit credential authority
real macOS dry-run plus rollback proof on disposable fixtures
removing or justifying scenarioBoost ranking hand tuning
```
