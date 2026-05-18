package repair

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/journal"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

const (
	LevelSafe                = "safe"
	LevelStandard            = "standard"
	LevelDeep                = "deep"
	LevelTun                 = "tun"
	LevelCleanBaseline       = "clean-baseline"
	LevelStandardSystemReset = "standard-system-reset"
)

type Options struct {
	Level                         string
	Yes                           bool
	DryRun                        bool
	JSON                          bool
	Verbose                       bool
	Reboot                        bool
	IncludeNetworkExtensionPlists bool
	RulesDir                      string
	Stdin                         io.Reader
	Stdout                        io.Writer
}

type Result struct {
	ToolVersion             string         `json:"toolVersion"`
	Level                   string         `json:"level"`
	DryRun                  bool           `json:"dryRun"`
	Status                  string         `json:"status"`
	ExitCode                int            `json:"exitCode"`
	RestorePointID          string         `json:"restorePointId,omitempty"`
	RestorePointPath        string         `json:"restorePointPath,omitempty"`
	ReportPath              string         `json:"reportPath,omitempty"`
	RollbackCommand         string         `json:"rollbackCommand,omitempty"`
	Preflight               []string       `json:"preflight"`
	Postflight              []string       `json:"postflight,omitempty"`
	Actions                 []ActionResult `json:"actions"`
	FileRollbackStatus      string         `json:"fileRollbackStatus,omitempty"`
	NetworkRollbackStatus   string         `json:"networkRollbackStatus,omitempty"`
	PartialRollbackWarnings []string       `json:"partialRollbackWarnings,omitempty"`
	NetworkRollbackActions  []ActionResult `json:"networkRollbackActions,omitempty"`
	Warnings                []string       `json:"warnings,omitempty"`
	Error                   string         `json:"error,omitempty"`
}

type ActionResult struct {
	ID                   string   `json:"id"`
	Stage                string   `json:"stage,omitempty"`
	Description          string   `json:"description"`
	Command              []string `json:"command,omitempty"`
	DryRun               bool     `json:"dryRun"`
	ExitCode             int      `json:"exitCode,omitempty"`
	Error                string   `json:"error,omitempty"`
	RollbackAvailable    bool     `json:"rollbackAvailable"`
	RouteDeleteAttempted bool     `json:"routeDeleteAttempted,omitempty"`
	RouteAddAttempted    bool     `json:"routeAddAttempted,omitempty"`
	RouteAddGateway      string   `json:"routeAddGateway,omitempty"`
	SkippedReason        string   `json:"skippedReason,omitempty"`
}

type action struct {
	ID                string
	Stage             string
	Description       string
	Path              string
	Args              []string
	Dynamic           string
	Timeout           time.Duration
	IgnoreFailure     bool
	RollbackAvailable bool
	Sleep             time.Duration
}

