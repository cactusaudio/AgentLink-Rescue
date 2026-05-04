package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type Action string

const (
	ActionDoctor  Action = "doctor"
	ActionDryRun  Action = "dry-run"
	ActionInstall Action = "install"
	ActionVerify  Action = "verify"
	ActionOpen    Action = "open"
)

type Options struct {
	Action  Action
	ID      string
	Method  string
	Yes     bool
	Version string
}

func Run(ctx context.Context, runner command.Runner, catalog Catalog, opts Options) (Report, error) {
	in, ok := catalog.Find(opts.ID)
	if !ok {
		return Report{}, fmt.Errorf("unknown installer: %s", opts.ID)
	}
	methodStatuses := methodStatuses(in)
	method, ok := selectMethod(in, opts.Method, methodStatuses)
	if !ok {
		return Report{}, fmt.Errorf("installer %s method not found: %s", opts.ID, opts.Method)
	}
	rep := baseReport(in, method, opts.Version)
	rep.MethodStatuses = methodStatuses
	rep.AssetPath = assetPathFor(in.ID)
	rep.OfficialURL = officialURLFor(method, in)
	if in.ID == "clash-verge-rev" && rep.AssetPath == "" {
		rep.Warnings = append(rep.Warnings, "embedded Clash Verge Rev DMG is missing; run scripts/fetch_clash_verge_rev.sh or use the official releases page")
	}
	rep.Verification = verify(ctx, runner, in)
	installed := anyVerificationPass(rep.Verification)
	switch opts.Action {
	case ActionDoctor:
		if installed {
			rep.Status = "installed"
			rep.NextAction = "No install needed."
		} else if in.ID == "codex-app" {
			rep.Status = "manual_action_required"
			rep.NextAction = "Open the official Codex App page and complete install manually."
		} else if !anyMethodAvailable(methodStatuses) {
			rep.Status = "missing_dependency"
			rep.NextAction = methodDependencyHelp(methodStatuses)
		} else {
			rep.Status = "available"
			rep.NextAction = "Run installer dry-run before installing. Selected method: " + method.ID
		}
	case ActionDryRun:
		rep.Status = "dry_run"
		rep.NextAction = dryRunNextAction(in, method)
	case ActionVerify:
		if installed {
			rep.Status = "installed_verified"
			rep.NextAction = "Installed and verified."
		} else {
			rep.Status = "manual_action_required"
			rep.NextAction = "Install or open the official installer, then run verify again."
		}
	case ActionOpen:
		return runOpen(ctx, runner, in, method, rep)
	case ActionInstall:
		return runInstall(ctx, runner, in, method, rep, opts.Yes)
	default:
		return Report{}, fmt.Errorf("unsupported installer action: %s", opts.Action)
	}
	return rep, nil
}

func Doctor(ctx context.Context, runner command.Runner, catalog Catalog, version string) DoctorReport {
	out := DoctorReport{SchemaVersion: 1, ToolVersion: version}
	for _, in := range catalog.Installers {
		rep, err := Run(ctx, runner, catalog, Options{Action: ActionDoctor, ID: in.ID, Version: version})
		if err != nil {
			out.Warnings = append(out.Warnings, safety.RedactSensitive(err.Error()))
			continue
		}
		out.Reports = append(out.Reports, rep)
	}
	return out
}

func baseReport(in Installer, m Method, version string) Report {
	rep := Report{
		SchemaVersion:   1,
		ToolVersion:     version,
		ID:              in.ID,
		DisplayName:     in.DisplayName,
		Status:          "available",
		Method:          m.ID,
		RequiresNetwork: m.RequiresNetwork,
		RequiresAdmin:   m.RequiresAdmin,
		Warnings:        append([]string(nil), in.Warnings...),
	}
	if len(m.Command) > 0 {
		path, args := resolveCommand(m.Command)
		rep.Commands = append(rep.Commands, CommandPlan{Display: command.Render(path, args...), Path: path, Args: args, Mutates: mutates(m.Type)})
	}
	if in.ID == "codex-app" && m.OfficialURL != "" {
		rep.NextAction = "Open the official Codex App download page."
	}
	return rep
}

