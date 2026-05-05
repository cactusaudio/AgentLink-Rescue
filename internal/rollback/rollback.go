package rollback

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/repair"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

type Options struct {
	ID       string
	Last     bool
	DryRun   bool
	JSON     bool
	RulesDir string
}

type Result struct {
	ToolVersion            string                `json:"toolVersion"`
	DryRun                 bool                  `json:"dryRun"`
	RestorePointID         string                `json:"restorePointId,omitempty"`
	RestorePointPath       string                `json:"restorePointPath,omitempty"`
	Status                 string                `json:"status"`
	ExitCode               int                   `json:"exitCode"`
	Actions                []string              `json:"actions"`
	Warnings               []string              `json:"warnings,omitempty"`
	ReportPath             string                `json:"reportPath,omitempty"`
	FileRollbackStatus     string                `json:"fileRollbackStatus,omitempty"`
	NetworkRollbackStatus  string                `json:"networkRollbackStatus,omitempty"`
	NetworkRollbackActions []repair.ActionResult `json:"networkRollbackActions,omitempty"`
	Error                  string                `json:"error,omitempty"`
}

func Run(ctx context.Context, runner command.Runner, opts Options) Result {
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	result := Result{ToolVersion: system.Version, DryRun: opts.DryRun}
	if opts.ID == "" && !opts.Last {
		result.ExitCode = 50
		result.Status = "invalid rollback arguments"
		result.Error = "use --last or --id RESTORE_POINT_ID"
		return result
	}
	rp, err := loadRestorePoint(opts)
	if err != nil {
		result.ExitCode = 30
		result.Status = "restore point not found"
		result.Error = err.Error()
		return result
	}
	realUser := system.RealConsoleUser(ctx, runner)
	rp.SetMutationPolicy(system.MutationOptions{RealUserHome: realUser.Home, IncludeNetworkExtensionPlists: true})
	result.RestorePointID = rp.Manifest.ID
	result.RestorePointPath = rp.Path
	for _, entry := range rp.Manifest.Entries {
		if entry.QuarantinePath != "" {
			result.Actions = append(result.Actions, "move "+entry.QuarantinePath+" back to "+entry.OriginalPath)
		} else if entry.BackupPath != "" {
			result.Actions = append(result.Actions, "restore backup "+entry.BackupPath+" to "+entry.OriginalPath)
		}
	}
	if rp.Manifest.PreviousNetworkLocation != "" {
		result.Actions = append([]string{"switch network location back to " + rp.Manifest.PreviousNetworkLocation}, result.Actions...)
	}
	networkSnapshot, networkMutations, hasNetworkSnapshot, networkLoadWarnings := loadNetworkRollbackState(rp)
	result.Warnings = append(result.Warnings, networkLoadWarnings...)
	if hasNetworkSnapshot {
		result.Actions = append(result.Actions, "restore network state from "+filepath.Base(rp.Manifest.NetworkSnapshotPath))
	}
	if opts.DryRun {
		result.Status = "dry run only; no changes made"
		result.ExitCode = 0
		return result
	}
	if rp.Manifest.TouchedSystemPaths && !system.IsRoot() {
		result.ExitCode = 40
		result.Status = "root required"
		result.Error = "rollback requires root for this restore point"
		return result
	}
	switchBeforeOK := true
	needsLocationSwitch := rp.Manifest.PreviousNetworkLocation != ""
	restoreFilesFirst := hasSystemConfigurationEntries(rp.Manifest)
	if needsLocationSwitch && !restoreFilesFirst {
		var detail string
		switchBeforeOK, detail = switchLocation(ctx, runner, &rp, "rollback.location.switch.before_restore")
		if !switchBeforeOK {
			result.Warnings = append(result.Warnings, "network location restore failed before file restore: "+detail)
		}
	}
	if err := rp.RestoreAll(); err != nil {
		result.ExitCode = 30
		result.Status = "file rollback failed"
		result.FileRollbackStatus = "failed"
		result.Error = err.Error()
		return result
	}
	result.FileRollbackStatus = "restored"
	if needsLocationSwitch && (restoreFilesFirst || !switchBeforeOK) {
		ok, detail := switchLocation(ctx, runner, &rp, "rollback.location.switch.after_restore")
		if !ok {
			result.ExitCode = 30
			result.Status = "files restored; network location restore failed"
			result.NetworkRollbackStatus = "partial_rollback_failed"
			result.Error = detail
			result.Warnings = append(result.Warnings, "files were restored, but network location restore failed: "+detail)
			return result
		}
		if !switchBeforeOK {
			result.Warnings = append(result.Warnings, "network location restore succeeded after file restore retry")
		}
	}
	if hasNetworkSnapshot {
		networkRestore := repair.RestoreNetworkSnapshot(ctx, runner, networkSnapshot, networkMutations)
		result.NetworkRollbackStatus = networkRestore.Status
		result.NetworkRollbackActions = append(result.NetworkRollbackActions, networkRestore.Actions...)
		result.Warnings = append(result.Warnings, networkRestore.Warnings...)
		if networkRestore.Status == "partial_rollback_failed" {
			result.Status = "rollback complete with warnings"
			result.ExitCode = 20
		}
	} else {
		result.NetworkRollbackStatus = "skipped"
	}
	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: opts.RulesDir})
	post := engine.Run(ctx)
	classify.Apply(&post)
	data, _ := json.MarshalIndent(post, "", "  ")
	reportPath := filepath.Join(rp.Path, "rollback-diagnostic.json")
	_ = os.WriteFile(reportPath, data, 0644)
	result.ReportPath = reportPath
	if result.Status == "" {
		result.Status = "rollback complete"
		result.ExitCode = 0
	}
	return result
}

