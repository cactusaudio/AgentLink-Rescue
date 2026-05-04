# AgentLink OpenCode Plugin

This plugin exposes read-only AgentLink tools to OpenCode for local report analysis and coding assistance.

It does not expose direct rescue execution, sudo commands, quarantine actions, or privileged network mutation. If OpenCode suggests system rescue commands such as `sudo`, `networksetup`, `route`, `ifconfig`, `launchctl`, or `killall`, the plugin policy is to redirect the user to AgentLink Guided Rescue or a Terminal Repair Ticket.

Safe tools:

- `agentlink_doctor`
- `agentlink_readiness`
- `agentlink_support_bundle`
- `agentlink_guided_rescue_dryrun`
- `agentlink_recipe_list`
- `agentlink_verify_network`
- `agentlink_tun_diagnose`