func Run(ctx context.Context, runner command.Runner, opts Options) Result {
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	if opts.Level == "" {
		opts.Level = LevelSafe
	}
	result := Result{ToolVersion: system.Version, Level: opts.Level, DryRun: opts.DryRun}
	if !validLevel(opts.Level) {
		result.ExitCode = 50
		result.Status = "invalid rescue level"
		result.Error = "level must be safe, tun, standard, clean-baseline, standard-system-reset, or deep"
		return result
	}
	if opts.Level == LevelDeep && !opts.Yes {
		result.ExitCode = 50
		result.Status = "deep rescue refused"
		result.Error = "deep rescue requires --yes"
		return result
	}
	if opts.Level == LevelStandardSystemReset && !opts.Yes {
		result.ExitCode = 50
		result.Status = "standard-system-reset rescue refused"
		result.Error = "standard-system-reset rescue requires --yes"
		return result
	}
	if opts.Level == LevelCleanBaseline && !opts.DryRun && !opts.Yes {
		result.ExitCode = 50
		result.Status = "clean-baseline rescue refused"
		result.Error = "clean-baseline rescue requires --yes"
		return result
	}
	if opts.Level == LevelTun && !opts.DryRun && !opts.Yes {
		result.ExitCode = 50
		result.Status = "tun rescue refused"
		result.Error = "tun rescue requires --yes"
		return result
	}
	if !opts.DryRun && !system.IsRoot() {
		result.ExitCode = 40
		result.Status = "root required"
		result.Error = "rescue requires root; re-run with sudo"
		return result
	}

	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: opts.RulesDir, Verbose: opts.Verbose})
	pre := engine.Run(ctx)
	classify.Apply(&pre)
	result.Preflight = append([]string(nil), pre.Classifications...)
	if protectedTopologyBoundary(pre) {
		result.Warnings = append(result.Warnings, "protected topology repair boundary detected; refusing mutating rescue to preserve protected interfaces/services")
		if opts.DryRun {
			result.ExitCode = 0
			result.Status = "protected_topology_report_only"
			return result
		}
		result.ExitCode = 50
		result.Status = "protected_topology_refused"
		result.Error = "protected topology repair boundary detected; use a support bundle / incident report before any route, DHCP, proxy, TUN, clean-baseline, or VLAN mutation"
		return result
	}
	planned := plannedActions(opts.Level, pre)

	if opts.DryRun {
		if opts.Level == LevelTun {
			for _, a := range buildTunPreActions(pre) {
				result.Actions = append(result.Actions, dryAction(a))
			}
			for _, a := range buildQuarantinePlan(pre, opts.Level) {
				result.Actions = append(result.Actions, a)
			}
			for _, a := range buildTunPostActions(pre) {
				result.Actions = append(result.Actions, dryAction(a))
			}
		} else {
			for _, a := range buildActions(opts.Level, pre) {
				result.Actions = append(result.Actions, dryAction(a))
			}
			for _, a := range buildQuarantinePlan(pre, opts.Level) {
				result.Actions = append(result.Actions, a)
			}
		}
		for _, a := range buildDeepPlan(opts, nil) {
			result.Actions = append(result.Actions, a)
		}
		result.Status = "dry run only; no changes made"
		result.ExitCode = 0
		return result
	}
	jm := journal.New(pre.Host.RealUserHome)
	tx, err := jm.Start(journal.StartOptions{Home: pre.Host.RealUserHome, Target: opts.Level, Recipe: "rescue:" + opts.Level, RequiresAdmin: true})
	if err != nil {
		result.ExitCode = 30
		result.Status = "transaction journal creation failed"
		result.Error = err.Error()
		return result
	}
	_ = jm.WriteJSON(tx, "preflight.json", pre)
	_ = jm.WriteJSON(tx, "planned-actions.json", planned)

	policy := system.MutationOptions{RealUserHome: pre.Host.RealUserHome, IncludeNetworkExtensionPlists: opts.IncludeNetworkExtensionPlists}
	rp, err := snapshot.NewRestorePointWithPolicy(system.RestorePointsDir(), system.Version, policy)
	if err != nil {
		result.ExitCode = 30
		result.Status = "restore point creation failed"
		result.Error = err.Error()
		return result
	}
	result.RestorePointID = rp.Manifest.ID
	result.RestorePointPath = rp.Path
	result.RollbackCommand = CurrentRollbackCommand(rp.Manifest.ID)
	tx.RestorePoint = rp.Manifest.ID
	tx.RollbackAvailable = true
	_ = jm.UpdateState(tx, journal.StateCheckpointed)
	_, _ = rp.WriteJSON("preflight.json", pre)
	if pre.Network.CurrentLocation != "" {
		rp.Manifest.PreviousNetworkLocation = pre.Network.CurrentLocation
		_ = rp.Save()
	}
	networkSnapshot := captureNetworkState(pre)
	_, _ = rp.WriteJSON("network-preflight.json", networkSnapshot)

	failures := 0
	if opts.Level == LevelTun {
		for _, a := range buildTunPreActions(pre) {
			recordJournalMutation(jm, tx, a)
			ar := runAction(ctx, runner, &rp, a, false)
			result.Actions = append(result.Actions, ar)
			if ar.Error != "" && !a.IgnoreFailure {
				failures++
			}
		}
		qFailures := runQuarantine(ctx, runner, &rp, opts, pre, &result, jm, tx)
		failures += qFailures
		for _, a := range buildTunPostActions(pre) {
			recordJournalMutation(jm, tx, a)
			ar := runAction(ctx, runner, &rp, a, false)
			result.Actions = append(result.Actions, ar)
			if ar.Error != "" && !a.IgnoreFailure {
				failures++
			}
		}
	} else {
		for _, a := range buildActions(opts.Level, pre) {
			recordJournalMutation(jm, tx, a)
			ar := runAction(ctx, runner, &rp, a, false)
			result.Actions = append(result.Actions, ar)
			if ar.Error != "" && !a.IgnoreFailure {
				failures++
			}
		}
		qFailures := runQuarantine(ctx, runner, &rp, opts, pre, &result, jm, tx)
		failures += qFailures
	}
	dFailures := runDeep(ctx, &rp, opts, &result, jm, tx)
	failures += dFailures
	if opts.Reboot && opts.Level == LevelDeep {
		a := action{ID: "deep.reboot", Description: "Reboot now", Path: "/sbin/shutdown", Args: []string{"-r", "now"}, Timeout: 5 * time.Second, IgnoreFailure: false}
		recordJournalMutation(jm, tx, a)
		ar := runAction(ctx, runner, &rp, a, false)
		result.Actions = append(result.Actions, ar)
		if ar.Error != "" {
			failures++
		}
	}
	networkMutations := InferNetworkMutationsFromActions(result.Actions)
	_, _ = rp.WriteJSON("network-mutations.json", networkMutations)
	_ = jm.WriteJSON(tx, "mutation-summary.json", networkMutations)
	_ = jm.UpdateState(tx, journal.StateVerifying)

	post := engine.Run(ctx)
	classify.Apply(&post)
	result.Postflight = append([]string(nil), post.Classifications...)
	_, _ = rp.WriteJSON("postflight.json", post)
	_ = jm.WriteJSON(tx, "verifier.json", post)
	worsened := criticalWorsened(pre, post)
	_, _ = rp.WriteJSON("postflight-comparator.json", map[string]any{"criticalWorsened": worsened, "preflightClasses": pre.Classifications, "postflightClasses": post.Classifications})
	result.ReportPath = rp.Manifest.HumanReportPath
	result.ExitCode, result.Status = compare(pre, post, failures)
	if !opts.DryRun && worsened {
		result.Warnings = append(result.Warnings, "postflight was worse than preflight; automatic rollback started")
		rp.SetMutationPolicy(system.MutationOptions{RealUserHome: pre.Host.RealUserHome, IncludeNetworkExtensionPlists: true, ExtraAllowedPaths: manifestPaths(rp.Manifest)})
		if err := rp.RestoreAllWithRunner(ctx, runner); err != nil {
			result.FileRollbackStatus = "failed"
			result.ExitCode = 30
			result.Status = "rollback_failed_after_worsening"
			result.Error = "postflight worsened and rollback failed: " + err.Error()
		} else {
			result.FileRollbackStatus = "restored"
			result.NetworkRollbackStatus = restoreNetworkSnapshotWithMutations(ctx, runner, networkSnapshot, networkMutations, &result)
			_, _ = rp.WriteJSON("network-rollback.json", map[string]any{"status": result.NetworkRollbackStatus, "actions": result.NetworkRollbackActions, "warnings": result.PartialRollbackWarnings})
			_ = jm.WriteJSON(tx, "rollback.json", map[string]any{"fileRollbackStatus": result.FileRollbackStatus, "networkRollbackStatus": result.NetworkRollbackStatus, "actions": result.NetworkRollbackActions, "warnings": result.PartialRollbackWarnings})
			if result.NetworkRollbackStatus == "restored" || result.NetworkRollbackStatus == "skipped" {
				result.ExitCode = 20
				result.Status = "rolled_back_after_worsening"
				_ = jm.UpdateState(tx, journal.StateRolledBack)
			} else {
				result.ExitCode = 30
				result.Status = "partial_rollback_failed"
				_ = jm.UpdateState(tx, journal.StatePartialRollback)
			}
			rollbackDiag := engine.Run(ctx)
			classify.Apply(&rollbackDiag)
			_, _ = rp.WriteJSON("rollback-postflight.json", rollbackDiag)
		}
	}
	if opts.Level == LevelDeep {
		result.Warnings = append(result.Warnings, "reboot is recommended after deep rescue")
	}
	if !worsened {
		if result.ExitCode == 0 {
			_ = jm.UpdateState(tx, journal.StateSucceeded)
		} else {
			_ = jm.UpdateState(tx, journal.StateFailed)
		}
	}
	_ = jm.WriteJSON(tx, "status.json", result)
	_ = rp.WriteHumanReport(HumanResult(result, pre, post))
	_ = rp.Save()
	return result
}

func validLevel(level string) bool {
	return level == LevelSafe || level == LevelTun || level == LevelStandard || level == LevelCleanBaseline || level == LevelStandardSystemReset || level == LevelDeep
}

func plannedActions(level string, pre diagnose.DiagnosticReport) []ActionResult {
	var out []ActionResult
	if protectedTopologyBoundary(pre) {
		return out
	}
	if level == LevelTun {
		for _, a := range buildTunPreActions(pre) {
			out = append(out, dryAction(a))
		}
		out = append(out, buildQuarantinePlan(pre, level)...)
		for _, a := range buildTunPostActions(pre) {
			out = append(out, dryAction(a))
		}
		return out
	}
	for _, a := range buildActions(level, pre) {
		out = append(out, dryAction(a))
	}
	out = append(out, buildQuarantinePlan(pre, level)...)
	if level == LevelDeep {
		out = append(out, buildDeepPlan(Options{Level: level}, nil)...)
	}
	return out
}

func CurrentRollbackCommand(id string) string {
	exe, err := os.Executable()
	if err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(exe); resolveErr == nil {
			exe = resolved
		}
	} else {
		exe = "agentlink"
	}
	return FormatRollbackCommand(exe, id)
}

func FormatRollbackCommand(exe string, id string) string {
	return "sudo " + shellQuote(exe) + " rollback --id " + shellQuote(id)
}

func shellQuote(s string) string {
	if s == "" {
		return `""`
	}
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "$", `\$`)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	return `"` + escaped + `"`
}

