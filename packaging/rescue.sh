#!/bin/bash
# Cactus AgentLink Rescue fallback script.
# This is intentionally simpler than the Go engine. It exists for cases where
# the binary is blocked or unavailable. Prefer bin/agentlink whenever possible.

set -u

APP="Cactus AgentLink Rescue"
BASE="/Library/Application Support/$APP"
RESTORE="$BASE/restore-points"
TS="$(date +%Y%m%d-%H%M%S)"
POINT="$RESTORE/fallback-$TS"

usage() {
  echo "Usage:"
  echo "  /bin/bash rescue.sh diagnose"
  echo "  sudo /bin/bash rescue.sh safe"
  echo "  sudo /bin/bash rescue.sh standard"
  echo "  sudo /bin/bash rescue.sh deep"
  echo "  sudo /bin/bash rescue.sh rollback /path/to/restore-point"
}

need_root() {
  if [ "$(id -u)" != "0" ]; then
    echo "Root required. Re-run with sudo."
    exit 40
  fi
}

make_point() {
  mkdir -p "$POINT/files" "$POINT/quarantine"
  echo "restore point: $POINT"
}

log() {
  mkdir -p "$POINT"
  printf '%s | %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >> "$POINT/commands.log"
}

run_cmd() {
  log "$*"
  "$@"
}

backup_path() {
  src="$1"
  [ -e "$src" ] || return 0
  dst="$POINT/files${src}"
  mkdir -p "$(dirname "$dst")"
  if [ -d "$src" ]; then
    /usr/bin/ditto "$src" "$dst"
  else
    /bin/cp -p "$src" "$dst"
  fi
}

quarantine_path() {
  src="$1"
  [ -e "$src" ] || return 0
  backup_path "$src"
  dst="$POINT/quarantine/$(basename "$src")"
  i=1
  while [ -e "$dst" ]; do
    dst="$POINT/quarantine/$(basename "$src").$i"
    i=$((i + 1))
  done
  /bin/mv "$src" "$dst"
  echo "$src -> $dst" >> "$POINT/quarantine.log"
}

services() {
  /usr/sbin/networksetup -listallnetworkservices 2>/dev/null | /usr/bin/awk 'BEGIN{skip=0} /^An asterisk/ {next} /^\*/ {next} NF {print}'
}

hardware_devices() {
  /usr/sbin/networksetup -listallhardwareports 2>/dev/null | /usr/bin/awk '/^Device: / {print $2}'
}

diagnose() {
  echo "Cactus AgentLink Rescue fallback diagnose"
  echo
  /usr/bin/sw_vers 2>/dev/null || true
  /bin/hostname 2>/dev/null || true
  echo
  echo "Network location:"
  /usr/sbin/networksetup -getcurrentlocation 2>/dev/null || true
  echo
  echo "Services:"
  /usr/sbin/networksetup -listallnetworkservices 2>/dev/null || true
  echo
  echo "Default route:"
  /sbin/route -n get default 2>/dev/null || true
  echo
  echo "Proxy:"
  /usr/sbin/scutil --proxy 2>/dev/null || true
  echo
  echo "DNS apple.com:"
  /usr/bin/dscacheutil -q host -a name apple.com 2>/dev/null || true
}

safe() {
  need_root
  make_point
  services | while IFS= read -r svc; do
    [ -n "$svc" ] || continue
    run_cmd /usr/sbin/networksetup -setwebproxystate "$svc" off || true
    run_cmd /usr/sbin/networksetup -setsecurewebproxystate "$svc" off || true
    run_cmd /usr/sbin/networksetup -setsocksfirewallproxystate "$svc" off || true
    run_cmd /usr/sbin/networksetup -setautoproxystate "$svc" off || true
    run_cmd /usr/sbin/networksetup -setdnsservers "$svc" empty || true
    run_cmd /usr/sbin/networksetup -setsearchdomains "$svc" empty || true
    run_cmd /usr/sbin/networksetup -setdhcp "$svc" || true
    run_cmd /usr/sbin/networksetup -setv6automatic "$svc" || true
  done
  hardware_devices | while IFS= read -r dev; do
    [ -n "$dev" ] || continue
    run_cmd /usr/sbin/ipconfig set "$dev" DHCP || true
  done
  run_cmd /usr/bin/dscacheutil -flushcache || true
  run_cmd /usr/bin/killall -HUP mDNSResponder || true
  echo "Safe rescue complete."
  echo "Backup path: $POINT"
}

