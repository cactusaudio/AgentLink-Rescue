#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# v0.5.1 brain-aware GUI dogfood: main rescue stays one-button while Brain/Gemma details remain secondary.
DASH="gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue/Views/StatusDashboardView.swift"

grep -q 'Fix My Connection' "$DASH"
grep -q 'runGuidedRescue(allowRepair: false' "$DASH"
grep -q 'Export Support Bundle' "$DASH"
grep -q 'Developer Mode' "$DASH"

if grep -q 'Allow reversible repairs' "$DASH"; then
  echo "Dashboard still exposes repair mode toggle instead of one-button analyze flow" >&2
  exit 1
fi
if grep -q 'Open Expert Console' "$DASH"; then
  echo "Dashboard still exposes expert-console button clutter" >&2
  exit 1
fi
if grep -q 'Installer Center' "$DASH"; then
  echo "Dashboard still exposes installer center as a main rescue button" >&2
  exit 1
fi
if grep -q 'Fix Clash / TUN Network' "$DASH"; then
  echo "Dashboard incorrectly presents AgentLink as a Clash/TUN-only fixer" >&2
  exit 1
fi

echo "dogfood_one_button_rescue OK"
