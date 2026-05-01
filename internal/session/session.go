package session

import (
	"time"

	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/verifier"
)

type Session struct {
	SchemaVersion     int                  `json:"schemaVersion"`
	ToolVersion       string               `json:"toolVersion"`
	SessionID         string               `json:"sessionID"`
	StartedAt         string               `json:"startedAt"`
	EndedAt           string               `json:"endedAt,omitempty"`
	CommandLine       []string             `json:"commandLine"`
	RealUser          string               `json:"realUser"`
	HostSummary       string               `json:"hostSummary"`
	InitialFacts      facts.Facts          `json:"initialFacts"`
	SelectedTarget    string               `json:"selectedTarget,omitempty"`
	SelectedRecipe    string               `json:"selectedRecipe,omitempty"`
	RiskLevel         string               `json:"riskLevel,omitempty"`
	DryRun            bool                 `json:"dryRun"`
	SnapshotID        string               `json:"snapshotID,omitempty"`
	SnapshotPath      string               `json:"snapshotPath,omitempty"`
	ChangedFiles      []string             `json:"changedFiles"`
	CommandsRun       []snapshot.ActionLog `json:"commandsRun"`
	VerifierResults   []verifier.Result    `json:"verifierResults"`
	Warnings          []string             `json:"warnings,omitempty"`
	FinalState        string               `json:"finalState"`
	RollbackAvailable bool                 `json:"rollbackAvailable"`
	HumanReportPath   string               `json:"humanReportPath,omitempty"`
	AgentDispatchPath string               `json:"agentDispatchPath,omitempty"`
}

func New(toolVersion string, commandLine []string) Session {
	now := time.Now()
	return Session{
		SchemaVersion:   1,
		ToolVersion:     toolVersion,
		SessionID:       now.Format("20060102-150405.000000000"),
		StartedAt:       now.Format(time.RFC3339),
		CommandLine:     append([]string(nil), commandLine...),
		ChangedFiles:    []string{},
		CommandsRun:     []snapshot.ActionLog{},
		VerifierResults: []verifier.Result{},
		Warnings:        []string{},
	}
}

func (s *Session) Finish(state string) {
	s.FinalState = state
	s.EndedAt = time.Now().Format(time.RFC3339)
}
