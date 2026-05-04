package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion int      `json:"schemaVersion"`
	ToolVersion   string   `json:"toolVersion"`
	Action        string   `json:"action"`
	Status        string   `json:"status"`
	ConfigPath    string   `json:"configPath,omitempty"`
	Commands      []string `json:"commands,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	Error         string   `json:"error,omitempty"`
}

func Doctor(ctx context.Context, runner command.Runner, home string) Report {
	rep := base("doctor")
	if path := commandPath("opencode"); path != "" {
		res := runner.Run(ctx, path, "--version")
		if command.Success(res) {
			rep.Status = "installed"
		} else {
			rep.Status = "installed_unverified"
			rep.Warnings = append(rep.Warnings, res.Error+res.Stderr)
		}
	} else {
		rep.Status = "missing"
		rep.Commands = []string{"agentlink installer dry-run opencode-cli --json"}
	}
	rep.ConfigPath = configPath(home)
	return rep
}

func InstallDryRun() Report {
	rep := base("install-dry-run")
	rep.Status = "dry_run"
	rep.Commands = []string{"npm install -g opencode-ai", "brew install anomalyco/tap/opencode"}
	rep.Warnings = []string{"AgentLink will not auto-run curl | bash for OpenCode."}
	return rep
}

func ConfigureLocalGemma(home string, yes bool) Report {
	rep := base("configure-local-gemma")
	rep.ConfigPath = configPath(home)
	template := `{
  "provider": {
    "agentlink-gemma": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "AgentLink Gemma",
      "options": {
        "baseURL": "http://127.0.0.1:8080/v1",
        "apiKey": "agentlink-local"
      },
      "models": {
        "gemma-4-e4b-it": {}
      }
    }
  }
}
`
	if !yes {
		rep.Status = "dry_run"
		rep.Commands = []string{"start Gemma server: agentlink brain server start --port 8080 --json", "write OpenCode provider agentlink-gemma to " + rep.ConfigPath}
		return rep
	}
	if err := os.MkdirAll(filepath.Dir(rep.ConfigPath), 0700); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	if system.Exists(rep.ConfigPath) {
		rep.Status = "manual_action_required"
		rep.Warnings = append(rep.Warnings, "existing OpenCode config found; AgentLink will not overwrite it in v0.5.0")
		return rep
	}
	if err := os.WriteFile(rep.ConfigPath, []byte(template), 0600); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	rep.Status = "configured"
	return rep
}

func InstallPluginDryRun() Report {
	rep := base("install-plugin")
	rep.Status = "dry_run"
	rep.Commands = []string{"copy opencode/agentlink-plugin into OpenCode plugin directory"}
	rep.Warnings = []string{"Plugin exposes read-only AgentLink tools only; direct sudo rescue execution is not exposed."}
	return rep
}

func Verify(ctx context.Context, runner command.Runner, home string) Report {
	rep := Doctor(ctx, runner, home)
	rep.Action = "verify"
	return rep
}

func Marshal(rep Report) string {
	data, _ := json.MarshalIndent(rep, "", "  ")
	return string(data)
}

func base(action string) Report {
	return Report{SchemaVersion: 1, ToolVersion: system.Version, Action: action, Status: "unknown"}
}

func configPath(home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".config", "opencode", "agentlink-gemma.json")
}

func commandPath(name string) string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, name)
		if system.CommandExists(path) {
			return path
		}
	}
	for _, path := range []string{"/opt/homebrew/bin/" + name, "/usr/local/bin/" + name, "/usr/bin/" + name} {
		if system.CommandExists(path) {
			return path
		}
	}
	return ""
}

func Human(rep Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "OpenCode %s: %s\n", rep.Action, rep.Status)
	for _, c := range rep.Commands {
		fmt.Fprintf(&b, "- %s\n", c)
	}
	for _, w := range rep.Warnings {
		fmt.Fprintf(&b, "Warning: %s\n", w)
	}
	if rep.Error != "" {
		fmt.Fprintf(&b, "Error: %s\n", rep.Error)
	}
	return b.String()
}
