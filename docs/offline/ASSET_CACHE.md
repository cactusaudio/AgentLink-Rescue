# Asset Cache

Cactus AgentLink Rescue source checkouts intentionally keep large binary
payloads out of git:

- Gemma GGUF models
- llama.cpp runtime folders
- Clash Verge Rev DMG installers
- `dist/` release zips
- `bin/` and GUI build output

Default cache:

```text
~/CactusLocalAgent/.asset-cache/AgentLink-Rescue
```

Override:

```sh
export AGENTLINK_ASSET_CACHE="/path/to/AgentLink-Rescue-asset-cache"
```

Expected layout:

```text
models/gemma-4-E4B-it-Q4_K_M.gguf
runtimes/llama.cpp/arm64/llama-cli
runtimes/llama.cpp/arm64/*.dylib
installers/clash-verge-rev/macos-arm64/*.dmg
releases/v0.5.1/*.zip
releases/v0.5.2/*.zip
ASSET_MANIFEST_SHA256.txt
```

Package scripts search in this order:

1. Repo-local ignored asset paths, useful after `scripts/fetch_brain_assets.sh`.
2. `AGENTLINK_ASSET_CACHE`.
3. `~/CactusLocalAgent/.asset-cache/AgentLink-Rescue`.

The cache is not a source of truth for product behavior. It only stores
reproducible binary inputs and archived release artifacts.
