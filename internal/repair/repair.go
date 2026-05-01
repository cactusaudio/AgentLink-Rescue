package repair

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

const (
	LevelSafe     = "safe"
	LevelStandard = "standard"
	LevelDeep     = "deep"
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
	ToolVersion      string         `json:"toolVersion"`
	Level            string         `json:"level"`
	DryRun           bool           `json:"dryRun"`
	Status           string         `json:"status"`
	ExitCode         int            `json:"exitCode"`
	RestorePointID   string         `json:"restorePointId,omitempty"`
	RestorePointPath string         `json:"restorePointPath,omitempty"`
	ReportPath       string         `json:"reportPath,omitempty"`
	RollbackCommand  string         `json:"rollbackCommand,omitempty"`
	Preflight        []string       `json:"preflight"`
	Postflight       []string       `json:"postflight,omitempty"`
	Actions          []ActionResult `json:"actions"`
	Warnings         []string       `json:"warnings,omitempty"`
	Error            string         `json:"error,omitempty"`
}

type ActionResult struct {
	ID                string   `json:"id"`
	Description       string   `json:"description"`
	Command           []string `json:"command,omitempty"`
	DryRun            bool     `json:"dryRun"`
	ExitCode          int      `json:"exitCode,omitempty"`
	Error             string   `json:"error,omitempty"`
	RollbackAvailable bool     `json:"rollbackAvailable"`
}

type action struct {
	ID                string
	Description       string
	Path              string
	Args              []string
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
		result.Error = "level must be safe, standard, or deep"
		return result
	}
	if opts.Level == LevelDeep && !opts.Yes {
		result.ExitCode = 50
		result.Status = "deep rescue refused"
		result.Error = "deep rescue requires --yes"
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

	planned := buildActions(opts.Level, pre)
	if opts.DryRun {
		for _, a := range planned {
			result.Actions = append(result.Actions, dryAction(a))
		}
		for _, a := range buildQuarantinePlan(pre, opts.Level) {
			result.Actions = append(result.Actions, a)
		}
		for _, a := range buildDeepPlan(opts, nil) {
			result.Actions = append(result.Actions, a)
		}
		result.Status = "dry run only; no changes made"
		result.ExitCode = 0
		return result
	}

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
	_, _ = rp.WriteJSON("preflight.json", pre)
	if pre.Network.CurrentLocation != "" {
		rp.Manifest.PreviousNetworkLocation = pre.Network.CurrentLocation
		_ = rp.Save()
	}

	failures := 0
	for _, a := range planned {
		ar := runAction(ctx, runner, &rp, a, false)
		result.Actions = append(result.Actions, ar)
		if ar.Error != "" && !a.IgnoreFailure {
			failures++
		}
	}
	qFailures := runQuarantine(ctx, runner, &rp, opts, pre, &result)
	failures += qFailures
	dFailures := runDeep(ctx, &rp, opts, &result)
	failures += dFailures
	if opts.Reboot && opts.Level == LevelDeep {
		a := action{ID: "deep.reboot", Description: "Reboot now", Path: "/sbin/shutdown", Args: []string{"-r", "now"}, Timeout: 5 * time.Second, IgnoreFailure: false}
		ar := runAction(ctx, runner, &rp, a, false)
		result.Actions = append(result.Actions, ar)
		if ar.Error != "" {
			failures++
		}
	}

	post := engine.Run(ctx)
	classify.Apply(&post)
	result.Postflight = append([]string(nil), post.Classifications...)
	_, _ = rp.WriteJSON("postflight.json", post)
	result.ReportPath = rp.Manifest.HumanReportPath
	result.ExitCode, result.Status = compare(pre, post, failures)
	if opts.Level == LevelDeep {
		result.Warnings = append(result.Warnings, "reboot is recommended after deep rescue")
	}
	_ = rp.WriteHumanReport(HumanResult(result, pre, post))
	_ = rp.Save()
	return result
}

func validLevel(level string) bool {
	return level == LevelSafe || level == LevelStandard || level == LevelDeep
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

func runAction(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, a action, dry bool) ActionResult {
	ar := dryAction(a)
	ar.DryRun = dry
	if dry {
		return ar
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
	ar := ActionResult{ID: a.ID, Description: a.Description, DryRun: true, RollbackAvailable: a.RollbackAvailable}
	if a.Path != "" {
		ar.Command = append([]string{a.Path}, a.Args...)
	}
	if a.Sleep > 0 {
		ar.Command = []string{"sleep", fmt.Sprintf("%.0f", a.Sleep.Seconds())}
	}
	return ar
}

func runQuarantine(ctx context.Context, runner command.Runner, rp *snapshot.RestorePoint, opts Options, pre diagnose.DiagnosticReport, result *Result) int {
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
		if item.Kind == "launchDaemon" {
			a := action{ID: "quarantine.bootout.system." + sanitizeID(filepath.Base(item.Path)), Description: "Boot out launch daemon " + item.Path, Path: "/bin/launchctl", Args: []string{"bootout", "system", item.Path}, IgnoreFailure: true}
			result.Actions = append(result.Actions, runAction(ctx, runner, rp, a, false))
		}
		if item.Kind == "launchAgent" {
			a := action{ID: "quarantine.bootout.user." + sanitizeID(filepath.Base(item.Path)), Description: "Boot out launch agent " + item.Path, Path: "/bin/launchctl", Args: []string{"bootout", "gui/" + uid, item.Path}, IgnoreFailure: true}
			result.Actions = append(result.Actions, runAction(ctx, runner, rp, a, false))
		}
		desc := "Quarantine " + item.Path
		ar := ActionResult{ID: "quarantine.path." + sanitizeID(filepath.Base(item.Path)), Description: desc, RollbackAvailable: true}
		if _, err := rp.QuarantinePath(item.Path); err != nil {
			ar.Error = err.Error()
			failures++
		} else {
			ar.ExitCode = 0
		}
		result.Actions = append(result.Actions, ar)
	}
	for _, name := range []string{"Clash Verge", "clash-verge", "verge-mihomo", "mihomo", "clash-meta", "clash"} {
		a := action{ID: "quarantine.pkill." + sanitizeID(name), Description: "Stop process " + name, Path: "/usr/bin/pkill", Args: []string{"-x", name}, IgnoreFailure: true}
		result.Actions = append(result.Actions, runAction(ctx, runner, rp, a, false))
	}
	return failures
}

func runDeep(ctx context.Context, rp *snapshot.RestorePoint, opts Options, result *Result) int {
	if opts.Level != LevelDeep {
		return 0
	}
	failures := 0
	for _, path := range deepPlists(opts) {
		if !system.Exists(path) {
			continue
		}
		ar := ActionResult{ID: "deep.quarantine." + sanitizeID(filepath.Base(path)), Description: "Backup and remove " + path, RollbackAvailable: true}
		if _, err := rp.QuarantinePath(path); err != nil {
			ar.Error = err.Error()
			failures++
		}
		result.Actions = append(result.Actions, ar)
	}
	_ = ctx
	return failures
}

func buildQuarantinePlan(pre diagnose.DiagnosticReport, level string) []ActionResult {
	if level == LevelSafe {
		return nil
	}
	var out []ActionResult
	for _, item := range eligibleResidues(pre) {
		out = append(out, ActionResult{ID: "quarantine.path." + sanitizeID(filepath.Base(item.Path)), Description: "Would quarantine " + item.Path, DryRun: true, RollbackAvailable: true})
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
