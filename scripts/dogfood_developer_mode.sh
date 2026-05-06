#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# v0.5.1 brain-aware GUI dogfood: Brain tools move behind Developer Mode.
APPSTATE="gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/AppState.swift"
MAIN="gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Views/MainWindow.swift"
SETTINGS="gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Views/SettingsView.swift"

grep -q 'developerModeEnabled' "$APPSTATE"
grep -q 'private let mainPages: \[RescuePage\] = \[.dashboard, .reports, .settings\]' "$MAIN"
grep -q 'private let developerPages: \[RescuePage\]' "$MAIN"
grep -q 'Section("Developer Mode")' "$MAIN"
grep -q 'Enable Developer Mode in Settings' "$MAIN"
grep -q 'Toggle("Enable Developer Mode"' "$SETTINGS"

for page in guided readiness installer doctor brain plan dryRun rescue rollback; do
  grep -q "\\.$page" "$MAIN" || {
    echo "Developer Mode missing $page page" >&2
    exit 1
  }
done

echo "dogfood_developer_mode OK"
