# Recipe Authoring

Recipes are bounded JSON documents. They declare preconditions, deterministic patch types, verifier refs, risk, and rollback. v0.2 rejects raw shell, arbitrary deletion, recursive ownership/mode changes, curl-pipe-shell, and overwrite-without-backup patterns.

Writable recipes must be reversible and must define rollback. The deterministic runner owns execution; planner output can only select a registered recipe and parameters.

Codex provider recipes must write API key references with `env_key = "NAME"`. They must never write literal API keys. Static provider recipes should report that an online smoke test is still required unless an explicit network-action recipe has been run by the user.
