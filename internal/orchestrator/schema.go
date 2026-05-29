package orchestrator

import (
	"encoding/json"
	"fmt"

	"cactus-agentlink-rescue/internal/genomekernel"
)

type Report struct {
	SchemaVersion               int            `json:"schemaVersion"`
	ToolVersion                 string         `json:"toolVersion"`
	StartedAt                   string         `json:"startedAt"`
	EndedAt                     string         `json:"endedAt"`
	Mode                        string         `json:"mode"`
	Target                      string         `json:"target"`
	Status                      string         `json:"status"`
	FailureClasses              []string       `json:"failureClasses"`
	BrainAvailable              bool           `json:"brainAvailable"`
	GemmaCalled                 bool           `json:"gemmaCalled"`
	SupervisorMode              string         `json:"supervisorMode"`
	GemmaCallCount              int            `json:"gemmaCallCount,omitempty"`
	GemmaUsed                   bool           `json:"gemmaUsed,omitempty"`
	PlanSource                  string         `json:"planSource,omitempty"`
	GemmaOverrideAccepted       bool           `json:"gemmaOverrideAccepted,omitempty"`
	GemmaOverrideRejectedReason string         `json:"gemmaOverrideRejectedReason,omitempty"`
	SelectedRecipe              string         `json:"selectedRecipe,omitempty"`
	RequiresAdmin               bool           `json:"requiresAdmin"`
	TerminalTicketPath          string         `json:"terminalTicketPath,omitempty"`
	SnapshotID                  string         `json:"snapshotID,omitempty"`
	RollbackAvailable           bool           `json:"rollbackAvailable"`
	RollbackCommand             string         `json:"rollbackCommand,omitempty"`
	RestartGate                 map[string]any `json:"restartGate,omitempty"`
	Cycles                      []Cycle        `json:"cycles"`
	VerifierResults             []any          `json:"verifierResults,omitempty"`
	WorsenedSignals             []string       `json:"worsenedSignals,omitempty"`
	IncidentReportPath          string         `json:"incidentReportPath,omitempty"`
	SupportBundlePath           string         `json:"supportBundlePath,omitempty"`
	HumanSummary                string         `json:"humanSummary"`
	NextAction                  string         `json:"nextAction,omitempty"`
	Warnings                    []string       `json:"warnings,omitempty"`
	GenomeAdvisory              *genomekernel.Advisory `json:"genomeAdvisory,omitempty"`
}

type Cycle struct {
	Index                int            `json:"index"`
	State                string         `json:"state"`
	FactsSummary         map[string]any `json:"factsSummary,omitempty"`
	FailureClasses       []string       `json:"failureClasses,omitempty"`
	SupervisorDecision   any            `json:"supervisorDecision,omitempty"`
	PlanValidation       map[string]any `json:"planValidation,omitempty"`
	DryRun               any            `json:"dryRun,omitempty"`
	Execution            any            `json:"execution,omitempty"`
	Verification         any            `json:"verification,omitempty"`
	PostflightComparison any            `json:"postflightComparison,omitempty"`
	Result               string         `json:"result"`
}

type RescuePlanDecision struct {
	SchemaVersion          int      `json:"schemaVersion"`
	Intent                 string   `json:"intent"`
	FailureClass           string   `json:"failureClass"`
	Confidence             float64  `json:"confidence"`
	SelectedRecipe         string   `json:"selectedRecipe"`
	Evidence               []string `json:"evidence"`
	RequiresAdmin          bool     `json:"requiresAdmin"`
	RequiresTerminalTicket bool     `json:"requiresTerminalTicket"`
	RequiresRestartGate    bool     `json:"requiresRestartGate"`
	RequiresUserConsent    bool     `json:"requiresUserConsent"`
	ExpectedVerifiers      []string `json:"expectedVerifiers"`
	ExplanationForUser     string   `json:"explanationForUser"`
	StopReason             string   `json:"stopReason,omitempty"`
}

func MarshalReport(report Report) string {
	data, _ := json.MarshalIndent(report, "", "  ")
	return string(data)
}

func ValidatePlan(decision RescuePlanDecision, tunHighConfidence bool) error {
	if decision.SchemaVersion != 1 {
		return fmt.Errorf("invalid rescue plan schema version")
	}
	switch decision.Intent {
	case "repair", "probe", "report", "restart_gate", "manual_action", "last_resort_offer":
	default:
		return fmt.Errorf("invalid rescue plan intent: %s", decision.Intent)
	}
	if decision.Confidence < 0 || decision.Confidence > 1 {
		return fmt.Errorf("confidence out of range")
	}
	if decision.Intent == "repair" && decision.Confidence < 0.55 {
		return fmt.Errorf("repair confidence below threshold")
	}
	if decision.Intent == "repair" && decision.SelectedRecipe == "" {
		return fmt.Errorf("repair requires selected recipe")
	}
	if decision.Intent == "repair" && decision.SelectedRecipe != "macos-clash-tun-force-repair" {
		return fmt.Errorf("unknown or unsupported rescue recipe: %s", decision.SelectedRecipe)
	}
	if decision.Intent == "last_resort_offer" && decision.SelectedRecipe != "" && decision.SelectedRecipe != "macos-clean-network-baseline-reset" {
		return fmt.Errorf("unsupported last-resort recipe: %s", decision.SelectedRecipe)
	}
	if tunHighConfidence && (decision.SelectedRecipe == "standard" || decision.SelectedRecipe == "deep" || decision.SelectedRecipe == "standard-system-reset") {
		return fmt.Errorf("standard/deep refused while high-confidence TUN repair is applicable")
	}
	if decision.RequiresAdmin && !decision.RequiresTerminalTicket {
		return fmt.Errorf("privileged rescue plan must use terminal ticket")
	}
	if decision.Intent == "repair" && decision.RequiresAdmin && !decision.RequiresUserConsent {
		return fmt.Errorf("privileged repair requires explicit user consent")
	}
	return nil
}
