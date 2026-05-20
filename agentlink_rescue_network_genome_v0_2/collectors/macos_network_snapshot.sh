#!/usr/bin/env bash
# AgentLink Rescue v0.5 draft collector
# Read-only snapshot collector. Do not mutate network state.
set -euo pipefail
OUT_DIR="${1:-./agentlink_network_snapshot_$(date +%Y%m%d_%H%M%S)}"
mkdir -p "$OUT_DIR"

run() {
  local name="$1"; shift
  {
    echo "# $name"
    echo "# command: $*"
    echo "# timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    "$@" 2>&1 || true
  } > "$OUT_DIR/$name.txt"
}

run sw_vers sw_vers
run uname uname -a
run network_services networksetup -listallnetworkservices
run network_service_order networksetup -listnetworkserviceorder
run scutil_dns scutil --dns
run scutil_proxy scutil --proxy
run scutil_nc_list scutil --nc list
run ifconfig ifconfig
run netstat_routes netstat -rn
run route_default route -n get default
run env_proxy /usr/bin/env
run lsof_listeners lsof -nP -iTCP -sTCP:LISTEN
run ps_network_processes ps aux
run curl_example_direct curl -I --connect-timeout 5 --noproxy '*' https://example.com
run curl_example_default curl -I --connect-timeout 5 https://example.com
run dns_example dscacheutil -q host -a name example.com

# Optional app-specific files. Redact before sharing.
{
  echo "# git proxy config"
  git config --list --show-origin 2>/dev/null | grep -i proxy || true
} > "$OUT_DIR/git_proxy.txt"

{
  echo "# npm proxy config"
  npm config get proxy 2>/dev/null || true
  npm config get https-proxy 2>/dev/null || true
  npm config get registry 2>/dev/null || true
} > "$OUT_DIR/npm_config.txt"

{
  echo "# docker config exists"
  if [ -f "$HOME/.docker/config.json" ]; then
    echo "$HOME/.docker/config.json exists; redact before sharing"
    sed -E 's/(auth|identitytoken|credsStore|credHelpers)": *"[^"]+"/\1":"<redacted>"/g' "$HOME/.docker/config.json" 2>/dev/null || true
  else
    echo "No ~/.docker/config.json"
  fi
} > "$OUT_DIR/docker_config_redacted.txt"

echo "Snapshot written to $OUT_DIR"