func buildActions(level string, pre diagnose.DiagnosticReport) []action {
	if protectedTopologyBoundary(pre) {
		return nil
	}
	if level == LevelTun {
		return buildTunActions(pre)
	}
	if level == LevelCleanBaseline {
		return buildCleanBaselineActions(pre)
	}
	var actions []action
	for _, svc := range pre.Network.Services {
		if strings.TrimSpace(svc.Name) == "" || svc.Disabled {
			continue
		}
		name := svc.Name
		prefix := "safe.service." + sanitizeID(name)
		actions = append(actions,
			action{ID: prefix + ".webproxy", Description: "Disable web proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setwebproxystate", name, "off"}},
			action{ID: prefix + ".securewebproxy", Description: "Disable secure web proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setsecurewebproxystate", name, "off"}},
			action{ID: prefix + ".socks", Description: "Disable SOCKS proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setsocksfirewallproxystate", name, "off"}},
			action{ID: prefix + ".autoproxy", Description: "Disable automatic proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setautoproxystate", name, "off"}},
			action{ID: prefix + ".dns", Description: "Reset DNS servers for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setdnsservers", name, "empty"}},
			action{ID: prefix + ".searchdomains", Description: "Reset search domains for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setsearchdomains", name, "empty"}},
			action{ID: prefix + ".dhcp", Description: "Set DHCP for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setdhcp", name}},
			action{ID: prefix + ".ipv6", Description: "Set IPv6 automatic for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setv6automatic", name}},
		)
	}
	seenDev := map[string]bool{}
	for _, hp := range pre.Network.HardwarePorts {
		if hp.Device == "" || seenDev[hp.Device] {
			continue
		}
		seenDev[hp.Device] = true
		actions = append(actions, action{ID: "safe.ipconfig." + sanitizeID(hp.Device), Description: "Renew DHCP on " + hp.Device, Path: "/usr/sbin/ipconfig", Args: []string{"set", hp.Device, "DHCP"}})
	}
	actions = append(actions,
		action{ID: "safe.dns.flushcache", Description: "Flush DNS cache", Path: "/usr/bin/dscacheutil", Args: []string{"-flushcache"}},
		action{ID: "safe.dns.mdnsresponder", Description: "Signal mDNSResponder", Path: "/usr/bin/killall", Args: []string{"-HUP", "mDNSResponder"}, IgnoreFailure: true},
	)
	if level == LevelSafe {
		return actions
	}
	if level == LevelStandard {
		if wifiService := findWiFiService(pre); wifiService != "" {
			actions = append(actions,
				action{ID: "standard.wifi.off", Description: "Turn Wi-Fi off", Path: "/usr/sbin/networksetup", Args: []string{"-setairportpower", wifiService, "off"}, IgnoreFailure: true},
				action{ID: "standard.wifi.sleep", Description: "Wait before turning Wi-Fi back on", Sleep: 2 * time.Second, IgnoreFailure: true},
				action{ID: "standard.wifi.on", Description: "Turn Wi-Fi on", Path: "/usr/sbin/networksetup", Args: []string{"-setairportpower", wifiService, "on"}, IgnoreFailure: true},
			)
		}
		return actions
	}
	if level != LevelStandardSystemReset && level != LevelDeep {
		return actions
	}
	ts := time.Now().Format("20060102-150405")
	cleanLocation := "AgentLink-Clean-" + ts
	actions = append(actions,
		action{ID: "standard.location.create", Description: "Create clean network location " + cleanLocation, Path: "/usr/sbin/networksetup", Args: []string{"-createlocation", cleanLocation, "populate"}, RollbackAvailable: true},
		action{ID: "standard.location.switch", Description: "Switch to clean network location " + cleanLocation, Path: "/usr/sbin/networksetup", Args: []string{"-switchtolocation", cleanLocation}, RollbackAvailable: true},
		action{ID: "standard.detectnewhardware", Description: "Detect new network hardware", Path: "/usr/sbin/networksetup", Args: []string{"-detectnewhardware"}},
		action{ID: "standard.route.flush.1", Description: "Flush route table", Path: "/sbin/route", Args: []string{"-n", "flush"}, IgnoreFailure: true},
		action{ID: "standard.route.flush.2", Description: "Flush route table again", Path: "/sbin/route", Args: []string{"-n", "flush"}, IgnoreFailure: true},
	)
	for _, hp := range pre.Network.HardwarePorts {
		if hp.Device == "" {
			continue
		}
		actions = append(actions, action{ID: "standard.ipconfig." + sanitizeID(hp.Device), Description: "Renew DHCP on " + hp.Device + " after route flush", Path: "/usr/sbin/ipconfig", Args: []string{"set", hp.Device, "DHCP"}})
	}
	if wifiService := findWiFiService(pre); wifiService != "" {
		actions = append(actions,
			action{ID: "standard.wifi.off", Description: "Turn Wi-Fi off", Path: "/usr/sbin/networksetup", Args: []string{"-setairportpower", wifiService, "off"}, IgnoreFailure: true},
			action{ID: "standard.wifi.sleep", Description: "Wait before turning Wi-Fi back on", Sleep: 2 * time.Second, IgnoreFailure: true},
			action{ID: "standard.wifi.on", Description: "Turn Wi-Fi on", Path: "/usr/sbin/networksetup", Args: []string{"-setairportpower", wifiService, "on"}, IgnoreFailure: true},
		)
	}
	return actions
}

func buildTunActions(pre diagnose.DiagnosticReport) []action {
	actions := append([]action{}, buildTunPreActions(pre)...)
	actions = append(actions, buildTunPostActions(pre)...)
	return actions
}

func protectedTopologyBoundary(report diagnose.DiagnosticReport) bool {
	if !diagnose.RepairCorridorAllowsMutation(report) {
		return true
	}
	for _, class := range report.Classifications {
		if class == classify.ProtectedAudioVLANRouteTrap || class == classify.ProtectedTopologyConstraint {
			return true
		}
	}
	return false
}

func buildCleanBaselineActions(pre diagnose.DiagnosticReport) []action {
	var actions []action
	for _, pattern := range []string{"Clash Verge", "clash-verge", "verge-mihomo", "mihomo", "clash-meta", "clash", "ClashX"} {
		actions = append(actions, action{Stage: "1_stop_interference_runtime", ID: "clean.stop." + sanitizeID(pattern), Description: "Force-stop network interference runtime matching " + pattern, Path: "/usr/bin/pkill", Args: []string{"-9", "-f", pattern}, IgnoreFailure: true})
	}
	actions = append(actions,
		action{Stage: "2_clean_location_baseline", ID: "clean.location.automatic", Description: "Switch to Automatic network location if available", Path: "/usr/sbin/networksetup", Args: []string{"-switchtolocation", "Automatic"}, IgnoreFailure: true, RollbackAvailable: true},
	)
	for _, svc := range pre.Network.Services {
		if strings.TrimSpace(svc.Name) == "" || svc.Disabled {
			continue
		}
		name := svc.Name
		prefix := "clean.service." + sanitizeID(name)
		actions = append(actions,
			action{Stage: "3_clean_service_baseline", ID: prefix + ".webproxy", Description: "Disable web proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setwebproxystate", name, "off"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".securewebproxy", Description: "Disable secure web proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setsecurewebproxystate", name, "off"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".socks", Description: "Disable SOCKS proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setsocksfirewallproxystate", name, "off"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".autoproxy", Description: "Disable automatic proxy for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setautoproxystate", name, "off"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".dns", Description: "Reset DNS servers for " + name + " to DHCP/default", Path: "/usr/sbin/networksetup", Args: []string{"-setdnsservers", name, "empty"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".searchdomains", Description: "Reset search domains for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setsearchdomains", name, "empty"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".dhcp", Description: "Set DHCP for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setdhcp", name}, IgnoreFailure: true},
			action{Stage: "3_clean_service_baseline", ID: prefix + ".ipv6", Description: "Set IPv6 automatic for " + name, Path: "/usr/sbin/networksetup", Args: []string{"-setv6automatic", name}, IgnoreFailure: true},
		)
	}
	wifiDevice := findWiFiDevice(pre)
	if wifiDevice != "" {
		actions = append(actions,
			action{Stage: "4_route_dns_awdl_verify_baseline", ID: "clean.ipconfig.dhcp." + sanitizeID(wifiDevice), Description: "Renew DHCP on active Wi-Fi/default device " + wifiDevice, Path: "/usr/sbin/ipconfig", Args: []string{"set", wifiDevice, "DHCP"}, IgnoreFailure: true},
			action{Stage: "4_route_dns_awdl_verify_baseline", ID: "clean.route.rebuild.safe_dhcp", Description: "Rebuild default route only from current safe DHCP router if missing", Dynamic: "safe-default-route-from-dhcp", Args: []string{wifiDevice}, RollbackAvailable: true},
		)
	}
	actions = append(actions,
		action{Stage: "4_route_dns_awdl_verify_baseline", ID: "clean.dns.flushcache", Description: "Flush DNS cache", Path: "/usr/bin/dscacheutil", Args: []string{"-flushcache"}, IgnoreFailure: true},
		action{Stage: "4_route_dns_awdl_verify_baseline", ID: "clean.dns.mdnsresponder", Description: "Signal mDNSResponder", Path: "/usr/bin/killall", Args: []string{"-HUP", "mDNSResponder"}, IgnoreFailure: true},
		action{Stage: "4_route_dns_awdl_verify_baseline", ID: "clean.awdl.up", Description: "Bring AWDL up for AirDrop discovery", Path: "/sbin/ifconfig", Args: []string{"awdl0", "up"}, IgnoreFailure: true},
		action{Stage: "4_route_dns_awdl_verify_baseline", ID: "clean.sharingd.restart", Description: "Restart sharingd for AirDrop receive discovery", Path: "/usr/bin/killall", Args: []string{"sharingd"}, IgnoreFailure: true},
	)
	return actions
}