func runOpen(ctx context.Context, runner command.Runner, in Installer, m Method, rep Report) (Report, error) {
	if in.ID == "clash-verge-rev" {
		if rep.AssetPath == "" {
			rep.Status = "manual_action_required"
			rep.NextAction = "Embedded DMG missing. Run scripts/fetch_clash_verge_rev.sh or open the official GitHub releases page."
			return rep, nil
		}
		res := run(ctx, runner, "/usr/bin/open", rep.AssetPath)
		rep.RawCommands = append(rep.RawCommands, redactResult(res))
		if command.Success(res) {
			rep.Status = "manual_action_required"
			rep.NextAction = "DMG opened. Drag Clash Verge Rev to Applications manually. AgentLink did not enable proxy or TUN."
			return rep, nil
		}
		rep.Status = "failed"
		rep.NextAction = "Open the DMG manually from Finder."
		return rep, fmt.Errorf("open failed: %s", safety.RedactSensitive(res.Error+res.Stderr))
	}
	if m.OfficialURL != "" {
		res := run(ctx, runner, "/usr/bin/open", m.OfficialURL)
		rep.RawCommands = append(rep.RawCommands, redactResult(res))
		if command.Success(res) {
			rep.Status = "manual_action_required"
			rep.NextAction = "Official page opened. Complete the vendor installer manually, then run verify."
			return rep, nil
		}
		rep.Status = "failed"
		return rep, fmt.Errorf("open official URL failed: %s", safety.RedactSensitive(res.Error+res.Stderr))
	}
	rep.Status = "manual_action_required"
	rep.NextAction = "No open action is available for this installer."
	return rep, nil
}

func runInstall(ctx context.Context, runner command.Runner, in Installer, m Method, rep Report, yes bool) (Report, error) {
	if !yes {
		rep.Status = "manual_action_required"
		rep.NextAction = "installer install requires --yes after dry-run"
		return rep, fmt.Errorf("installer install requires --yes")
	}
	if in.ID == "codex-app" {
		rep.Status = "manual_action_required"
		rep.NextAction = "Codex App install is manual in v0.5.0. Use installer open codex-app."
		return rep, nil
	}
	if in.ID == "clash-verge-rev" {
		rep.Status = "manual_action_required"
		rep.NextAction = "Clash Verge Rev is opened as a DMG. Use installer open clash-verge-rev and drag the app to Applications."
		return rep, nil
	}
	if m.Type == "manual" {
		rep.Status = "manual_action_required"
		rep.NextAction = "Manual method is copy-only. AgentLink will not auto-run install scripts; copy the official command if you choose to proceed."
		return rep, nil
	}
	if len(m.Command) == 0 {
		rep.Status = "manual_action_required"
		rep.NextAction = "No deterministic install command is available."
		return rep, nil
	}
	status := methodStatusByID(rep.MethodStatuses, m.ID)
	if !status.Available {
		rep.Status = "missing_dependency"
		rep.NextAction = "Selected method missing dependencies: " + strings.Join(status.MissingDependencies, ", ")
		return rep, fmt.Errorf("selected method missing dependencies: %s", strings.Join(status.MissingDependencies, ", "))
	}
	path, args := resolveCommand(m.Command)
	if path == "" {
		rep.Status = "missing_dependency"
		rep.NextAction = "Required installer command is missing: " + m.Command[0]
		return rep, fmt.Errorf("missing installer command: %s", m.Command[0])
	}
	res := run(ctx, runner, path, args...)
	rep.RawCommands = append(rep.RawCommands, redactResult(res))
	rep.Verification = verify(ctx, runner, in)
	if command.Success(res) && anyVerificationPass(rep.Verification) {
		rep.Status = "installed_verified"
		rep.NextAction = "Installed and verified."
		return rep, nil
	}
	rep.Status = "failed"
	rep.NextAction = "Install command did not verify. Review redacted output and run installer verify."
	if command.Success(res) {
		return rep, fmt.Errorf("installer verification failed")
	}
	return rep, fmt.Errorf("installer command failed: %s", safety.RedactSensitive(res.Error+res.Stderr))
}

func verify(ctx context.Context, runner command.Runner, in Installer) []VerifyResult {
	var out []VerifyResult
	for i, spec := range in.Verify {
		vr := VerifyResult{ID: fmt.Sprintf("%s-%d", in.ID, i+1), Type: spec.Type, Status: "fail"}
		switch spec.Type {
		case "app_exists":
			if exists(spec.Path) {
				vr.Status = "pass"
				vr.Evidence = spec.Path
			} else {
				vr.Error = "not found: " + spec.Path
			}
		case "command_runs":
			path := commandPath(spec.Command)
			if path == "" {
				vr.Error = "command missing: " + spec.Command
				break
			}
			res := run(ctx, runner, path, spec.Args...)
			if command.Success(res) {
				vr.Status = "pass"
				vr.Evidence = firstLine(safety.RedactSensitive(res.Stdout + res.Stderr))
			} else {
				vr.Error = safety.RedactSensitive(res.Error + res.Stderr)
			}
		default:
			vr.Status = "warn"
			vr.Error = "unknown verifier type"
		}
		out = append(out, vr)
	}
	return out
}

func run(ctx context.Context, runner command.Runner, path string, args ...string) command.Result {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	return runner.Run(ctx, path, args...)
}

func resolveCommand(cmd []string) (string, []string) {
	if len(cmd) == 0 {
		return "", nil
	}
	path := commandPath(cmd[0])
	if path == "" {
		path = cmd[0]
	}
	return path, append([]string(nil), cmd[1:]...)
}

