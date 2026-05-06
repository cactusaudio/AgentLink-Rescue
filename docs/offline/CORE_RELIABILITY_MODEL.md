# Core Reliability Model

v0.5.1 hardens AgentLink Rescue Core around failure containment. The goal is not more rescue surface; it is making every existing rescue path observable, reversible, bounded, and safe to retry on unfamiliar Macs.

## Invariants

1. Doctor is always read-only.
2. Dry-run is always no-mutation.
3. Every writable repair is a transaction.
4. Every transaction has preflight, checkpoint, mutation log, verifier, rollback path, and final report.
5. If postflight worsens, AgentLink rolls back automatically.
6. If the process crashes mid-transaction, the next launch can detect the incomplete journal.
7. `rollback --id` restores durable state, not only in-process memory.
8. GUI commands must not block indefinitely.
9. CLI command execution has timeouts, output caps, and redaction.
10. Optional Brain, installer, and proxy-kit assets can be missing without breaking Core diagnostics.
11. Core packages work without model/runtime/installer assets.
12. Field packages must work from Downloads and paths with spaces after `RUN-FIRST.command`.
13. Command success is not repair success; verifier success is repair success.
14. Failures should produce a support bundle or an actionable report.
15. Repeated safe repair and dry-run paths should be idempotent.

## Transaction Journal

Mutating operations write a durable journal under:

```text
~/Library/Application Support/Cactus AgentLink Rescue/journal/
```

Each transaction directory contains:

```text
transaction.json
preflight.json
planned-actions.json
mutation-log.jsonl
verifier.json
rollback.json
status.json
```

`agentlink journal recover --json` is read-only and reports incomplete transactions. `agentlink journal recover --yes --json` can mark no-mutation transactions abandoned and create rollback terminal tickets for transactions that mutated state.

## Package Self-Healing

`agentlink package doctor --json` checks package structure, executable bits, quarantine xattrs, app support directories, temp directory availability, and optional assets. It is read-only.

`agentlink package repair --yes --json` performs package-local fixes only:

- `xattr -cr` on the package root
- `chmod +x` for embedded binaries and scripts
- creation of AgentLink app-support directories

It never downloads assets and never runs sudo from the GUI.

## Safe Mode

Safe Mode is for degraded packages:

```bash
agentlink --safe-mode doctor
agentlink --safe-mode support bundle
agentlink --safe-mode package doctor
agentlink --safe-mode journal recover
```

Safe Mode does not run model, installer, OpenCode, or system repair execution. It preserves read-only diagnostics, package health checks, journal recovery checks, support bundle export, and copyable terminal commands.