func buildTunPreActions(pre diagnose.DiagnosticReport) []action {
	var actions []action
	for _, pattern := range []string{"Clash Verge", "clash-verge", "verge-mihomo", "mihomo", "clash-meta", "clash", "ClashX"} {
		actions = append(actions, action{Stage: "1_stop_runtime_processes", ID: "tun.stop." + sanitizeID(pattern), Description: "Force-stop Clash/Mihomo runtime matching " + pattern, Path: "/usr/bin/pkill", Args: []string{"-9", "-f", pattern}, IgnoreFailure: true})
	}
	uid := realUID()
	for _, item := range eligibleResidues(pre) {
		if item.Kind == "launchDaemon" {
			actions = append(actions, action{Stage: "2_bootout_launch_items", ID: "tun.bootout.system." + sanitizeID(filepath.Base(item.Path)), Description: "Boot out launch daemon " + item.Path, Path: "/bin/launchctl", Args: []string{"bootout", "system", item.Path}, IgnoreFailure: true})
		}
		if item.Kind == "launchAgent" {
			actions = append(actions, action{Stage: "2_bootout_launch_items", ID: "tun.bootout.user." + sanitizeID(filepath.Base(item.Path)), Description: "Boot out launch agent " + item.Path, Path: "/bin/launchctl", Args: []string{"bootout", "gui/" + uid, item.Path}, IgnoreFailure: true})
		}
	}
	return actions
}

func buildTunPostActions(pre diagnose.DiagnosticReport) []action {
	var actions []action
	for _, label := range []string{"system/com.apple.nesessionmanager", "system/com.apple.networkextensiond", "system/com.apple.nehelper"} {
		actions = append(actions, action{Stage: "4_kick_network_extension_daemons", ID: "tun.networkextension.kick." + sanitizeID(label), Description: "Kickstart " + label, Path: "/bin/launchctl", Args: []string{"kickstart", "-k", label}, IgnoreFailure: true})
	}
	for _, iface := range diagnose.DiagnoseTun(pre).UTunInterfaces {
		if !iface.Suspicious {
			continue
		}
		actions = append(actions, action{Stage: "5_down_stale_utun", ID: "tun.ifconfig.down." + sanitizeID(iface.Name), Description: "Down stale TUN interface " + iface.Name, Path: "/sbin/ifconfig", Args: []string{iface.Name, "down"}, IgnoreFailure: true})
	}
	wifiService := findWiFiService(pre)
	wifiDevice := findWiFiDevice(pre)
	if wifiService != "" {
		actions = append(actions,
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.webproxy", Description: "Disable Wi-Fi web proxy", Path: "/usr/sbin/networksetup", Args: []string{"-setwebproxystate", wifiService, "off"}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.securewebproxy", Description: "Disable Wi-Fi secure web proxy", Path: "/usr/sbin/networksetup", Args: []string{"-setsecurewebproxystate", wifiService, "off"}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.socks", Description: "Disable Wi-Fi SOCKS proxy", Path: "/usr/sbin/networksetup", Args: []string{"-setsocksfirewallproxystate", wifiService, "off"}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.autoproxy", Description: "Disable Wi-Fi automatic proxy", Path: "/usr/sbin/networksetup", Args: []string{"-setautoproxystate", wifiService, "off"}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.dns", Description: "Set Wi-Fi DNS to reliable regional/public resolvers", Path: "/usr/sbin/networksetup", Args: []string{"-setdnsservers", wifiService, "223.5.5.5", "119.29.29.29", "8.8.8.8"}, RollbackAvailable: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.searchdomains", Description: "Clear Wi-Fi search domains", Path: "/usr/sbin/networksetup", Args: []string{"-setsearchdomains", wifiService, "empty"}, IgnoreFailure: true, RollbackAvailable: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.dhcp", Description: "Set Wi-Fi DHCP", Path: "/usr/sbin/networksetup", Args: []string{"-setdhcp", wifiService}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.ipv6", Description: "Set Wi-Fi IPv6 automatic", Path: "/usr/sbin/networksetup", Args: []string{"-setv6automatic", wifiService}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.off", Description: "Turn Wi-Fi off", Path: "/usr/sbin/networksetup", Args: []string{"-setairportpower", wifiService, "off"}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.sleep", Description: "Wait before turning Wi-Fi back on", Sleep: 2 * time.Second, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.wifi.on", Description: "Turn Wi-Fi on", Path: "/usr/sbin/networksetup", Args: []string{"-setairportpower", wifiService, "on"}, IgnoreFailure: true},
		)
	}
	if wifiDevice != "" {
		actions = append(actions,
			action{Stage: "6_active_wifi_repair", ID: "tun.ipconfig.dhcp." + sanitizeID(wifiDevice), Description: "Renew DHCP on active Wi-Fi device " + wifiDevice, Path: "/usr/sbin/ipconfig", Args: []string{"set", wifiDevice, "DHCP"}, IgnoreFailure: true},
			action{Stage: "6_active_wifi_repair", ID: "tun.route.rebuild.safe_dhcp", Description: "Rebuild default route only from current safe DHCP router if missing", Dynamic: "safe-default-route-from-dhcp", Args: []string{wifiDevice}, RollbackAvailable: true},
		)
	}
	actions = append(actions,
		action{Stage: "7_dns_airdrop_awdl_repair", ID: "tun.dns.flushcache", Description: "Flush DNS cache", Path: "/usr/bin/dscacheutil", Args: []string{"-flushcache"}, IgnoreFailure: true},
		action{Stage: "7_dns_airdrop_awdl_repair", ID: "tun.dns.mdnsresponder", Description: "Signal mDNSResponder", Path: "/usr/bin/killall", Args: []string{"-HUP", "mDNSResponder"}, IgnoreFailure: true},
		action{Stage: "7_dns_airdrop_awdl_repair", ID: "tun.awdl.up", Description: "Bring AWDL up for AirDrop discovery", Path: "/sbin/ifconfig", Args: []string{"awdl0", "up"}, IgnoreFailure: true},
		action{Stage: "7_dns_airdrop_awdl_repair", ID: "tun.sharingd.restart", Description: "Restart sharingd for AirDrop receive discovery", Path: "/usr/bin/killall", Args: []string{"sharingd"}, IgnoreFailure: true},
	)
	return actions
}

