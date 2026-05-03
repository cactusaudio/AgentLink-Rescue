package installer

import "cactus-agentlink-rescue/internal/command"

type Report struct {
	SchemaVersion   int              `json:"schemaVersion"`
	ToolVersion     string           `json:"toolVersion"`
	ID              string           `json:"id"`
	DisplayName     string           `json:"displayName,omitempty"`
	Status          string           `json:"status"`
	Method          string           `json:"method"`
	RequiresNetwork bool             `json:"requiresNetwork"`
	RequiresAdmin   bool             `json:"requiresAdmin"`
	Commands        []CommandPlan    `json:"commands"`
	MethodStatuses  []MethodStatus   `json:"methodStatuses,omitempty"`
	Verification    []VerifyResult   `json:"verification"`
	Warnings        []string         `json:"warnings,omitempty"`
	AssetPath       string           `json:"assetPath,omitempty"`
	OfficialURL     string           `json:"officialURL,omitempty"`
	NextAction      string           `json:"nextAction,omitempty"`
	RawCommands     []command.Result `json:"rawCommands,omitempty"`
}

type MethodStatus struct {
	ID                  string      `json:"id"`
	Type                string      `json:"type"`
	Available           bool        `json:"available"`
	MissingDependencies []string    `json:"missingDependencies,omitempty"`
	Command             CommandPlan `json:"command,omitempty"`
}

type CommandPlan struct {
	Display string   `json:"display"`
	Path    string   `json:"path,omitempty"`
	Args    []string `json:"args,omitempty"`
	Mutates bool     `json:"mutates"`
}

type VerifyResult struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
	Error    string `json:"error,omitempty"`
}

type DoctorReport struct {
	SchemaVersion int      `json:"schemaVersion"`
	ToolVersion   string   `json:"toolVersion"`
	Reports       []Report `json:"installers"`
	Warnings      []string `json:"warnings,omitempty"`
}
