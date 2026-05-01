# Recipe Authoring

Recipes are bounded JSON documents. They declare preconditions, deterministic patch types, verifier refs, risk, and rollback. v0.2 rejects raw shell, arbitrary deletion, recursive ownership/mode changes, curl-pipe-shell, and overwrite-without-backup patterns.

Writable recipes must be reversible and must define rollback. The deterministic runner owns execution; planner output can only select a registered recipe and parameters.

Codex provider recipes must include `name`, `base_url`, and `env_key` fields. They must never write literal API keys. If a Codex provider recipe emits `wire_api`, the only allowed value is `responses`.

Codex provider templates must not emit legacy API-key-env TOML fields. Static provider recipes must report that an online smoke test is still required unless an explicit network-action recipe has been run by the user.
