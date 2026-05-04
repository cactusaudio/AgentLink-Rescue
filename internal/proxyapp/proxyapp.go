package proxyapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion int      `json:"schemaVersion"`
	ToolVersion   string   `json:"toolVersion"`
	AppID         string   `json:"appID"`
	Action        string   `json:"action"`
	Status        string   `json:"status"`
	DryRun        bool     `json:"dryRun"`
	QuarantineDir string   `json:"quarantineDir,omitempty"`
	Actions       []string `json:"actions"`
	Warnings      []string `json:"warnings"`
	NextAction    string   `json:"nextAction,omitempty"`
	Error         string   `json:"error,omitempty"`
}

func CleanReinstall(ctx context.Context, runner command.Runner, home, appID string, yes bool) Report {
	_ = ctx
	_ = runner
	rep := Report{
		SchemaVersion: 1,
		ToolVersion:   system.Version,
		AppID:         appID,
		Action:        "clean-reinstall",
		DryRun:        !yes,
		Status:        "dry_run",
		Actions: []string{
			"archive Clash/ClashX/Mihomo launch items and helper residue",
			"quarantine user app data instead of permanent deletion",
			"verify network is clean before reopening proxy app",
			"open embedded Clash Verge Rev installer only after TUN repair/restart gate fails",
		},
		Warnings: []string{
			"Final resort only: targeted Clash/TUN repair and restart-gate verification should be attempted first.",
			"AgentLink will not enable Clash proxy or TUN automatically.",
		},
		NextAction: "agentlink rescue --level tun --dry-run --json",
	}
	if appID != "clash-verge-rev" {
		rep.Status = "failed"
		rep.Error = "unsupported proxy app: " + appID
		return rep
	}
	if !yes {
		return rep
	}
	if !system.IsRoot() {
		rep.Status = "manual_action_required"
		rep.NextAction = "sudo ./bin/agentlink proxyapp clean-reinstall clash-verge-rev --yes --json"
		return rep
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	dir := filepath.Join(home, "Library", "Application Support", system.AppName, "proxyapp-quarantine", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	rep.QuarantineDir = dir
	for _, path := range residuePaths(home) {
		if _, err := os.Lstat(path); err != nil {
			continue
		}
		dest := filepath.Join(dir, filepath.Base(path))
		if err := os.Rename(path, dest); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("could not quarantine %s: %v", path, err))
			continue
		}
		rep.Actions = append(rep.Actions, "quarantined "+path)
	}
	rep.Status = "quarantined"
	rep.NextAction = "restart-gate prepare --json, then agentlink installer open clash-verge-rev after network verifies clean"
	return rep
}

func Marshal(rep Report) string {
	data, _ := json.MarshalIndent(rep, "", "  ")
	return string(data)
}

func residuePaths(home string) []string {
	return []string{
		filepath.Join(home, "Library", "LaunchAgents", "io.github.clash-verge-rev.clash-verge-rev.plist"),
		filepath.Join(home, "Library", "Group Containers", "io.github.clash-verge-rev.clash-verge-rev"),
		filepath.Join(home, "Library", "Application Support", "io.github.clash-verge-rev.clash-verge-rev"),
		filepath.Join(home, "Library", "Caches", "io.github.clash-verge-rev.clash-verge-rev"),
		filepath.Join(home, "Library", "Preferences", "io.github.clash-verge-rev.clash-verge-rev.plist"),
		filepath.Join(home, "Library", "Application Support", "com.west2online.ClashX"),
		filepath.Join(home, "Library", "Caches", "com.west2online.ClashX"),
		filepath.Join(home, "Library", "Preferences", "com.west2online.ClashX.plist"),
		"/Library/LaunchDaemons/io.github.clash-verge-rev.clash-verge-rev.service.plist",
		"/Library/LaunchDaemons/com.west2online.ClashX.ProxyConfigHelper.plist",
		"/Library/PrivilegedHelperTools/io.github.clash-verge-rev.clash-verge-rev.service.bundle",
		"/Library/PrivilegedHelperTools/com.west2online.ClashX.ProxyConfigHelper",
	}
}
