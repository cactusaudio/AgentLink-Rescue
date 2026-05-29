# Field Beta Status

Current status: v0.5.2 GUI field beta with post-alpha hardening at commit
`01ced16` (`Harden AgentLink field beta`).

The shared CLI core still reports `agentlink 0.5.1`. Treat v0.5.2 as the
GUI/audit/portable-beta line, not as a public CLI version bump.

## Verified Portable Beta

Latest local portable bundle:

```text
/Volumes/CTS Dark/Cactus-AgentLink-Rescue-v0.5.2-field-beta-hardened-portable-universal-20260524T012027Z/
/Volumes/CTS Dark/Cactus-AgentLink-Rescue-v0.5.2-field-beta-hardened-portable-universal-20260524T012027Z.zip
SHA256 e9347c39c50c42920240cadca55f3a28f49079561822604919f8dda81caa17eb
```

Contents:

- `Cactus AgentLink Rescue.app`
- `Open AgentLink Rescue.command`
- `VERIFY-CHECKSUMS.command`
- `CHECKSUMS.txt`
- `PACKAGE-MANIFEST.json`
- `README-FIRST.txt`

Verified:

- app and embedded CLI are universal Mach-O (`arm64` + `x86_64`);
- ad-hoc codesign verifies locally;
- package checksum and zip integrity pass;
- no model weights, GGUF, llama runtime, DMG installer, `node_modules`, build
  metadata, or Finder metadata are included;
- `--selftest-gui` returns `ready_for_field_beta`;
- embedded guided rescue runs dry-run only;
- app launch from the external USB path was observed and then cleaned up.

## Hardening Delta

The post-alpha hardening pass added:

- private diagnostic/report/session/journal/snapshot files written with `0600`;
- private app-support/report/session directories written with `0700` where
  AgentLink owns the data;
- config-file patching that preserves an existing user file mode, falling back
  to `0600` for new files;
- support-bundle zipping that skips symlinks and non-regular files;
- dead-code cleanup in planner/repair/verifier paths;
- fuzz coverage for redaction, planner decision decode, recipe decode, and
  diagnosis-graph build-from-JSON paths;
- `.gitignore` guard for accidental root-level `agentlink` build artifacts.

## Latest Local Verification

```text
go test ./... -count=1 -timeout 35m
go vet ./...
staticcheck ./...
```

All passed before the hardening commit.

The pressure run reported:

- 64-way concurrent diagnosis and additional mixed CLI concurrency;
- chaos and GUI field-beta audit green;
- race tests green across packages;
- roughly 15.6M fuzz mutations across four harnesses with no panic;
- remaining gosec findings classified as intentional design or static-analysis
  false positives, not untriaged release blockers.

## Boundary

This is ready for controlled read-only CLI dogfood and controlled GUI field
beta on real Macs. It is not a public release until signing/notarization,
broader cross-Mac GUI QA, and real disposable mutation/rollback proof are
complete.
