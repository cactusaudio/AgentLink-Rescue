# Report Format

Human reports describe detected state, selected recipe, changed files, verifier results, rollback command, and next user action.

Agent dispatch reports are concise JSON documents for Codex or Claude. They include redacted facts, commands already run, files changed, verifier failures, and the exact next task. They must not contain secrets.

Recipe final states use bounded vocabulary: `success`, `success_with_warnings`, `config_template_generated`, `needs_user_secret`, `needs_online_smoke_test`, `verifier_failed`, and `rolled_back`.

Codex DeepSeek provider config reports `needs_user_secret` when the configured `env_key` is missing and `needs_online_smoke_test` when static config checks pass. Static checks prove the template shape only; they do not prove Codex can use DeepSeek online. Reports must say `provider_smoke_test_required` until an explicit online smoke recipe passes.
