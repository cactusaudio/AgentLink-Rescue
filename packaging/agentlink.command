#!/bin/bash
set -u

ROOT="$(cd "$(dirname "$0")" && pwd)"
BIN="$ROOT/bin/agentlink"
FALLBACK="$ROOT/rescue.sh"

clear
echo "Cactus AgentLink Rescue 0.3.1"
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
  echo "  1. Doctor"
  echo "  2. Brain Doctor"
  echo "  3. Brain Selftest"
  echo "  4. Brain Plan"
  echo "  5. Brain Auto Repair Dry-Run"
  echo "  6. Network Rescue Safe"
  echo "  7. Network Rescue Standard"
  echo "  8. Repair PATH"
  echo "  9. Proxy Detect"
  echo "  10. Codex Config Doctor"
  echo "  11. Rollback Last"
  echo "  12. Exit"
  printf "> "
  read -r choice
  case "$choice" in
    1)
      "$BIN" doctor
      break
      ;;
    2)
      "$BIN" brain doctor
      break
      ;;
    3)
      "$BIN" brain selftest
      break
      ;;
    4)
      "$BIN" brain plan --target path
      break
      ;;
    5)
      "$BIN" repair --auto --brain --target path --dry-run
      break
      ;;
    6)
      sudo "$BIN" rescue --level safe
      break
      ;;
    7)
      sudo "$BIN" rescue --level standard
      break
      ;;
    8)
      echo "PATH repair writes a managed block to ~/.zshrc and creates a rollback snapshot."
      read -r -p "Type YES to continue: " confirm
      if [ "$confirm" = "YES" ]; then
        "$BIN" recipe run macos-zsh-path-repair --yes
      else
        "$BIN" recipe run macos-zsh-path-repair --dry-run
      fi
      break
      ;;
    9)
      "$BIN" proxy detect
      break
      ;;
    10)
      "$BIN" config doctor
      break
      ;;
    11)
      "$BIN" restore last || sudo "$BIN" rollback --last
      break
      ;;
    12)
      exit 0
      ;;
    *)
      echo "Invalid choice."
      ;;
  esac
done

echo
read -r -p "Press Return to close. " _
