# Signing And Notarization

v0.4.0 does not sign or notarize packages automatically.

Release packages are intentionally built as local zip artifacts. If a future signed distribution is needed, add a separate signing pipeline that runs after package creation and does not change the deterministic CLI, recipe runner, snapshot, rollback, or Brain planner behavior.

Current local workaround for quarantine after copying or downloading:

```sh
xattr -cr "/path/to/Cactus AgentLink Rescue.app"
xattr -cr "/path/to/Cactus-AgentLink-Rescue"
```