func loadNetworkRollbackState(rp snapshot.RestorePoint) (repair.NetworkStateSnapshot, repair.NetworkMutations, bool, []string) {
	var warnings []string
	if rp.Manifest.NetworkSnapshotPath == "" {
		return repair.NetworkStateSnapshot{}, repair.NetworkMutations{}, false, nil
	}
	snapshotPath := rp.Manifest.NetworkSnapshotPath
	if !filepath.IsAbs(snapshotPath) {
		snapshotPath = filepath.Join(rp.Path, snapshotPath)
	}
	var snap repair.NetworkStateSnapshot
	if err := readJSONInsideRestorePoint(rp.Path, snapshotPath, &snap); err != nil {
		return repair.NetworkStateSnapshot{}, repair.NetworkMutations{}, false, []string{"network rollback snapshot could not be loaded: " + err.Error()}
	}
	mutations := repair.AllNetworkMutations()
	if rp.Manifest.NetworkMutationsPath != "" {
		mutationPath := rp.Manifest.NetworkMutationsPath
		if !filepath.IsAbs(mutationPath) {
			mutationPath = filepath.Join(rp.Path, mutationPath)
		}
		var loadedMutations repair.NetworkMutations
		if err := readJSONInsideRestorePoint(rp.Path, mutationPath, &loadedMutations); err != nil {
			warnings = append(warnings, "network rollback mutations could not be loaded; restoring all captured network fields: "+err.Error())
			mutations = repair.AllNetworkMutations()
		} else {
			mutations = loadedMutations
		}
	}
	return snap, mutations, true, warnings
}

func readJSONInsideRestorePoint(root string, path string, v any) error {
	rootClean := filepath.Clean(root)
	pathClean := filepath.Clean(path)
	if pathClean != rootClean && !strings.HasPrefix(pathClean, rootClean+string(os.PathSeparator)) {
		return fmt.Errorf("path outside restore point refused: %s", path)
	}
	data, err := os.ReadFile(pathClean)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func loadRestorePoint(opts Options) (snapshot.RestorePoint, error) {
	if opts.Last {
		return snapshot.Latest(system.RestorePointsDir())
	}
	id := filepath.Base(opts.ID)
	return snapshot.Load(filepath.Join(system.RestorePointsDir(), id))
}

func switchLocation(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, actionID string) (bool, string) {
	res := runner.Run(ctx, "/usr/sbin/networksetup", "-switchtolocation", rp.Manifest.PreviousNetworkLocation)
	res.Stdout = safety.RedactSensitive(res.Stdout)
	res.Stderr = safety.RedactSensitive(res.Stderr)
	_ = rp.LogCommand(actionID, res, false)
	if res.ExitCode == 0 {
		return true, ""
	}
	detail := strings.TrimSpace(res.Error + " " + res.Stderr)
	if detail == "" {
		detail = fmt.Sprintf("networksetup exited %d", res.ExitCode)
	}
	return false, detail
}

func hasSystemConfigurationEntries(m snapshot.Manifest) bool {
	for _, entry := range m.Entries {
		if strings.HasPrefix(entry.OriginalPath, "/Library/Preferences/SystemConfiguration/") {
			return true
		}
	}
	return false
}

func Human(result Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cactus AgentLink Rescue %s\n\n", result.ToolVersion)
	fmt.Fprintf(&b, "Rollback target: %s\n", result.RestorePointPath)
	fmt.Fprintf(&b, "Status: %s\n", result.Status)
	if len(result.Actions) > 0 {
		fmt.Fprintf(&b, "\nActions:\n")
		for _, a := range result.Actions {
			fmt.Fprintf(&b, "- %s\n", a)
		}
	}
	if result.ReportPath != "" {
		fmt.Fprintf(&b, "\nReport path: %s\n", result.ReportPath)
	}
	if result.Error != "" {
		fmt.Fprintf(&b, "\nError: %s\n", result.Error)
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintf(&b, "\nWarnings:\n")
		for _, w := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}
	return b.String()
}

func JSON(result Result) string {
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}