standard() {
  need_root
  safe
  /usr/sbin/networksetup -getcurrentlocation > "$POINT/previous_location.txt" 2>/dev/null || true
  loc="AgentLink-Clean-$TS"
  run_cmd /usr/sbin/networksetup -createlocation "$loc" populate || true
  run_cmd /usr/sbin/networksetup -switchtolocation "$loc" || true
  run_cmd /usr/sbin/networksetup -detectnewhardware || true
  run_cmd /sbin/route -n flush || true
  run_cmd /sbin/route -n flush || true
  hardware_devices | while IFS= read -r dev; do
    [ -n "$dev" ] || continue
    run_cmd /usr/sbin/ipconfig set "$dev" DHCP || true
  done
  echo "Standard rescue complete."
  echo "Backup path: $POINT"
}

deep() {
  need_root
  standard
  for p in \
    "/Library/Preferences/SystemConfiguration/preferences.plist" \
    "/Library/Preferences/SystemConfiguration/NetworkInterfaces.plist" \
    "/Library/Preferences/SystemConfiguration/com.apple.airport.preferences.plist" \
    "/Library/Preferences/SystemConfiguration/com.apple.network.identification.plist" \
    "/Library/Preferences/SystemConfiguration/com.apple.network.eapolclient.configuration.plist" \
    "/Library/Preferences/SystemConfiguration/com.apple.wifi.message-tracer.plist"
  do
    quarantine_path "$p"
  done
  echo "Deep rescue complete. Reboot is recommended."
  echo "Backup path: $POINT"
}

rollback() {
  need_root
  rp="${1:-}"
  if [ -z "$rp" ] || [ ! -d "$rp" ]; then
    echo "Restore point directory required."
    usage
    exit 50
  fi
  files="$rp/files"
  quarantine_log="$rp/quarantine.log"
  previous_location="$rp/previous_location.txt"
  echo "Fallback rollback target: $rp"
  if [ -f "$previous_location" ]; then
    previous="$(/usr/bin/head -n 1 "$previous_location")"
    if [ -n "$previous" ]; then
      echo "Restoring network location: $previous"
      if /usr/sbin/networksetup -switchtolocation "$previous"; then
        echo "Network location restored."
      else
        echo "Warning: network location restore failed; continuing with files/quarantine rollback."
      fi
    fi
  fi
  if [ -d "$files" ]; then
    echo "Restoring backed-up files from $files"
    /usr/bin/ditto "$files" /
  fi
  if [ -f "$quarantine_log" ]; then
    while IFS= read -r line; do
      src="${line%% -> *}"
      dst="${line#* -> }"
      [ -n "$src" ] && [ -n "$dst" ] || continue
      if [ -e "$src" ]; then
        echo "Not moving quarantine item because original exists: $src"
        continue
      fi
      if [ -e "$dst" ]; then
        mkdir -p "$(dirname "$src")"
        /bin/mv "$dst" "$src"
        echo "Restored quarantine item: $src"
      fi
    done < "$quarantine_log"
  fi
  echo "Fallback rollback complete."
}

case "${1:-}" in
  diagnose) diagnose ;;
  safe) safe ;;
  standard) standard ;;
  deep) deep ;;
  rollback) rollback "${2:-}" ;;
  *) usage; exit 50 ;;
esac
