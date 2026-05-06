#!/bin/bash
set -u

ROOT="$(cd "$(dirname "$0")" && pwd)"
BIN="$ROOT/bin/agentlink"
FALLBACK="$ROOT/rescue.sh"

clear
echo "Cactus AgentLink Rescue 0.5.1"
echo

xattr -cr "$ROOT" 2>/dev/null || true
chmod +x "$BIN" "$FALLBACK" 2>/dev/null || true
chmod +x "$ROOT"/assets/runtimes/llama.cpp/*/llama-* "$ROOT"/assets/runtimes/llama.cpp/*/*.dylib 2>/dev/null || true

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

echo "Running doctor first..."
"$BIN" doctor
echo
echo "Brain assets are optional. If Brain Doctor reports missing assets, use:"
echo "  \"$BIN\" brain fetch"
echo

while true; do
  echo "Choose an action:"
  echo "  1. Fix My Connection Analyze"
  echo "  2. Readiness Center"
  echo "  3. Save Last-Good Profile"
  echo "  4. Create Support Bundle"
  echo "  5. Doctor"
  echo "  6. Brain Doctor"
  echo "  7. Brain Selftest"
  echo "  8. Brain Plan"
  echo "  9. Brain Auto Repair Dry-Run"
  echo "  10. Installer Center Doctor"
  echo "  11. Brain Chat Sandbox"
  echo "  12. Clash/TUN Repair Ticket"
  echo "  13. Targeted Clash/TUN Repair"
  echo "  14. Network Rescue Safe"
  echo "  15. Network System Reset"
  echo "  16. Repair PATH"
  echo "  17. Proxy Detect"
  echo "  18. Codex Config Doctor"
  echo "  19. Rollback Last"
  echo "  20. Exit"
  printf "> "
  read -r choice
  case "$choice" in
    1)
      "$BIN" orchestrator rescue --target auto --dry-run
      break
      ;;
    2)
      "$BIN" readiness doctor
      break
      ;;
    3)
      "$BIN" last-good save
      break
      ;;
    4)
      "$BIN" support bundle
      break
      ;;
    5)
      "$BIN" doctor
      break
      ;;
    6)
      "$BIN" brain doctor
      break
      ;;
    7)
      "$BIN" brain selftest
      break
      ;;
    8)
      "$BIN" brain plan --target path
      break
      ;;
    9)
      "$BIN" repair --auto --brain --target path --dry-run
      break
      ;;
    10)
      "$BIN" installer doctor
      break
      ;;
    11)
      read -r -p "Brain prompt: " prompt
      "$BIN" brain chat --prompt "$prompt"
      break
      ;;
    12)
      "$BIN" ticket create --type clash-tun-fix
      break
      ;;
    13)
      sudo "$BIN" rescue --level tun --yes
      break
      ;;
    14)
      sudo "$BIN" rescue --level safe
      break
      ;;
    15)
      sudo "$BIN" rescue --level standard-system-reset --yes
      break
      ;;
    16)
      echo "PATH repair writes a managed block to ~/.zshrc and creates a rollback snapshot."
      read -r -p "Type YES to continue: " confirm
      if [ "$confirm" = "YES" ]; then
        "$BIN" recipe run macos-zsh-path-repair --yes
      else
        "$BIN" recipe run macos-zsh-path-repair --dry-run
      fi
      break
      ;;
    17)
      "$BIN" proxy detect
      break
      ;;
    18)
      "$BIN" config doctor
      break
      ;;
    19)
      "$BIN" restore last || sudo "$BIN" rollback --last
      break
      ;;
    20)
      exit 0
      ;;
    *)
      echo "Invalid choice."
      ;;
  esac
done

echo
read -r -p "Press Return to close. " _