var commandPathLookup = defaultCommandPath

func commandPath(name string) string {
	return commandPathLookup(name)
}

func defaultCommandPath(name string) string {
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		if system.CommandExists(name) {
			return name
		}
		return ""
	}
	if p, err := exec.LookPath(name); err == nil && p != "" {
		return p
	}
	for _, p := range []string{"/opt/homebrew/bin/" + name, "/usr/local/bin/" + name, "/usr/bin/" + name, "/bin/" + name} {
		if system.CommandExists(p) {
			return p
		}
	}
	return ""
}

func anyVerificationPass(results []VerifyResult) bool {
	for _, r := range results {
		if r.Status == "pass" {
			return true
		}
	}
	return false
}

func selectMethod(in Installer, requested string, statuses []MethodStatus) (Method, bool) {
	if requested != "" {
		return in.Method(requested)
	}
	if in.DefaultMethod != "" && methodStatusByID(statuses, in.DefaultMethod).Available {
		return in.Method(in.DefaultMethod)
	}
	for _, status := range statuses {
		if status.Available {
			return in.Method(status.ID)
		}
	}
	return in.Method(in.DefaultMethod)
}

func methodStatuses(in Installer) []MethodStatus {
	var out []MethodStatus
	for _, m := range in.Methods {
		st := MethodStatus{ID: m.ID, Type: m.Type, Available: true}
		if len(m.Command) > 0 {
			path, args := resolveCommand(m.Command)
			st.Command = CommandPlan{Display: command.Render(path, args...), Path: path, Args: args, Mutates: mutates(m.Type)}
		}
		for _, dep := range methodDependencies(in, m) {
			if commandPath(dep) == "" {
				st.Available = false
				st.MissingDependencies = append(st.MissingDependencies, dep)
			}
		}
		out = append(out, st)
	}
	return out
}

func methodDependencies(in Installer, m Method) []string {
	deps := []string{}
	switch m.Type {
	case "npm":
		deps = append(deps, "node", "npm")
	case "homebrew":
		deps = append(deps, "brew")
	case "official_url":
		deps = append(deps, "/usr/bin/open")
	case "embedded_dmg":
		deps = append(deps, "/usr/bin/open")
	}
	for _, dep := range in.Dependencies {
		if dep.Required && !containsExact(deps, dep.Command) {
			if m.Type == "npm" || len(in.Methods) == 1 {
				deps = append(deps, dep.Command)
			}
		}
	}
	return deps
}

func anyMethodAvailable(statuses []MethodStatus) bool {
	for _, st := range statuses {
		if st.Available {
			return true
		}
	}
	return false
}

func methodStatusByID(statuses []MethodStatus, id string) MethodStatus {
	for _, st := range statuses {
		if st.ID == id {
			return st
		}
	}
	return MethodStatus{ID: id}
}

func methodDependencyHelp(statuses []MethodStatus) string {
	var parts []string
	for _, st := range statuses {
		if !st.Available {
			parts = append(parts, st.ID+" missing "+strings.Join(st.MissingDependencies, ", "))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "No supported install method is currently available: " + strings.Join(parts, "; ")
}

func dryRunNextAction(in Installer, m Method) string {
	if in.ID == "clash-verge-rev" {
		if ClashAssetPath() == "" {
			return "Embedded DMG missing. Run scripts/fetch_clash_verge_rev.sh or use the official GitHub releases page."
		}
		return "Dry-run complete. Run installer open clash-verge-rev to open the cached DMG."
	}
	if in.ID == "codex-app" {
		return "Dry-run complete. Run installer open codex-app to open the official OpenAI page."
	}
	if len(m.Command) > 0 {
		return "Dry-run complete. Run installer install " + in.ID + " --yes to execute: " + strings.Join(m.Command, " ")
	}
	return "Dry-run complete. Manual action required."
}

func mutates(methodType string) bool {
	switch methodType {
	case "npm", "homebrew":
		return true
	default:
		return false
	}
}

func assetPathFor(id string) string {
	if id == "clash-verge-rev" {
		return ClashAssetPath()
	}
	return ""
}

func officialURLFor(m Method, in Installer) string {
	if m.OfficialURL != "" {
		return m.OfficialURL
	}
	if len(in.OfficialDocs) > 0 {
		return in.OfficialDocs[0]
	}
	return ""
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func redactResult(res command.Result) command.Result {
	res.Stdout = safety.RedactSensitive(res.Stdout)
	res.Stderr = safety.RedactSensitive(res.Stderr)
	res.Error = safety.RedactSensitive(res.Error)
	return res
}

func CopyCommand(rep Report) string {
	if len(rep.Commands) == 0 {
		return ""
	}
	return rep.Commands[0].Display
}

func AppExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
