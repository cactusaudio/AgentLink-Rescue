#!/usr/bin/env bash
# AgentLink Rescue v0.2 read-only macOS network snapshot collector.
# No mutation. Redacts obvious credentials/tokens from proxy URLs and env.
set -u
OUT_DIR="${1:-$HOME/Desktop/agentlink_rescue_snapshot_$(date +%Y%m%d_%H%M%S)}"
mkdir -p "$OUT_DIR"
redact() {
  sed -E \
    -e 's#(https?://|socks5?h?://)([^/@:]+):([^/@]+)@#\1REDACTED:REDACTED@#g' \
    -e 's#(OPENAI_API_KEY|ANTHROPIC_API_KEY|CODEX_API_KEY|GITHUB_TOKEN|GH_TOKEN)=.*#\1=REDACTED#g' \
    -e 's#(Authorization: Bearer )[A-Za-z0-9._-]+#\1REDACTED#g'
}
run() {
  local name="$1"; shift
  {
    echo "### $name"
    echo "### command: $*"
    "$@" 2>&1 | redact
  } > "$OUT_DIR/$name.txt" || true
}
run_shell() {
  local name="$1"; shift
  {
    echo "### $name"
    echo "### shell: $*"
    bash -lc "$*" 2>&1 | redact
  } > "$OUT_DIR/$name.txt" || true
}
run date date
run sw_vers sw_vers
run uname uname -a
run ifconfig ifconfig
run route_default route -n get default
run netstat_routes netstat -rn
run scutil_dns scutil --dns
run scutil_proxy scutil --proxy
run scutil_nwi scutil --nwi
run_shell scutil_nc 'scutil --nc list || true'
run network_services networksetup -listallnetworkservices
run network_service_order networksetup -listnetworkserviceorder
run hardware_ports networksetup -listallhardwareports
run_shell per_service_info 'for s in $(networksetup -listallnetworkservices | tail -n +2 | sed "s/^\*//"); do echo "--- $s"; networksetup -getinfo "$s"; networksetup -getdnsservers "$s"; networksetup -getsearchdomains "$s"; networksetup -getwebproxy "$s"; networksetup -getsecurewebproxy "$s"; networksetup -getsocksfirewallproxy "$s"; done'
run_shell proxy_env 'env | sort | grep -Ei "proxy|no_proxy|ssl|cert|ca|api|base" || true'
run_shell listeners 'lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null | grep -Ei "(789|808|108|3128|clash|mihomo|proxy|dns|53)" || true'
run_shell processes 'ps aux | grep -Ei "clash|mihomo|verge|wireguard|tailscale|zerotier|vpn|docker|colima|ollama|codex" | grep -v grep || true'
run_shell hosts_file 'cat /etc/hosts 2>/dev/null || true'
run_shell resolver_dir 'ls -la /etc/resolver 2>/dev/null && for f in /etc/resolver/*; do echo "--- $f"; cat "$f"; done 2>/dev/null || true'
run_shell git_config 'git config --list --show-origin 2>/dev/null | grep -Ei "proxy|ssl|url|instead|github" || true'
run_shell npm_config 'npm config list -l 2>/dev/null | grep -Ei "proxy|registry|strict-ssl|cafile" || true'
run_shell brew_env 'brew config 2>/dev/null; echo ---; brew --env 2>/dev/null | grep -Ei "proxy|ssl|cert|api|github|curl|git" || true'
run_shell docker_config 'docker info 2>/dev/null | sed -n "1,120p"; echo --- ~/.docker/config.json; python3 - <<PY 2>/dev/null
import json, os
p=os.path.expanduser("~/.docker/config.json")
try:
    data=json.load(open(p))
    for k in ["auths","credsStore","credHelpers"]: data.pop(k, None)
    print(json.dumps(data, indent=2))
except Exception as e: print(e)
PY'
run_shell pf_info 'pfctl -s info 2>/dev/null || true; pfctl -sr 2>/dev/null | head -200 || true'
run_shell quick_probes 'for h in 1.1.1.1 8.8.8.8; do ping -c 2 -t 3 $h; done; for h in example.com github.com api.openai.com; do echo --- $h; dig +time=2 +tries=1 $h; curl -Iv --max-time 10 https://$h/ 2>&1 | head -80; done'
cat > "$OUT_DIR/README.txt" <<EOF
AgentLink Rescue snapshot generated at $(date).
This snapshot is read-only but may contain hostnames, IPs, routes, domains, and redacted config values.
Review before sharing.
EOF
echo "$OUT_DIR"
