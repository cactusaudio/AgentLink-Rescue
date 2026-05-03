# Installer Center

Installer Center restores missing local agent tools from fixed official sources. It is part of AgentLink Rescue, but it does not replace the recipe runner or Guided Rescue.

## Rules

- Dry-run before install.
- `install` requires `--yes`.
- No curl-pipe-shell.
- No random mirrors.
- No secret writes.
- No silent sudo.
- No password collection.
- GUI install buttons call the package-local `bin/agentlink`.
- Dependency availability is evaluated per method. If npm is missing but Homebrew is available, Codex CLI and Gemini CLI can still be available through the Homebrew method. If Homebrew is missing but npm is available, npm-backed methods can still be available.

## Supported Installers

- `codex-cli`: `npm install -g @openai/codex` or `brew install codex`
- `codex-app`: opens the official OpenAI Codex App page
- `claude-code-cli`: `npm install -g @anthropic-ai/claude-code`
- `gemini-cli`: `npm install -g @google/gemini-cli` or `brew install gemini-cli`
- `clash-verge-rev`: opens a cached official GitHub release DMG when present

## CLI

```bash
./bin/agentlink installer list
./bin/agentlink installer doctor
./bin/agentlink installer dry-run codex-cli
./bin/agentlink installer install codex-cli --yes
./bin/agentlink installer verify codex-cli
./bin/agentlink installer open clash-verge-rev
```

Installer Center verifies after install where possible. If verification cannot prove success, the status is `manual_action_required` or `failed`.

Installer status semantics:

- `installed`: verifier already found the tool/app.
- `available`: at least one official install method is available.
- `missing_dependency`: no supported method is currently available.
- `manual_action_required`: a vendor app/download flow requires user action.
