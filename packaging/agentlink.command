#!/bin/bash
set -u

ROOT="$(cd "$(dirname "$0")" && pwd)"
BIN="$ROOT/bin/agentlink"
FALLBACK="$ROOT/rescue.sh"

clear
echo "Cactus AgentLink Rescue 0.2.1"
echo

xattr -cr "$ROOT" 2>/dev/null || true
chmod +x "$BIN" "$FALLBACK" 2>/dev/null || true

if [ ! -x "$BIN" ]; then
  echo "agentlink binary was not found or is not executable."
  if [ -f "$FALLBACK" ]; then
    echo "Fallback available:"
    echo "  /bin/bash \"$FALLBACK\" diagnose"
    echo "  sudo /bin/bash \"$FALLBACK\" safe"
    echo "  sudo /bin/bash \"$FALLBACK\" standard"
    echo "  sudo /bin/bash \"$FALLBACK\" deep"
  fi
  echo
  read -r -p "Press Return to close. " _
  exit 1
fi

if ! "$BIN" version >/dev/null 2>&1; then
  echo "agentlink could not run. Gatekeeper or quarantine may be blocking it."
  echo
  echo "Fallback commands:"
  echo "  /bin/bash \"$FALLBACK\" diagnose"
  echo "  sudo /bin/bash \"$FALLBACK\" safe"
  echo "  sudo /bin/bash \"$FALLBACK\" standard"
  echo "  sudo /bin/bash \"$FALLBACK\" deep"
  echo
  read -r -p "Press Return to close. " _
  exit 1
fi

echo "Running diagnose first..."
"$BIN" diagnose
echo

while true; do
  echo "Choose an action:"
  echo "  1. Doctor"
  echo "  2. Network Rescue Safe"
  echo "  3. Network Rescue Standard"
  echo "  4. Repair PATH"
  echo "  5. Proxy Detect"
  echo "  6. Codex Config Doctor"
  echo "  7. Rollback Last"
  echo "  8. Exit"
  printf "> "
  read -r choice
  case "$choice" in
    1)
      "$BIN" doctor
      break
      ;;
    2)
      sudo "$BIN" rescue --level safe
      break
      ;;
    3)
      sudo "$BIN" rescue --level standard
      break
      ;;
    4)
      echo "PATH repair writes a managed block to ~/.zshrc and creates a rollback snapshot."
      read -r -p "Type YES to continue: " confirm
      if [ "$confirm" = "YES" ]; then
        "$BIN" recipe run macos-zsh-path-repair --yes
      else
        "$BIN" recipe run macos-zsh-path-repair --dry-run
      fi
      break
      ;;
    5)
      "$BIN" proxy detect
      break
      ;;
    6)
      "$BIN" config doctor
      break
      ;;
    7)
      "$BIN" restore last || sudo "$BIN" rollback --last
      break
      ;;
    8)
      exit 0
      ;;
    *)
      echo "Invalid choice."
      ;;
  esac
done

echo
read -r -p "Press Return to close. " _
