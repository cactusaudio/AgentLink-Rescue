package devdoctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion int         `json:"schemaVersion"`
	ToolVersion   string      `json:"toolVersion"`
	CreatedAt     string      `json:"createdAt"`
	Status        string      `json:"status"`
	Tools         []ToolCheck `json:"tools"`
	Warnings      []string    `json:"warnings,omitempty"`
	NextActions   []string    `json:"nextActions,omitempty"`
}

type ToolCheck struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	Status    string `json:"status"`
}

func Run(ctx context.Context, runner command.Runner, version string) Report {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	report := Report{
		SchemaVersion: 1,
		ToolVersion:   version,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Status:        "ok",
	}
	specs := []struct {
		id       string
		name     string
		required bool
		args     []string
	}{
		{"git", "git", true, []string{"--version"}},
		{"curl", "curl", true, []string{"--version"}},
		{"node", "node", false, []string{"--version"}},
		{"npm", "npm", false, []string{"--version"}},
		{"homebrew", "brew", false, []string{"--version"}},
		{"xcode-clt", "xcode-select", false, []string{"-p"}},
	}
	for _, spec := range specs {
		check := ToolCheck{ID: spec.id, Name: spec.name, Required: spec.required, Status: "missing"}
		path := commandPath(spec.name)
		if path != "" {
			check.Installed = true
			check.Path = path
			check.Status = "ok"
			res := runner.Run(ctx, path, spec.args...)
			if res.ExitCode == 0 {
				check.Version = firstLine(safety.RedactSensitive(res.Stdout + res.Stderr))
			} else {
				check.Status = "warn"
				check.Version = firstLine(safety.RedactSensitive(res.Stderr + res.Error))
			}
		}
		if spec.required && !check.Installed {
			report.Status = "missing_required"
			report.NextActions = append(report.NextActions, fmt.Sprintf("Install %s from Apple/macOS or the official vendor source.", spec.name))
		}
		report.Tools = append(report.Tools, check)
	}
	if !hasTool(report.Tools, "node") || !hasTool(report.Tools, "npm") {
		report.Warnings = append(report.Warnings, "Node.js/npm missing: AI CLI installers may need Node before Codex, Claude Code, or Gemini CLI can be installed.")
	}
	if !hasTool(report.Tools, "homebrew") {
		report.Warnings = append(report.Warnings, "Homebrew missing: npm-based installers can still work if Node/npm are present.")
	}
	return report
}

func commandPath(name string) string {
	if name == "" {
		return ""
	}
	if p, err := exec.LookPath(name); err == nil && p != "" && system.CommandExists(p) {
		return p
	}
	for _, prefix := range []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"} {
		p := prefix + "/" + name
		if system.CommandExists(p) {
			return p
		}
	}
	return ""
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	line := strings.Split(text, "\n")[0]
	if len(line) > 180 {
		line = line[:180]
	}
	return line
}

func hasTool(tools []ToolCheck, id string) bool {
	for _, tool := range tools {
		if tool.ID == id && tool.Installed {
			return true
		}
	}
	return false
}
