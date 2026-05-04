type AgentLinkTool = {
  name: string
  command: string[]
  description: string
  mutatesSystem: false
}

const safeTools: AgentLinkTool[] = [
  { name: "agentlink_doctor", command: ["agentlink", "doctor", "--json"], description: "Read local AgentLink diagnostic facts.", mutatesSystem: false },
  { name: "agentlink_readiness", command: ["agentlink", "readiness", "doctor", "--json"], description: "Read offline readiness state.", mutatesSystem: false },
  { name: "agentlink_support_bundle", command: ["agentlink", "support", "bundle", "--json"], description: "Create a redacted support bundle.", mutatesSystem: false },
  { name: "agentlink_guided_rescue_dryrun", command: ["agentlink", "orchestrator", "rescue", "--target", "auto", "--dry-run", "--json"], description: "Run orchestrator planning without mutation.", mutatesSystem: false },
  { name: "agentlink_recipe_list", command: ["agentlink", "recipe", "list", "--json"], description: "List approved rescue recipes.", mutatesSystem: false },
  { name: "agentlink_verify_network", command: ["agentlink", "verify", "network", "--json"], description: "Verify current network state.", mutatesSystem: false },
  { name: "agentlink_tun_diagnose", command: ["agentlink", "diagnose", "tun", "--json"], description: "Diagnose Clash/Mihomo TUN residue.", mutatesSystem: false }
]

const blockedSystemRescueTokens = [
  "sudo",
  "networksetup",
  "route",
  "ifconfig",
  "launchctl",
  "killall",
  "pkill"
]

export function agentlinkTools() {
  return safeTools
}

export function reviewCommand(command: string) {
  const lower = command.toLowerCase()
  const blocked = blockedSystemRescueTokens.filter((token) => lower.includes(token))
  if (blocked.length === 0) {
    return { allowed: true, warning: "" }
  }
  return {
    allowed: false,
    warning: "Use AgentLink Guided Rescue or a Terminal Repair Ticket. OpenCode is not the system rescue executor."
  }
}

export default {
  name: "agentlink-rescue",
  tools: safeTools,
  reviewCommand
}
