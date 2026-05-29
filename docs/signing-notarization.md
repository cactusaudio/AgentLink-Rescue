# Signing And Notarization

AgentLink Rescue does not sign or notarize packages automatically.

Release packages are intentionally built as local zip artifacts. Signing and
notarization are a distribution step that must run after deterministic package
creation and must not change the CLI, recipe runner, snapshot, rollback,
diagnosis, or model-advisory behavior.

## Current v0.5.2 Local Boundary

The local Mac Studio used for v0.5.2 hardening has no usable Developer ID code
signing identity:

```text
security find-identity -v -p codesigning
     0 valid identities found
```

That means notarization is a credentials/Apple-account boundary, not a product
runtime proof. The current portable field-beta proof is ad-hoc signed local zip
integrity plus GUI selftest, not public distribution readiness.

The latest portable beta on `CTS Dark` was ad-hoc signed with `codesign -s -`
and verified with `codesign --verify --deep --strict`. Ad-hoc signing improves
local bundle consistency but is not Developer ID signing and is not
notarization.

Do not attempt notarization from the rescue kernel unless the operator
explicitly supplies Developer ID credentials and approves that release step.

Current local workaround for quarantine after copying or downloading:

```sh
xattr -cr "/path/to/Cactus AgentLink Rescue.app"
xattr -cr "/path/to/Cactus-AgentLink-Rescue"
```