func runAction(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, a action, dry bool) ActionResult {
	ar := dryAction(a)
	ar.DryRun = dry
	if dry {
		return ar
	}
	if a.Dynamic != "" {
		return runDynamicAction(ctx, runner, rp, a)
	}
	if a.Sleep > 0 {
		time.Sleep(a.Sleep)
		ar.ExitCode = 0
		return ar
	}
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = command.DefaultTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res := runner.Run(callCtx, a.Path, a.Args...)
	res.Stdout = safety.RedactSensitive(res.Stdout)
	res.Stderr = safety.RedactSensitive(res.Stderr)
	_ = rp.LogCommand(a.ID, res, a.RollbackAvailable)
	ar.ExitCode = res.ExitCode
	if res.ExitCode != 0 && !a.IgnoreFailure {
		ar.Error = strings.TrimSpace(res.Error + " " + res.Stderr)
		if ar.Error == "" {
			ar.Error = fmt.Sprintf("command exited %d", res.ExitCode)
		}
	}
	return ar
}

func dryAction(a action) ActionResult {
	ar := ActionResult{ID: a.ID, Stage: a.Stage, Description: a.Description, DryRun: true, RollbackAvailable: a.RollbackAvailable}
	if a.Path != "" {
		ar.Command = append([]string{a.Path}, a.Args...)
	}
	if a.Dynamic != "" {
		ar.Command = append([]string{"dynamic", a.Dynamic}, a.Args...)
	}
	if a.Sleep > 0 {
		ar.Command = []string{"sleep", fmt.Sprintf("%.0f", a.Sleep.Seconds())}
	}
	return ar
}

func runDynamicAction(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, a action) ActionResult {
	ar := dryAction(a)
	ar.DryRun = false
	switch a.Dynamic {
	case "safe-default-route-from-dhcp":
		return runSafeDefaultRouteFromDHCP(ctx, runner, rp, a, ar)
	default:
		ar.ExitCode = 1
		ar.Error = "unknown dynamic action: " + a.Dynamic
		return ar
	}
}

func runSafeDefaultRouteFromDHCP(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, a action, ar ActionResult) ActionResult {
	route := runner.Run(ctx, "/sbin/route", "-n", "get", "default")
	_ = rp.LogCommand(a.ID+".check_default_route", route, false)
	routeExists := route.ExitCode == 0 && strings.TrimSpace(route.Stdout) != ""
	routeSuspicious := routeOutputSuspicious(route.Stdout)
	if routeExists && !routeSuspicious {
		ar.ExitCode = 0
		ar.SkippedReason = "default_route_already_safe"
		return ar
	}
	if len(a.Args) == 0 || strings.TrimSpace(a.Args[0]) == "" {
		ar.ExitCode = 0
		ar.SkippedReason = "route_rebuild_skipped_no_wifi_device"
		return ar
	}
	device := a.Args[0]
	router := runner.Run(ctx, "/usr/sbin/ipconfig", "getoption", device, "router")
	_ = rp.LogCommand(a.ID+".dhcp_router", router, false)
	fields := strings.Fields(router.Stdout)
	gateway := ""
	if len(fields) > 0 {
		gateway = strings.TrimSpace(fields[0])
	}
	if router.ExitCode != 0 || !safeDHCPGateway(gateway) {
		ar.ExitCode = 0
		ar.SkippedReason = "route_rebuild_skipped_no_safe_gateway"
		return ar
	}
	ar.RouteAddGateway = gateway
	if routeExists && routeSuspicious {
		ar.RouteDeleteAttempted = true
		del := runner.Run(ctx, "/sbin/route", "delete", "default")
		_ = rp.LogCommand(a.ID+".delete_suspicious_default", del, true)
		if del.ExitCode != 0 {
			ar.ExitCode = del.ExitCode
			ar.Error = strings.TrimSpace(del.Error + " " + del.Stderr)
			if ar.Error == "" {
				ar.Error = fmt.Sprintf("route delete exited %d", del.ExitCode)
			}
			return ar
		}
	}
	ar.RouteAddAttempted = true
	add := runner.Run(ctx, "/sbin/route", "add", "default", gateway)
	_ = rp.LogCommand(a.ID+".add_default", add, true)
	ar.ExitCode = add.ExitCode
	if add.ExitCode != 0 {
		ar.Error = strings.TrimSpace(add.Error + " " + add.Stderr)
		if ar.Error == "" {
			ar.Error = fmt.Sprintf("route add exited %d", add.ExitCode)
		}
	}
	return ar
}

func runQuarantine(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, opts Options, pre diagnose.DiagnosticReport, result *Result, jm journal.Manager, tx *journal.Transaction) int {
	if opts.Level == LevelSafe {
		return 0
	}
	items := eligibleResidues(pre)
	if len(items) == 0 {
		return 0
	}
	if !opts.Yes && !confirm(opts, fmt.Sprintf("Quarantine %d Clash-family residue item(s)?", len(items))) {
		result.Warnings = append(result.Warnings, "known residue quarantine skipped; pass --yes or confirm interactively")
		return 0
	}
	failures := 0
	uid := realUID()
	for _, item := range items {
		if opts.Level != LevelTun && item.Kind == "launchDaemon" {
			a := action{ID: "quarantine.bootout.system." + sanitizeID(filepath.Base(item.Path)), Description: "Boot out launch daemon " + item.Path, Path: "/bin/launchctl", Args: []string{"bootout", "system", item.Path}, IgnoreFailure: true}
			recordJournalMutation(jm, tx, a)
			result.Actions = append(result.Actions, runAction(ctx, runner, rp, a, false))
		}
		if opts.Level != LevelTun && item.Kind == "launchAgent" {
			a := action{ID: "quarantine.bootout.user." + sanitizeID(filepath.Base(item.Path)), Description: "Boot out launch agent " + item.Path, Path: "/bin/launchctl", Args: []string{"bootout", "gui/" + uid, item.Path}, IgnoreFailure: true}
			recordJournalMutation(jm, tx, a)
			result.Actions = append(result.Actions, runAction(ctx, runner, rp, a, false))
		}
		desc := "Quarantine " + item.Path
		ar := ActionResult{ID: "quarantine.path." + sanitizeID(filepath.Base(item.Path)), Stage: "3_quarantine_residue", Description: desc, RollbackAvailable: true}
		recordJournalMutation(jm, tx, action{ID: ar.ID, Stage: ar.Stage, Description: desc, Path: "snapshot.QuarantinePath", Args: []string{item.Path}})
		if _, err := rp.QuarantinePath(item.Path); err != nil {
			ar.Error = err.Error()
			failures++
		} else {
			ar.ExitCode = 0
		}
		result.Actions = append(result.Actions, ar)
	}
	if opts.Level != LevelTun {
		for _, name := range []string{"Clash Verge", "clash-verge", "verge-mihomo", "mihomo", "clash-meta", "clash"} {
			a := action{ID: "quarantine.pkill." + sanitizeID(name), Description: "Stop process " + name, Path: "/usr/bin/pkill", Args: []string{"-x", name}, IgnoreFailure: true}
			recordJournalMutation(jm, tx, a)
			result.Actions = append(result.Actions, runAction(ctx, runner, rp, a, false))
		}
	}
	return failures
}

