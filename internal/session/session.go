package session

import (
	"time"

	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/verifier"
)

type Session struct {
	SchemaVersion        int                  `json:"schemaVersion"`
	ToolVersion          string               `json:"toolVersion"`
	SessionID            string               `json:"sessionID"`
	StartedAt            string               `json:"startedAt"`
	EndedAt              string               `json:"endedAt,omitempty"`
	CommandLine          []string             `json:"commandLine"`
	RealUser             string               `json:"realUser"`
	HostSummary          string               `json:"hostSummary"`
	InitialFacts         facts.Facts          `json:"initialFacts"`
	SelectedTarget       string               `json:"selectedTarget,omitempty"`
	SelectedRecipe       string               `json:"selectedRecipe,omitempty"`
	RiskLevel            string               `json:"riskLevel,omitempty"`
	DryRun               bool                 `json:"dryRun"`
	SnapshotID           string               `json:"snapshotID,omitempty"`
	SnapshotPath         string               `json:"snapshotPath,omitempty"`
	ChangedFiles         []string             `json:"changedFiles"`
	CommandsRun          []snapshot.ActionLog `json:"commandsRun"`
	VerifierResults      []verifier.Result    `json:"verifierResults"`
	Warnings             []string             `json:"warnings,omitempty"`
	FinalState           string               `json:"finalState"`
	RollbackAvailable    bool                 `json:"rollbackAvailable"`
	HumanReportPath      string               `json:"humanReportPath,omitempty"`
	AgentDispatchPath    string               `json:"agentDispatchPath,omitempty"`
	BrainEnabled         bool                 `json:"brainEnabled,omitempty"`
	BrainModelID         string               `json:"brainModelID,omitempty"`
	BrainBackend         string               `json:"brainBackend,omitempty"`
	PlannerCalls         []PlannerCall        `json:"plannerCalls,omitempty"`
	PlannerRawOutputPath string               `json:"plannerRawOutputPath,omitempty"`
	PlannerDecision      any                  `json:"plannerDecision,omitempty"`
	PlannerIntent        string               `json:"plannerIntent,omitempty"`
	PlannerConfidence    float64              `json:"plannerConfidence,omitempty"`
	ValidationErrors     []string             `json:"validationErrors,omitempty"`
	LoopStateTransitions []string             `json:"loopStateTransitions,omitempty"`
	StopReason           string               `json:"stopReason,omitempty"`
}

type PlannerCall struct {
	Index      int    `json:"index"`
	Backend    string `json:"backend"`
	DurationMs int64  `json:"durationMs"`
	RawOutput  string `json:"rawOutput,omitempty"`
	Validation string `json:"validation,omitempty"`
	Correction bool   `json:"correction,omitempty"`
}

func New(toolVersion string, commandLine []string) Session {
	now := time.Now()
	return Session{
		SchemaVersion:        1,
		ToolVersion:          toolVersion,
		SessionID:            now.Format("20060102-150405.000000000"),
		StartedAt:            now.Format(time.RFC3339),
		CommandLine:          append([]string(nil), commandLine...),
		ChangedFiles:         []string{},
		CommandsRun:          []snapshot.ActionLog{},
		VerifierResults:      []verifier.Result{},
		Warnings:             []string{},
		PlannerCalls:         []PlannerCall{},
		ValidationErrors:     []string{},
		LoopStateTransitions: []string{},
	}
}

func (s *Session) Finish(state string) {
	s.FinalState = state
	s.EndedAt = time.Now().Format(time.RFC3339)
}
