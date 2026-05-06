package packagehealth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

type Options struct {
	PackageRoot string
	Yes         bool
	DryRun      bool
}

type Check struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Path     string `json:"path,omitempty"`
	Required bool   `json:"required"`
	Message  string `json:"message,omitempty"`
}

type Report struct {
	SchemaVersion int      `json:"schemaVersion"`
	ToolVersion   string   `json:"toolVersion"`
	CreatedAt     string   `json:"createdAt"`
	PackageRoot   string   `json:"packageRoot"`
	AppBundleRoot string   `json:"appBundleRoot,omitempty"`
	Status        string   `json:"status"`
	SafeMode      bool     `json:"safeMode"`
	Checks        []Check  `json:"checks"`
	Actions       []string `json:"actions,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	Error         string   `json:"error,omitempty"`
}

func Doctor(ctx context.Context, runner command.Runner, opts Options) Report {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	root := opts.PackageRoot
	if root == "" {
		root = FindPackageRoot()
	}
	rep := Report{SchemaVersion: 1, ToolVersion: system.Version, CreatedAt: time.Now().Format(time.RFC3339), PackageRoot: root, AppBundleRoot: appBundleRoot(root), Status: "ok"}
	setStatus := func(status string) {
		rep.Status = worseStatus(rep.Status, status)
	}
	add := func(id, path string, required bool, executable bool) {
		check := Check{ID: id, Path: path, Required: required, Status: "ok"}
		info, err := os.Stat(path)
		if err != nil {
			check.Status = "missing"
			check.Message = err.Error()
			if required {
				setStatus("broken")
			} else {
				setStatus("warning")
			}
			rep.Checks = append(rep.Checks, check)
			return
		}
		if executable && info.Mode()&0111 == 0 {
			check.Status = "not_executable"
			check.Message = "executable bit is missing"
			if required {
				setStatus("needs_repair")
			}
		}
		rep.Checks = append(rep.Checks, check)
	}
	add("embedded_agentlink", filepath.Join(root, "bin", "agentlink"), true, true)
	add("rescue_sh", filepath.Join(root, "rescue.sh"), true, true)
	add("agentlink_command", filepath.Join(root, "agentlink.command"), false, true)
	add("recipes_dir", filepath.Join(root, "recipes"), true, false)
	add("rules_dir", filepath.Join(root, "rules"), true, false)
	add("offline_docs", filepath.Join(root, "docs", "offline"), true, false)
	add("asset_manifests", filepath.Join(root, "assets", "manifests"), true, false)
	add("model_assets", filepath.Join(root, "assets", "models"), false, false)
	add("llama_runtime", filepath.Join(root, "assets", "runtimes", "llama.cpp"), false, false)
	add("installer_catalog", filepath.Join(root, "assets", "installers", "catalog.json"), false, false)
	if strings.Contains(root, " ") {
		rep.Warnings = append(rep.Warnings, "package path contains spaces; command quoting is required")
	}
	if runtime.GOOS == "darwin" {
		res := runner.Run(ctx, "/usr/bin/xattr", "-p", "com.apple.quarantine", root)
		if res.ExitCode == 0 {
			rep.Checks = append(rep.Checks, Check{ID: "quarantine_xattr", Status: "needs_repair", Path: root, Message: "com.apple.quarantine present"})
			setStatus("needs_repair")
		} else {
			rep.Checks = append(rep.Checks, Check{ID: "quarantine_xattr", Status: "ok", Path: root})
		}
	}
	for _, dir := range []string{system.UserSupportBundleDir(userHome()), system.UserSessionDir(userHome()), system.UserRestorePointsDir(userHome())} {
		check := Check{ID: "app_support_dir", Status: "ok", Path: dir, Required: true}
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				check.Status = "needs_repair"
				check.Message = "directory missing; package repair can create it"
				setStatus("needs_repair")
			} else {
				check.Status = "failed"
				check.Message = err.Error()
				setStatus("broken")
			}
		} else if !info.IsDir() {
			check.Status = "failed"
			check.Message = "path exists but is not a directory"
			setStatus("broken")
		} else if !modeLooksWritable(info.Mode()) {
			check.Status = "warning"
			check.Message = "directory may not be writable by the current user"
			setStatus("warning")
		}
		rep.Checks = append(rep.Checks, check)
	}
	if tmp := os.TempDir(); tmp == "" {
		rep.Checks = append(rep.Checks, Check{ID: "temp_dir_writable", Status: "failed", Message: "empty temp directory", Required: true})
		setStatus("broken")
	} else if info, err := os.Stat(tmp); err != nil {
		rep.Checks = append(rep.Checks, Check{ID: "temp_dir_writable", Status: "failed", Message: err.Error(), Required: true})
		setStatus("broken")
	} else if !info.IsDir() || !modeLooksWritable(info.Mode()) {
		rep.Checks = append(rep.Checks, Check{ID: "temp_dir_writable", Status: "failed", Path: tmp, Message: "temp directory is not writable by mode", Required: true})
		setStatus("broken")
	} else {
		rep.Checks = append(rep.Checks, Check{ID: "temp_dir_writable", Status: "ok", Path: tmp, Required: true})
	}
	rep.SafeMode = rep.Status == "broken" || rep.Status == "needs_repair"
	_ = ctx
	return rep
}

func Repair(ctx context.Context, runner command.Runner, opts Options) Report {
	rep := Doctor(ctx, runner, opts)
	rep.Actions = nil
	root := rep.PackageRoot
	if opts.DryRun || !opts.Yes {
		rep.Status = "dry_run"
		rep.Actions = plannedActions(root)
		return rep
	}
	for _, path := range []string{filepath.Join(root, "bin", "agentlink"), filepath.Join(root, "rescue.sh"), filepath.Join(root, "agentlink.command")} {
		if _, err := os.Stat(path); err == nil {
			if err := os.Chmod(path, 0755); err != nil {
				rep.Warnings = append(rep.Warnings, "chmod failed for "+path+": "+err.Error())
			} else {
				rep.Actions = append(rep.Actions, "chmod +x "+path)
			}
		}
	}
	for _, dir := range []string{system.UserSupportBundleDir(userHome()), system.UserSessionDir(userHome()), system.UserRestorePointsDir(userHome())} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			rep.Warnings = append(rep.Warnings, "mkdir failed for "+dir+": "+err.Error())
		} else {
			rep.Actions = append(rep.Actions, "ensure dir "+dir)
		}
	}
	if runner != nil && runtime.GOOS == "darwin" {
		res := runner.Run(ctx, "/usr/bin/xattr", "-cr", root)
		if res.ExitCode != 0 {
			rep.Warnings = append(rep.Warnings, "xattr -cr failed: "+strings.TrimSpace(res.Error+" "+res.Stderr))
		} else {
			rep.Actions = append(rep.Actions, "xattr -cr "+root)
		}
	}
	after := Doctor(ctx, runner, opts)
	rep.Checks = after.Checks
	rep.SafeMode = after.SafeMode
	if after.Status == "ok" || after.Status == "warning" {
		rep.Status = "repaired"
	} else if len(rep.Warnings) > 0 {
		rep.Status = "partial_repair"
	} else {
		rep.Status = after.Status
	}
	return rep
}

func Marshal(rep Report) string {
	data, _ := json.MarshalIndent(rep, "", "  ")
	return string(data)
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

func plannedActions(root string) []string {
	return []string{
		"xattr -cr " + root,
		"chmod +x " + filepath.Join(root, "bin", "agentlink"),
		"chmod +x " + filepath.Join(root, "rescue.sh"),
		"chmod +x " + filepath.Join(root, "agentlink.command"),
		"ensure app support directories",
	}
}

func appBundleRoot(root string) string {
	parts := strings.Split(filepath.Clean(root), string(os.PathSeparator))
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasSuffix(parts[i], ".app") {
			return string(os.PathSeparator) + filepath.Join(parts[:i+1]...)
		}
	}
	return ""
}

func userHome() string {
	home, _ := os.UserHomeDir()
	return home
}

func modeLooksWritable(mode os.FileMode) bool {
	return mode&0222 != 0
}

func worseStatus(current, next string) string {
	priority := map[string]int{"ok": 0, "warning": 1, "needs_repair": 2, "broken": 3}
	if priority[next] > priority[current] {
		return next
	}
	return current
}