func runDeep(ctx context.Context, rp *snapshot.RestorePoint, opts Options, result *Result, jm journal.Manager, tx *journal.Transaction) int {
	if opts.Level != LevelDeep {
		return 0
	}
	failures := 0
	for _, path := range deepPlists(opts) {
		if !system.Exists(path) {
			continue
		}
		ar := ActionResult{ID: "deep.quarantine." + sanitizeID(filepath.Base(path)), Description: "Backup and remove " + path, RollbackAvailable: true}
		recordJournalMutation(jm, tx, action{ID: ar.ID, Description: ar.Description, Path: "snapshot.QuarantinePath", Args: []string{path}})
		if _, err := rp.QuarantinePath(path); err != nil {
			ar.Error = err.Error()
			failures++
		}
		result.Actions = append(result.Actions, ar)
	}
	_ = ctx
	return failures
}

func recordJournalMutation(jm journal.Manager, tx *journal.Transaction, a action) {
	if tx == nil || a.ID == "" {
		return
	}
	_ = jm.UpdateState(tx, journal.StateMutating)
	cmd := ""
	if a.Path != "" {
		cmd = command.Render(a.Path, a.Args...)
	} else if a.Dynamic != "" {
		cmd = "dynamic " + a.Dynamic + " " + strings.Join(a.Args, " ")
	}
	_ = jm.AppendMutation(tx, journal.MutationEntry{ActionID: a.ID, Stage: a.Stage, Command: cmd, State: journal.StateMutating})
}

func buildQuarantinePlan(pre diagnose.DiagnosticReport, level string) []ActionResult {
	if level == LevelSafe {
		return nil
	}
	var out []ActionResult
	for _, item := range eligibleResidues(pre) {
		out = append(out, ActionResult{ID: "quarantine.path." + sanitizeID(filepath.Base(item.Path)), Stage: "3_quarantine_residue", Description: "Would quarantine " + item.Path, DryRun: true, RollbackAvailable: true})
	}
	return out
}

func buildDeepPlan(opts Options, _ *snapshot.RestorePoint) []ActionResult {
	if opts.Level != LevelDeep {
		return nil
	}
	var out []ActionResult
	for _, path := range deepPlists(opts) {
		out = append(out, ActionResult{ID: "deep.quarantine." + sanitizeID(filepath.Base(path)), Description: "Would backup and remove " + path, DryRun: true, RollbackAvailable: true})
	}
	return out
}

func shouldUseSystemReset(pre diagnose.DiagnosticReport) bool {
	if !pre.Network.DefaultRoute.Present {
		return true
	}
	for _, class := range pre.Classifications {
		if class == classify.NoActiveInterface || class == classify.NetworkLocationSuspected || class == classify.SysconfigSuspected {
			return true
		}
	}
	return false
}

func eligibleResidues(pre diagnose.DiagnosticReport) []diagnose.ResidueMatch {
	var out []diagnose.ResidueMatch
	add := func(items []diagnose.ResidueMatch) {
		for _, item := range items {
			if item.AutoQuarantineAllowed && strings.Contains(item.RuleID, "clash") {
				out = append(out, item)
			}
		}
	}
	add(pre.Residues.LaunchDaemons)
	add(pre.Residues.LaunchAgents)
	add(pre.Residues.PrivilegedHelpers)
	add(pre.Residues.GroupContainers)
	add(pre.Residues.AppSupport)
	add(pre.Residues.Caches)
	add(pre.Residues.Preferences)
	return out
}

func deepPlists(opts Options) []string {
	paths := []string{
		"/Library/Preferences/SystemConfiguration/preferences.plist",
		"/Library/Preferences/SystemConfiguration/NetworkInterfaces.plist",
		"/Library/Preferences/SystemConfiguration/com.apple.airport.preferences.plist",
		"/Library/Preferences/SystemConfiguration/com.apple.network.identification.plist",
		"/Library/Preferences/SystemConfiguration/com.apple.network.eapolclient.configuration.plist",
		"/Library/Preferences/SystemConfiguration/com.apple.wifi.message-tracer.plist",
	}
	if opts.IncludeNetworkExtensionPlists {
		for _, pattern := range []string{"/Library/Preferences/SystemConfiguration/com.apple.networkextension*.plist", "/Library/Preferences/com.apple.networkextension*.plist"} {
			if matches, err := filepath.Glob(pattern); err == nil {
				paths = append(paths, matches...)
			}
		}
	}
	return paths
}

func findWiFiService(pre diagnose.DiagnosticReport) string {
	for _, hp := range pre.Network.HardwarePorts {
		if strings.Contains(strings.ToLower(hp.Port), "wi-fi") || strings.Contains(strings.ToLower(hp.Port), "airport") {
			if hp.Port != "" {
				return hp.Port
			}
		}
	}
	for _, svc := range pre.Network.Services {
		name := strings.ToLower(svc.Name)
		if strings.Contains(name, "wi-fi") || strings.Contains(name, "wifi") || strings.Contains(name, "airport") {
			return svc.Name
		}
	}
	return ""
}

func findWiFiDevice(pre diagnose.DiagnosticReport) string {
	for _, hp := range pre.Network.HardwarePorts {
		lower := strings.ToLower(hp.Port)
		if hp.Device != "" && (strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wifi") || strings.Contains(lower, "airport")) {
			return hp.Device
		}
	}
	if pre.Network.DefaultRoute.Interface != "" {
		return pre.Network.DefaultRoute.Interface
	}
	return ""
}

func compare(pre, post diagnose.DiagnosticReport, failures int) (int, string) {
	if failures > 0 {
		return 30, "one or more actions failed"
	}
	preCount := criticalCount(pre.Classifications)
	postCount := criticalCount(post.Classifications)
	if len(post.Classifications) == 1 && post.Classifications[0] == classify.OK {
		return 0, "agent link restored"
	}
	if postCount < preCount {
		return 0, "critical connectivity improved"
	}
	if postCount > preCount {
		return 20, "connectivity worsened; rollback recommended"
	}
	return 10, "no improvement detected"
}

func criticalWorsened(pre, post diagnose.DiagnosticReport) bool {
	if pre.Network.DefaultRoute.Present && !post.Network.DefaultRoute.Present {
		return true
	}
	if anyProbeOK(pre.Reachability.RawIPs) && !anyProbeOK(post.Reachability.RawIPs) {
		return true
	}
	if anyProbeOK(pre.Reachability.DNSNames) && !anyProbeOK(post.Reachability.DNSNames) {
		return true
	}
	if anyProbeOK(pre.Reachability.HTTPSTargets) && !anyProbeOK(post.Reachability.HTTPSTargets) {
		return true
	}
	if !pre.Network.ProxySummary.Dirty && post.Network.ProxySummary.Dirty {
		return true
	}
	if !hasStaleTun(pre) && hasStaleTun(post) {
		return true
	}
	if awdlUp(pre) && !awdlUp(post) {
		return true
	}
	return false
}

func hasStaleTun(r diagnose.DiagnosticReport) bool {
	return diagnose.DiagnoseTun(r).RecommendedRepair == "tun"
}

func awdlUp(r diagnose.DiagnosticReport) bool {
	for _, iface := range r.Network.Interfaces {
		if iface.Name == "awdl0" {
			return strings.EqualFold(iface.Status, "active")
		}
	}
	return false
}

type NetworkStateSnapshot struct {
	CurrentLocation       string                `json:"currentLocation,omitempty"`
	WiFiService           string                `json:"wifiService,omitempty"`
	WiFiDevice            string                `json:"wifiDevice,omitempty"`
	DNSServers            []string              `json:"dnsServers,omitempty"`
	SearchDomains         []string              `json:"searchDomains,omitempty"`
	ProxySummary          diagnose.ProxySummary `json:"proxySummary,omitempty"`
	DefaultRouteGateway   string                `json:"defaultRouteGateway,omitempty"`
	DefaultRouteInterface string                `json:"defaultRouteInterface,omitempty"`
	AWDLUp                bool                  `json:"awdlUp,omitempty"`
}

