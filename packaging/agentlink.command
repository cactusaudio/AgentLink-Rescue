#!/bin/bash
set -u

ROOT="$(cd "$(dirname "$0")" && pwd)"
BIN="$ROOT/bin/agentlink"
FALLBACK="$ROOT/rescue.sh"

clear
echo "Cactus AgentLink Rescue 0.4.3"
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
  echo "  1. Guided Rescue Analyze"
  echo "  2. Doctor"
  echo "  3. Brain Doctor"
  echo "  4. Brain Selftest"
  echo "  5. Brain Plan"
  echo "  6. Brain Auto Repair Dry-Run"
  echo "  7. Installer Center Doctor"
  echo "  8. Brain Chat Sandbox"
  echo "  9. Network Rescue Safe"
  echo "  10. Network Rescue Standard"
  echo "  11. Repair PATH"
  echo "  12. Proxy Detect"
  echo "  13. Codex Config Doctor"
  echo "  14. Rollback Last"
  echo "  15. Exit"
  printf "> "
  read -r choice
  case "$choice" in
    1)
      "$BIN" guided rescue --target auto --dry-run
      break
      ;;
    2)
      "$BIN" doctor
      break
      ;;
    3)
      "$BIN" brain doctor
      break
      ;;
    4)
      "$BIN" brain selftest
      break
      ;;
    5)
      "$BIN" brain plan --target path
      break
      ;;
    6)
      "$BIN" repair --auto --brain --target path --dry-run
      break
      ;;
    7)
      "$BIN" installer doctor
      break
      ;;
    8)
      read -r -p "Brain prompt: " prompt
      "$BIN" brain chat --prompt "$prompt"
      break
      ;;
    9)
      sudo "$BIN" rescue --level safe
      break
      ;;
    10)
      sudo "$BIN" rescue --level standard
      break
      ;;
    11)
      echo "PATH repair writes a managed block to ~/.zshrc and creates a rollback snapshot."
      read -r -p "Type YES to continue: " confirm
      if [ "$confirm" = "YES" ]; then
        "$BIN" recipe run macos-zsh-path-repair --yes
      else
        "$BIN" recipe run macos-zsh-path-repair --dry-run
      fi
      break
      ;;
    12)
      "$BIN" proxy detect
      break
      ;;
    13)
      "$BIN" config doctor
      break
      ;;
    14)
      "$BIN" restore last || sudo "$BIN" rollback --last
      break
      ;;
    15)
      exit 0
      ;;
    *)
      echo "Invalid choice."
      ;;
  esac
done

echo
read -r -p "Press Return to close. " _
