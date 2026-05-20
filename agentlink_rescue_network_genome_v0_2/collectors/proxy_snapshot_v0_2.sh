#!/usr/bin/env bash
# Read-only proxy plane collector for AgentLink Rescue v0.2.
set -u
OUT="${1:-proxy_snapshot.txt}"
redact(){ sed -E 's#(https?://|socks5?h?://)([^/@:]+):([^/@]+)@#\1REDACTED:REDACTED@#g'; }
{
 echo "## scutil --proxy"; scutil --proxy 2>&1
 echo "## env proxy"; env | sort | grep -Ei 'proxy|no_proxy|ssl|cert|ca' || true
 echo "## system proxy per service";
 networksetup -listallnetworkservices | tail -n +2 | sed 's/^\*//' | while IFS= read -r s; do
   echo "--- $s"; networksetup -getwebproxy "$s"; networksetup -getsecurewebproxy "$s"; networksetup -getsocksfirewallproxy "$s"; networksetup -getautoproxyurl "$s"; networksetup -getproxybypassdomains "$s";
 done
 echo "## local listeners"; lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null | grep -Ei '(789|808|108|3128|proxy|clash|mihomo)' || true
} | redact > "$OUT"
echo "$OUT"