func captureNetworkState(pre diagnose.DiagnosticReport) NetworkStateSnapshot {
	return NetworkStateSnapshot{
		CurrentLocation:       pre.Network.CurrentLocation,
		WiFiService:           findWiFiService(pre),
		WiFiDevice:            findWiFiDevice(pre),
		DNSServers:            append([]string(nil), pre.Network.DNSSummary.Nameservers...),
		SearchDomains:         append([]string(nil), pre.Network.DNSSummary.Search...),
		ProxySummary:          pre.Network.ProxySummary,
		DefaultRouteGateway:   pre.Network.DefaultRoute.Gateway,
		DefaultRouteInterface: pre.Network.DefaultRoute.Interface,
		AWDLUp:                awdlUp(pre),
	}
}

type NetworkMutations struct {
	LocationChanged      bool     `json:"locationChanged,omitempty"`
	DNSChanged           bool     `json:"dnsChanged,omitempty"`
	SearchDomainsChanged bool     `json:"searchDomainsChanged,omitempty"`
	ProxyChanged         bool     `json:"proxyChanged,omitempty"`
	DefaultRouteChanged  bool     `json:"defaultRouteChanged,omitempty"`
	WiFiPowerBounced     bool     `json:"wifiPowerBounced,omitempty"`
	DHCPRenewed          bool     `json:"dhcpRenewed,omitempty"`
	AWDLChanged          bool     `json:"awdlChanged,omitempty"`
	Actions              []string `json:"actions,omitempty"`
}

func (m NetworkMutations) empty() bool {
	return !m.LocationChanged && !m.DNSChanged && !m.SearchDomainsChanged && !m.ProxyChanged && !m.DefaultRouteChanged && !m.WiFiPowerBounced && !m.DHCPRenewed && !m.AWDLChanged
}

func AllNetworkMutations() NetworkMutations {
	return NetworkMutations{LocationChanged: true, DNSChanged: true, SearchDomainsChanged: true, ProxyChanged: true, DefaultRouteChanged: true, WiFiPowerBounced: true, DHCPRenewed: true, AWDLChanged: true}
}

func InferNetworkMutationsFromActions(actions []ActionResult) NetworkMutations {
	var m NetworkMutations
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		m.Actions = append(m.Actions, id)
	}
	for _, action := range actions {
		id := strings.ToLower(action.ID)
		if strings.Contains(id, "location.") {
			m.LocationChanged = true
			add(action.ID)
		}
		if strings.Contains(id, ".webproxy") || strings.Contains(id, ".securewebproxy") || strings.Contains(id, ".socks") || strings.Contains(id, ".autoproxy") {
			m.ProxyChanged = true
			add(action.ID)
		}
		if strings.HasSuffix(id, ".dns") || strings.Contains(id, ".wifi.dns") {
			m.DNSChanged = true
			add(action.ID)
		}
		if strings.Contains(id, "searchdomains") {
			m.SearchDomainsChanged = true
			add(action.ID)
		}
		if strings.Contains(id, "route.rebuild") || action.RouteDeleteAttempted || action.RouteAddAttempted {
			m.DefaultRouteChanged = true
			add(action.ID)
		}
		if strings.Contains(id, ".dhcp") || strings.Contains(id, "ipconfig") {
			m.DHCPRenewed = true
			add(action.ID)
		}
		if strings.Contains(id, "wifi.off") || strings.Contains(id, "wifi.on") {
			m.WiFiPowerBounced = true
			add(action.ID)
		}
		if strings.Contains(id, "awdl") || strings.Contains(id, "sharingd") {
			m.AWDLChanged = true
			add(action.ID)
		}
	}
	return m
}

