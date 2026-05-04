# Proxy App Clean Reinstall

Clean reinstall is a final resort only.

Use it only if:

- targeted TUN repair failed,
- restart-gate verification failed, or
- a human explicitly chooses final resort.

Clean reinstall must quarantine app data/helpers/config residue with rollback metadata. It must not permanently delete by default, auto-launch proxy apps, enable TUN, enable system proxy, or import proxy profiles.

