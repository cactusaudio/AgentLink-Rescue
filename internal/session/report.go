package session

import (
	"encoding/json"
	"strings"

	"cactus-agentlink-rescue/internal/safety"
)

func HumanReport(s Session) string {
	var b strings.Builder
	b.WriteString("Cactus AgentLink Rescue " + s.ToolVersion + "\n\n")
	b.WriteString("Session: " + s.SessionID + "\n")
	if s.SelectedTarget != "" {
		b.WriteString("Target: " + s.SelectedTarget + "\n")
	}
	if s.SelectedRecipe != "" {
		b.WriteString("Recipe: " + s.SelectedRecipe + "\n")
	}
	if s.RiskLevel != "" {
		b.WriteString("Risk: " + s.RiskLevel + "\n")
	}
	if s.DryRun {
		b.WriteString("Dry run: true\n")
	}
	if s.FinalState != "" {
		b.WriteString("Recipe status: " + s.FinalState + "\n")
	}
	if isTemplateOnly(s) {
		b.WriteString("Template generation: static config template only; not proven operational until explicit online smoke test passes.\n")
	}
	if s.SnapshotID != "" {
		b.WriteString("Snapshot ID: " + s.SnapshotID + "\n")
	}
	if s.SnapshotPath != "" {
		b.WriteString("Snapshot path: " + s.SnapshotPath + "\n")
	}
	if len(s.ChangedFiles) > 0 {
		b.WriteString("\nChanged files:\n")
		for _, path := range s.ChangedFiles {
			b.WriteString("- " + path + "\n")
		}
	}
	if len(s.VerifierResults) > 0 {
		b.WriteString("\nVerifier results:\n")
		for _, v := range s.VerifierResults {
			line := "- " + v.ID + ": " + v.Status
			if v.Evidence != "" {
				line += " (" + v.Evidence + ")"
			}
			if v.Error != "" {
				line += " (" + v.Error + ")"
			}
			b.WriteString(line + "\n")
		}
	}
	if len(s.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, w := range s.Warnings {
			b.WriteString("- " + w + "\n")
		}
	}
	if s.SnapshotID != "" {
		b.WriteString("\nRollback:\n")
		b.WriteString("agentlink restore last\n")
	}
	return safety.RedactSensitive(b.String())
}

func AgentDispatch(s Session) string {
	out := map[string]any{
		"schemaVersion":     1,
		"toolVersion":       s.ToolVersion,
		"sessionID":         s.SessionID,
		"selectedTarget":    s.SelectedTarget,
		"selectedRecipe":    s.SelectedRecipe,
		"riskLevel":         s.RiskLevel,
		"dryRun":            s.DryRun,
		"snapshotID":        s.SnapshotID,
		"changedFiles":      s.ChangedFiles,
		"commandsRun":       s.CommandsRun,
		"verifierResults":   s.VerifierResults,
		"warnings":          s.Warnings,
		"finalState":        s.FinalState,
		"rollbackAvailable": s.RollbackAvailable,
		"exactNextTask":     nextTask(s),
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return safety.RedactSensitive(string(data))
}

func nextTask(s Session) string {
	for _, v := range s.VerifierResults {
		if v.Status == "fail" {
			return "Inspect the failed verifier with: agentlink report --for-codex --latest"
		}
	}
	if s.FinalState == "success" || s.FinalState == "ok" {
		return "No immediate action required."
	}
	if s.FinalState == "needs_user_secret" {
		return "Set DEEPSEEK_API_KEY in the shell, then run: agentlink recipe run codex-deepseek-online-smoke-test --yes"
	}
	if s.FinalState == "needs_online_smoke_test" {
		return "Run: agentlink recipe run codex-deepseek-online-smoke-test --yes"
	}
	if s.FinalState == "dry-run" && s.SelectedRecipe != "" {
		return "Review planned actions, then run: agentlink recipe run " + s.SelectedRecipe + " --yes"
	}
	return "Run: agentlink doctor"
}

func isTemplateOnly(s Session) bool {
	return s.FinalState == "config_template_generated" ||
		s.FinalState == "needs_user_secret" ||
		s.FinalState == "needs_online_smoke_test" ||
		containsWarning(s.Warnings, "provider_smoke_test_required")
}

func containsWarning(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