type NetworkRestoreResult struct {
	Status   string         `json:"status"`
	Actions  []ActionResult `json:"actions,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

func restoreNetworkSnapshot(ctx context.Context, runner command.Runner, snap NetworkStateSnapshot, result *Result) string {
	return restoreNetworkSnapshotWithMutations(ctx, runner, snap, AllNetworkMutations(), result)
}

func restoreNetworkSnapshotWithMutations(ctx context.Context, runner command.Runner, snap NetworkStateSnapshot, mutations NetworkMutations, result *Result) string {
	restore := RestoreNetworkSnapshot(ctx, runner, snap, mutations)
	result.NetworkRollbackActions = append(result.NetworkRollbackActions, restore.Actions...)
	result.PartialRollbackWarnings = append(result.PartialRollbackWarnings, restore.Warnings...)
	return restore.Status
}

func RestoreNetworkSnapshot(ctx context.Context, runner command.Runner, snap NetworkStateSnapshot, mutations NetworkMutations) NetworkRestoreResult {
	restore := NetworkRestoreResult{Status: "restored"}
	if mutations.empty() || (snap.WiFiService == "" && snap.DefaultRouteGateway == "" && !snap.AWDLUp && snap.CurrentLocation == "") {
		restore.Status = "skipped"
		return restore
	}
	if runner == nil {
		runner = command.NewExecRunner()
	}
	failures := 0
	run := func(id, desc, path string, args ...string) {
		res := runner.Run(ctx, path, args...)
		res.Stdout = safety.RedactSensitive(res.Stdout)
		res.Stderr = safety.RedactSensitive(res.Stderr)
		ar := ActionResult{ID: id, Description: desc, Command: append([]string{path}, args...), ExitCode: res.ExitCode}
		if res.ExitCode != 0 {
			ar.Error = strings.TrimSpace(res.Error + " " + res.Stderr)
			if ar.Error == "" {
				ar.Error = fmt.Sprintf("command exited %d", res.ExitCode)
			}
			failures++
		}
		restore.Actions = append(restore.Actions, ar)
	}
	if mutations.LocationChanged && snap.CurrentLocation != "" {
		run("network.rollback.location", "Restore previous network location", "/usr/sbin/networksetup", "-switchtolocation", snap.CurrentLocation)
	}
	if snap.WiFiService != "" && mutations.ProxyChanged {
		if snap.ProxySummary.Dirty {
			restore.Warnings = append(restore.Warnings, "previous proxy state was dirty; exact proxy host/port restoration is not supported, so AgentLink did not re-enable proxy settings")
		} else {
			run("network.rollback.proxy.web.off", "Restore clean Wi-Fi web proxy state", "/usr/sbin/networksetup", "-setwebproxystate", snap.WiFiService, "off")
			run("network.rollback.proxy.secure.off", "Restore clean Wi-Fi secure web proxy state", "/usr/sbin/networksetup", "-setsecurewebproxystate", snap.WiFiService, "off")
			run("network.rollback.proxy.socks.off", "Restore clean Wi-Fi SOCKS proxy state", "/usr/sbin/networksetup", "-setsocksfirewallproxystate", snap.WiFiService, "off")
			run("network.rollback.proxy.auto.off", "Restore clean Wi-Fi automatic proxy state", "/usr/sbin/networksetup", "-setautoproxystate", snap.WiFiService, "off")
		}
	}
	if snap.WiFiService != "" && mutations.DNSChanged {
		dnsArgs := []string{"-setdnsservers", snap.WiFiService}
		if len(snap.DNSServers) == 0 {
			dnsArgs = append(dnsArgs, "empty")
		} else {
			dnsArgs = append(dnsArgs, snap.DNSServers...)
		}
		run("network.rollback.dns", "Restore Wi-Fi DNS servers", "/usr/sbin/networksetup", dnsArgs...)
	}
	if snap.WiFiService != "" && mutations.SearchDomainsChanged {
		searchArgs := []string{"-setsearchdomains", snap.WiFiService}
		if len(snap.SearchDomains) == 0 {
			searchArgs = append(searchArgs, "empty")
		} else {
			searchArgs = append(searchArgs, snap.SearchDomains...)
		}
		run("network.rollback.searchdomains", "Restore Wi-Fi search domains", "/usr/sbin/networksetup", searchArgs...)
	}
	if mutations.DefaultRouteChanged && snap.DefaultRouteGateway != "" {
		if safeRouteGatewayForRestore(snap.DefaultRouteGateway, snap.DefaultRouteInterface) {
			run("network.rollback.route.delete_default", "Delete current default route before restoring previous safe route", "/sbin/route", "delete", "default")
			run("network.rollback.route.add_default", "Restore previous safe default route", "/sbin/route", "add", "default", snap.DefaultRouteGateway)
		} else {
			restore.Warnings = append(restore.Warnings, "previous default route was suspicious and was not restored: "+snap.DefaultRouteGateway+" via "+snap.DefaultRouteInterface)
		}
	}
	if mutations.AWDLChanged && snap.AWDLUp {
		run("network.rollback.awdl.up", "Restore AWDL up state", "/sbin/ifconfig", "awdl0", "up")
	}
	if len(restore.Actions) == 0 && len(restore.Warnings) == 0 {
		restore.Status = "skipped"
		return restore
	}
	if failures > 0 || len(restore.Warnings) > 0 {
		restore.Status = "partial_rollback_failed"
		return restore
	}
	return restore
}

func safeRouteGatewayForRestore(gateway, iface string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(iface)), "utun") {
		return false
	}
	return safeDHCPGateway(gateway)
}

func routeOutputSuspicious(raw string) bool {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "interface: utun") {
		return true
	}
	for _, field := range strings.Fields(raw) {
		field = strings.Trim(field, " ,")
		if !safeDHCPGateway(field) {
			ip := net.ParseIP(field)
			if ip != nil {
				return true
			}
		}
	}
	return false
}

func safeDHCPGateway(gateway string) bool {
	gateway = strings.TrimSpace(gateway)
	if gateway == "" {
		return false
	}
	ip := net.ParseIP(gateway)
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 198 && (v4[1] == 18 || v4[1] == 19) {
			return false
		}
		return true
	}
	lower := strings.ToLower(gateway)
	if strings.HasPrefix(lower, "fdfe:dcba:9876") {
		return false
	}
	return true
}

func anyProbeOK(m map[string]diagnose.ProbeResult) bool {
	for _, p := range m {
		if p.OK {
			return true
		}
	}
	return false
}

func manifestPaths(m snapshot.Manifest) []string {
	var out []string
	for _, entry := range m.Entries {
		if entry.OriginalPath != "" {
			out = append(out, entry.OriginalPath)
		}
	}
	return out
}

func criticalCount(classes []string) int {
	critical := map[string]bool{
		classify.NoActiveInterface: true, classify.LinkLocalOnly: true, classify.NoDHCPLease: true,
		classify.NoDefaultRoute: true, classify.GatewayUnreachable: true, classify.RawIPUnreachable: true,
		classify.DNSFail: true, classify.HTTPSFail: true, classify.KnownAgentResidue: true,
		classify.NetworkExtensionSuspected: true, classify.SysconfigSuspected: true,
	}
	n := 0
	for _, c := range classes {
		if critical[c] {
			n++
		}
	}
	return n
}

func HumanResult(result Result, pre, post diagnose.DiagnosticReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cactus AgentLink Rescue %s\n\n", result.ToolVersion)
	fmt.Fprintf(&b, "Preflight:\n")
	writeClassSummary(&b, pre)
	fmt.Fprintf(&b, "\nActions:\n")
	if len(result.Actions) == 0 {
		fmt.Fprintf(&b, "- No actions selected\n")
	} else {
		for _, a := range result.Actions {
			if a.Error != "" {
				fmt.Fprintf(&b, "- %s: failed (%s)\n", a.Description, a.Error)
			} else {
				fmt.Fprintf(&b, "- %s\n", a.Description)
			}
		}
	}
	fmt.Fprintf(&b, "\nPostflight:\n")
	if len(result.Postflight) == 0 {
		fmt.Fprintf(&b, "- Not run\n")
	} else {
		writeClassSummary(&b, post)
	}
	fmt.Fprintf(&b, "\nStatus:\n%s.\n", sentence(result.Status))
	if result.RestorePointPath != "" {
		fmt.Fprintf(&b, "\nRestore point: %s\n", result.RestorePointPath)
	}
	if result.ReportPath != "" {
		fmt.Fprintf(&b, "Report path: %s\n", result.ReportPath)
	}
	if result.RollbackCommand != "" {
		fmt.Fprintf(&b, "\nRollback:\n%s\n", result.RollbackCommand)
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintf(&b, "\nWarnings:\n")
		for _, w := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}
	return b.String()
}

func writeClassSummary(b *strings.Builder, r diagnose.DiagnosticReport) {
	fmt.Fprintf(b, "- System proxy: %s\n", cleanDirty(r.Network.ProxySummary.Dirty))
	fmt.Fprintf(b, "- Default route: %s\n", okMissing(r.Network.DefaultRoute.Present))
	fmt.Fprintf(b, "- DNS: %s\n", probeMapSummary(r.Reachability.DNSNames))
	fmt.Fprintf(b, "- HTTPS apple/github: %s\n", probeMapSummary(r.Reachability.HTTPSTargets))
	fmt.Fprintf(b, "- Agent endpoints: %s\n", probeMapSummary(r.Reachability.AgentTargets))
	if len(r.Classifications) > 0 {
		fmt.Fprintf(b, "- Failure classes: %s\n", strings.Join(r.Classifications, ", "))
	}
	if r.RecommendedRepairLevel != "" {
		fmt.Fprintf(b, "- Recommended level: %s\n", r.RecommendedRepairLevel)
	}
}

func JSON(result Result) string {
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

func confirm(opts Options, prompt string) bool {
	if opts.Yes {
		return true
	}
	if opts.Stdin == nil || opts.Stdout == nil {
		return false
	}
	if f, ok := opts.Stdin.(*os.File); ok {
		info, err := f.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	fmt.Fprintf(opts.Stdout, "%s [y/N] ", prompt)
	scanner := bufio.NewScanner(opts.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}

func sanitizeID(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "item"
	}
	return out
}

func realUID() string {
	if uid := os.Getenv("SUDO_UID"); uid != "" {
		return uid
	}
	if sudo := os.Getenv("SUDO_USER"); sudo != "" {
		if u, err := user.Lookup(sudo); err == nil {
			return u.Uid
		}
	}
	return fmt.Sprintf("%d", os.Getuid())
}

func cleanDirty(dirty bool) string {
	if dirty {
		return "dirty"
	}
	return "clean"
}

func okMissing(ok bool) string {
	if ok {
		return "OK"
	}
	return "missing"
}

func probeMapSummary(m map[string]diagnose.ProbeResult) string {
	if len(m) == 0 {
		return "not checked"
	}
	ok := 0
	for _, p := range m {
		if p.OK {
			ok++
		}
	}
	if ok == len(m) {
		return "OK"
	}
	if ok == 0 {
		return "failed"
	}
	return fmt.Sprintf("partial (%d/%d)", ok, len(m))
}

func sentence(s string) string {
	if s == "" {
		return "Done"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
