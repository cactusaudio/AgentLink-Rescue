package ticket

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

type Options struct {
	Home        string
	Type        string
	ID          string
	Version     string
	PackageRoot string
}

type Report struct {
	SchemaVersion int               `json:"schemaVersion"`
	ToolVersion   string            `json:"toolVersion"`
	Type          string            `json:"type"`
	ID            string            `json:"id"`
	Directory     string            `json:"directory"`
	Files         map[string]string `json:"files"`
	Status        string            `json:"status"`
	Warnings      []string          `json:"warnings,omitempty"`
	Error         string            `json:"error,omitempty"`
}

func Create(ctx context.Context, runner command.Runner, opts Options) Report {
	_ = ctx
	_ = runner
	if opts.Version == "" {
		opts.Version = system.Version
	}
	if opts.Home == "" {
		opts.Home, _ = os.UserHomeDir()
	}
	id := time.Now().Format("20060102-150405")
	root := filepath.Join(opts.Home, "Library", "Application Support", system.AppName, "tickets", id)
	report := Report{SchemaVersion: 1, ToolVersion: opts.Version, Type: opts.Type, ID: id, Directory: root, Files: map[string]string{}, Status: "created"}
	if err := os.MkdirAll(root, 0755); err != nil {
		report.Status = "failed"
		report.Error = err.Error()
		return report
	}
	packageRoot := opts.PackageRoot
	if packageRoot == "" {
		packageRoot = FindPackageRoot()
	}
	switch opts.Type {
	case "clash-tun-fix":
		report.write(root, "Run-Clash-TUN-Fix.command", commandScript(packageRoot, []string{"sudo ./bin/agentlink rescue --level tun --yes", "./bin/agentlink verify network --json"}, "Clash/TUN repair finished. Review the output above."))
		report.write(root, "Verify-Network.command", commandScript(packageRoot, []string{"./bin/agentlink verify network --json"}, "Network verification finished."))
	case "rollback":
		if opts.ID == "" {
			report.Status = "failed"
			report.Error = "rollback ticket requires --id"
			return report
		}
		report.write(root, "Rollback-"+filepath.Base(opts.ID)+".command", commandScript(packageRoot, []string{"sudo ./bin/agentlink rollback --id " + shellQuote(filepath.Base(opts.ID)), "./bin/agentlink verify network --json"}, "Rollback finished."))
	case "verify-network":
		report.write(root, "Verify-Network.command", commandScript(packageRoot, []string{"./bin/agentlink verify network --json"}, "Network verification finished."))
	default:
		report.Status = "failed"
		report.Error = "unsupported ticket type"
		return report
	}
	readme := "Cactus AgentLink Rescue Terminal Repair Ticket\n\nNo password is stored in these files. macOS Terminal may ask for your sudo password.\nRun the .command file, then return to AgentLink and click Verify.\n"
	report.write(root, "README.txt", readme)
	data, _ := json.MarshalIndent(report, "", "  ")
	_ = os.WriteFile(filepath.Join(root, "ticket.json"), data, 0644)
	report.Files["ticket"] = filepath.Join(root, "ticket.json")
	return report
}

func (r *Report) write(root, name, content string) {
	path := filepath.Join(root, name)
	mode := os.FileMode(0644)
	if strings.HasSuffix(name, ".command") {
		mode = 0755
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		r.Status = "failed"
		r.Error = err.Error()
		return
	}
	r.Files[name] = path
}

func commandScript(packageRoot string, commands []string, done string) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -e\n")
	b.WriteString("cd " + shellQuote(packageRoot) + "\n")
	b.WriteString("xattr -cr . 2>/dev/null || true\n")
	b.WriteString("chmod +x ./bin/agentlink ./rescue.sh 2>/dev/null || true\n")
	for _, cmd := range commands {
		b.WriteString(cmd + "\n")
	}
	b.WriteString("echo\n")
	b.WriteString("echo " + shellQuote(done) + "\n")
	b.WriteString("echo 'Press Return to close this window.'\n")
	b.WriteString("read _\n")
	return b.String()
}

func FindPackageRoot() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if filepath.Base(dir) == "bin" {
			return filepath.Dir(dir)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func Human(report Report) string {
	if report.Status == "failed" {
		return fmt.Sprintf("Ticket failed: %s\n", report.Error)
	}
	return fmt.Sprintf("Ticket created: %s\n", report.Directory)
}
