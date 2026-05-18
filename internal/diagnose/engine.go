package diagnose

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/rules"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type Options struct {
	RulesDir string
	Verbose  bool
}

type Engine struct {
	Runner  command.Runner
	Options Options
}

func NewEngine(runner command.Runner, opts Options) Engine {
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	return Engine{Runner: runner, Options: opts}
}

func (e Engine) Run(ctx context.Context) DiagnosticReport {
	now := time.Now().Format(time.RFC3339)
	userInfo := system.RealConsoleUser(ctx, e.Runner)
	report := DiagnosticReport{
		SchemaVersion: 1,
		ToolVersion:   system.Version,
		CreatedAt:     now,
		Host: HostInfo{
			Arch:         runtime.GOARCH,
			RealUser:     userInfo.Name,
			RealUserHome: userInfo.Home,
			RealUserUID:  userInfo.UID,
			RealUserGID:  userInfo.GID,
			EUID:         os.Geteuid(),
		},
		Reachability: ReachabilityInfo{
			RawIPs:       map[string]ProbeResult{},
			DNSNames:     map[string]ProbeResult{},
			HTTPSTargets: map[string]ProbeResult{},
			AgentTargets: map[string]ProbeResult{},
		},
		UserConfig: UserConfigInfo{
			EnvProxy:  map[string]string{},
			GitProxy:  map[string]string{},
			NpmProxy:  map[string]string{},
			BrewProxy: map[string]string{},
		},
		Warnings: []string{},
	}
	if e.Options.Verbose {
		report.Raw = map[string]string{}
	}

	report.Host.MacOSVersion = strings.TrimSpace(e.runText(ctx, "/usr/bin/sw_vers", "-productVersion"))
	report.Host.BuildVersion = strings.TrimSpace(e.runText(ctx, "/usr/bin/sw_vers", "-buildVersion"))
	if arch := strings.TrimSpace(e.runText(ctx, "/usr/bin/uname", "-m")); arch != "" {
		report.Host.Arch = arch
	}
	if hostname := strings.TrimSpace(e.runText(ctx, "/bin/hostname")); hostname != "" {
		report.Host.Hostname = hostname
	} else if name, err := os.Hostname(); err == nil {
		report.Host.Hostname = name
	}

	report.Network.CurrentLocation = strings.TrimSpace(e.runText(ctx, "/usr/sbin/networksetup", "-getcurrentlocation"))
	servicesRaw := e.runText(ctx, "/usr/sbin/networksetup", "-listallnetworkservices")
	report.Network.Services = ParseNetworkServices(servicesRaw)
	e.recordRaw(&report, "networksetup_services", servicesRaw)
	portsRaw := e.runText(ctx, "/usr/sbin/networksetup", "-listallhardwareports")
	report.Network.HardwarePorts = ParseHardwarePorts(portsRaw)
	e.recordRaw(&report, "networksetup_hardware_ports", portsRaw)
	ifconfigRaw := e.runText(ctx, "/sbin/ifconfig")
	report.Network.Interfaces = ParseIfconfig(ifconfigRaw)
	e.recordRaw(&report, "ifconfig", ifconfigRaw)
	routeRaw := e.runText(ctx, "/sbin/route", "-n", "get", "default")
	netstatRaw := e.runText(ctx, "/usr/bin/netstat", "-rn")
	report.Network.DefaultRoute = ParseRouteDefault(routeRaw, netstatRaw)
	report.Network.DefaultRoute.Raw = safety.RedactSensitive(report.Network.DefaultRoute.Raw)
	e.recordRaw(&report, "route_default", routeRaw)
	e.recordRaw(&report, "netstat_rn", netstatRaw)

	report.Network.DHCP = e.collectDHCP(ctx, report.Network.HardwarePorts)
	proxyRaw := e.runText(ctx, "/usr/sbin/scutil", "--proxy")
	report.Network.ProxySummary = ParseScutilProxy(safety.RedactSensitive(proxyRaw))
	e.recordRaw(&report, "scutil_proxy", proxyRaw)
	report.UserConfig.EnvProxy = collectEnvProxy()
	report.UserConfig.GitProxy = e.collectGitProxy(ctx)
	report.UserConfig.NpmProxy = e.collectNpmProxy(ctx)
	report.UserConfig.BrewProxy = e.collectBrewProxy(ctx)

	endpoints := e.loadEndpoints()
	reachCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	report.Reachability.RawIPs = e.probeRawIPs(reachCtx, endpoints.RawIPs)
	report.Reachability.DNSNames = e.probeDNS(reachCtx, endpoints.DNSNames)
	dnsRaw := e.runText(ctx, "/usr/sbin/scutil", "--dns")
	report.Network.DNSSummary = ParseDNSSummary(safety.RedactSensitive(dnsRaw), report.Reachability.DNSNames)
	e.recordRaw(&report, "scutil_dns", dnsRaw)
	if report.Network.DefaultRoute.Gateway != "" {
		report.Reachability.Gateway = e.probePing(reachCtx, report.Network.DefaultRoute.Gateway)
	}
	report.Reachability.HTTPSTargets = e.probeHTTPS(reachCtx, endpoints.HTTPSTargets)
	report.Reachability.AgentTargets = e.probeHTTPS(reachCtx, endpoints.AgentTargets)

	agentRules := e.loadAgentRules(&report)
	report.Residues = e.collectResidues(ctx, agentRules, userInfo.Home)
	e.collectSystemExtensions(ctx, agentRules, &report)
	e.collectProfiles(ctx, &report)
	e.collectTopology(ctx, &report, userInfo.Home)
	AnalyzeTopology(&report)

	return report
}

func (e Engine) runText(ctx context.Context, path string, args ...string) string {
	res := e.runWithTimeout(ctx, 10*time.Second, path, args...)
	if res.Missing || res.TimedOut {
		return ""
	}
	return safety.RedactSensitive(res.Stdout)
}

func (e Engine) runWithTimeout(ctx context.Context, timeout time.Duration, path string, args ...string) command.Result {
	if timeout <= 0 {
		timeout = command.DefaultTimeout
	}
	trace("run " + command.Render(path, args...))
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	res := e.Runner.Run(callCtx, path, args...)
	res.Stdout = safety.RedactSensitive(res.Stdout)
	res.Stderr = safety.RedactSensitive(res.Stderr)
	trace(fmt.Sprintf("done %s exit=%d timeout=%v missing=%v", command.Render(path, args...), res.ExitCode, res.TimedOut, res.Missing))
	return res
}

func (e Engine) recordRaw(report *DiagnosticReport, key, value string) {
	if report.Raw != nil && value != "" {
		report.Raw[key] = safety.RedactSensitive(value)
	}
}

func (e Engine) collectDHCP(ctx context.Context, ports []HardwarePort) []DHCPInfo {
	var out []DHCPInfo
	seen := map[string]bool{}
	for _, port := range ports {
		dev := strings.TrimSpace(port.Device)
		if dev == "" || seen[dev] {
			continue
		}
		seen[dev] = true
		info := DHCPInfo{Device: dev}
		addr := strings.TrimSpace(e.runText(ctx, "/usr/sbin/ipconfig", "getifaddr", dev))
		if addr != "" {
			info.IPv4 = strings.Fields(addr)[0]
			info.HasLease = !IsLinkLocalIPv4(info.IPv4)
			info.LinkLocalOnly = IsLinkLocalIPv4(info.IPv4)
		}
		packet := e.runText(ctx, "/usr/sbin/ipconfig", "getpacket", dev)
		info.PacketSummary = summarizePacket(packet)
		if info.PacketSummary != "" && info.IPv4 == "" {
			info.HasLease = true
		}
		out = append(out, info)
	}
	return out
}

func summarizePacket(packet string) string {
	if strings.TrimSpace(packet) == "" {
		return ""
	}
	var parts []string
	for _, raw := range strings.Split(packet, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "yiaddr =") || strings.HasPrefix(line, "server_identifier") || strings.HasPrefix(line, "lease_time") || strings.HasPrefix(line, "router") {
			parts = append(parts, line)
		}
		if len(parts) >= 4 {
			break
		}
	}
	return strings.Join(parts, "; ")
}

func collectEnvProxy() map[string]string {
	keys := []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"}
	out := map[string]string{}
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
			out[key] = safety.RedactProxyValue(val)
		}
	}
	return out
}

func (e Engine) collectGitProxy(ctx context.Context) map[string]string {
	out := map[string]string{}
	if !system.CommandExists("/usr/bin/git") {
		return out
	}
	checks := map[string][]string{
		"global.http.proxy":  {"config", "--global", "--get", "http.proxy"},
		"global.https.proxy": {"config", "--global", "--get", "https.proxy"},
		"system.http.proxy":  {"config", "--system", "--get", "http.proxy"},
		"system.https.proxy": {"config", "--system", "--get", "https.proxy"},
	}
	for key, args := range checks {
		res := e.runWithTimeout(ctx, 5*time.Second, "/usr/bin/git", args...)
		val := strings.TrimSpace(res.Stdout)
		if res.ExitCode == 0 && val != "" {
			out[key] = safety.RedactProxyValue(val)
		}
	}
	return out
}

func (e Engine) collectNpmProxy(ctx context.Context) map[string]string {
	out := map[string]string{}
	if !system.CommandExists("/usr/bin/npm") {
		return out
	}
	checks := map[string][]string{
		"proxy":       {"config", "get", "proxy"},
		"https-proxy": {"config", "get", "https-proxy"},
	}
	for key, args := range checks {
		res := e.runWithTimeout(ctx, 5*time.Second, "/usr/bin/npm", args...)
		val := strings.TrimSpace(res.Stdout)
		if res.ExitCode == 0 && val != "" && val != "null" && val != "undefined" {
			out[key] = safety.RedactProxyValue(val)
		}
	}
	return out
}

func (e Engine) collectBrewProxy(ctx context.Context) map[string]string {
	out := map[string]string{}
	brewPath, ok := system.FindFirstExisting("/opt/homebrew/bin/brew", "/usr/local/bin/brew")
	if !ok {
		return out
	}
	res := e.runWithTimeout(ctx, 8*time.Second, brewPath, "config")
	if res.ExitCode != 0 {
		return out
	}
	for _, raw := range strings.Split(res.Stdout, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.Contains(strings.ToLower(line), "proxy") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		val := line
		if len(parts) == 2 {
			val = strings.TrimSpace(parts[1])
		}
		out[key] = safety.RedactProxyValue(val)
	}
	return out
}

func (e Engine) loadEndpoints() rules.EndpointRules {
	if e.Options.RulesDir == "" {
		return rules.DefaultEndpointRules()
	}
	out, err := rules.LoadEndpointRules(filepath.Join(e.Options.RulesDir, "endpoints.json"))
	if err != nil {
		return rules.DefaultEndpointRules()
	}
	return out
}

func (e Engine) loadAgentRules(report *DiagnosticReport) []rules.AgentRule {
	if e.Options.RulesDir == "" {
		return nil
	}
	out, err := rules.LoadAgentRules(filepath.Join(e.Options.RulesDir, "known-agents.json"))
	if err != nil {
		report.Warnings = append(report.Warnings, "known agent rules not loaded: "+err.Error())
		return nil
	}
	return out
}

func (e Engine) probeRawIPs(ctx context.Context, targets []string) map[string]ProbeResult {
	out := map[string]ProbeResult{}
	for _, target := range targets {
		out[target] = e.probePing(ctx, target)
	}
	return out
}

func (e Engine) probePing(ctx context.Context, target string) ProbeResult {
	res := e.runWithTimeout(ctx, 5*time.Second, "/sbin/ping", pingArgs(target)...)
	status := "failed"
	if res.ExitCode == 0 {
		status = "ok"
	}
	return ProbeResult{Target: target, OK: res.ExitCode == 0, Status: status, Error: res.Error, DurationMS: res.DurationMillis, RedactedInfo: firstLine(res.Stdout + res.Stderr)}
}

func pingArgs(target string) []string {
	return []string{"-c", "1", target}
}

func (e Engine) probeDNS(ctx context.Context, targets []string) map[string]ProbeResult {
	out := map[string]ProbeResult{}
	for _, target := range targets {
		res := e.runWithTimeout(ctx, 5*time.Second, "/usr/bin/dscacheutil", "-q", "host", "-a", "name", target)
		ok := res.ExitCode == 0 && (strings.Contains(res.Stdout, "ip_address") || strings.Contains(res.Stdout, "ipv6_address"))
		status := "failed"
		if ok {
			status = "ok"
		}
		out[target] = ProbeResult{Target: target, OK: ok, Status: status, Error: res.Error, DurationMS: res.DurationMillis, RedactedInfo: firstLine(res.Stdout + res.Stderr)}
	}
	return out
}

func (e Engine) probeHTTPS(ctx context.Context, targets []string) map[string]ProbeResult {
	out := map[string]ProbeResult{}
	for _, target := range targets {
		res := e.runWithTimeout(ctx, 8*time.Second, "/usr/bin/curl", "-I", "-L", "--connect-timeout", "5", "--max-time", "8", "-sS", "-o", "/dev/null", "-w", "%{http_code}", target)
		code := strings.TrimSpace(res.Stdout)
		ok := res.ExitCode == 0 && code != "" && code != "000"
		status := "failed"
		if ok {
			status = "reachable"
		}
		host := target
		if parsed, err := url.Parse(target); err == nil && parsed.Host != "" {
			host = parsed.Host
		}
		out[target] = ProbeResult{Target: target, OK: ok, Status: status, HTTPStatus: code, Error: res.Error, DurationMS: res.DurationMillis, RedactedInfo: host}
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	line := strings.SplitN(s, "\n", 2)[0]
	if len(line) > 240 {
		return line[:240]
	}
	return line
}

func (e Engine) collectResidues(ctx context.Context, agentRules []rules.AgentRule, home string) ResidueInfo {
	var res ResidueInfo
	if len(agentRules) == 0 {
		return res
	}
	roots := []string{
		"/Library/LaunchDaemons",
		"/Library/LaunchAgents",
		"/Library/PrivilegedHelperTools",
		"~/Library/LaunchAgents",
		"~/Library/Group Containers",
		"~/Library/Application Support",
		"~/Library/Caches",
		"~/Library/Preferences",
	}
	seen := map[string]bool{}
	for _, root := range roots {
		expanded := system.ExpandUserPath(root, home)
		if expanded == "" || !system.Exists(expanded) {
			continue
		}
		trace("scan " + expanded)
		matches := scanResidueRoot(expanded, rootKind(root), agentRules, seen, 1)
		trace(fmt.Sprintf("done scan %s matches=%d", expanded, len(matches)))
		for _, m := range matches {
			addResidue(&res, m)
		}
	}
	_ = ctx
	return res
}

func trace(msg string) {
	if os.Getenv("AGENTLINK_TRACE") != "" {
		fmt.Fprintln(os.Stderr, "agentlink trace:", msg)
	}
}

func scanResidueRoot(root, kind string, agentRules []rules.AgentRule, seen map[string]bool, maxDepth int) []ResidueMatch {
	root = filepath.Clean(root)
	var out []ResidueMatch
	rootDepth := pathDepth(root)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		clean := filepath.Clean(path)
		if clean == root {
			return nil
		}
		depth := pathDepth(clean) - rootDepth
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		for _, rule := range agentRules {
			if rules.MatchRule(rule, clean) {
				key := rule.ID + "|" + clean
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, ResidueMatch{
					RuleID:                rule.ID,
					DisplayName:           rule.DisplayName,
					Risk:                  rule.Risk,
					Path:                  clean,
					Kind:                  kind,
					AutoQuarantineAllowed: rule.AutoQuarantineAllowed,
					DetectOnly:            !rule.AutoQuarantineAllowed,
				})
			}
		}
		if d.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func pathDepth(path string) int {
	path = filepath.Clean(path)
	if path == string(filepath.Separator) {
		return 0
	}
	return strings.Count(path, string(filepath.Separator))
}

func rootKind(root string) string {
	lower := strings.ToLower(root)
	switch {
	case strings.Contains(lower, "launchdaemons"):
		return "launchDaemon"
	case strings.Contains(lower, "launchagents"):
		return "launchAgent"
	case strings.Contains(lower, "privilegedhelpertools"):
		return "privilegedHelper"
	case strings.Contains(lower, "group containers"):
		return "groupContainer"
	case strings.Contains(lower, "application support"):
		return "appSupport"
	case strings.Contains(lower, "caches"):
		return "cache"
	case strings.Contains(lower, "preferences"):
		return "preference"
	default:
		return "residue"
	}
}

func addResidue(res *ResidueInfo, m ResidueMatch) {
	switch m.Kind {
	case "launchDaemon":
		res.LaunchDaemons = append(res.LaunchDaemons, m)
	case "launchAgent":
		res.LaunchAgents = append(res.LaunchAgents, m)
	case "privilegedHelper":
		res.PrivilegedHelpers = append(res.PrivilegedHelpers, m)
	case "groupContainer":
		res.GroupContainers = append(res.GroupContainers, m)
	case "appSupport":
		res.AppSupport = append(res.AppSupport, m)
	case "cache":
		res.Caches = append(res.Caches, m)
	case "preference":
		res.Preferences = append(res.Preferences, m)
	default:
		res.AppSupport = append(res.AppSupport, m)
	}
}

func (e Engine) collectSystemExtensions(ctx context.Context, agentRules []rules.AgentRule, report *DiagnosticReport) {
	if !system.CommandExists("/usr/bin/systemextensionsctl") {
		return
	}
	res := e.runWithTimeout(ctx, 10*time.Second, "/usr/bin/systemextensionsctl", "list")
	if res.ExitCode != 0 {
		return
	}
	patterns := []string{"clash", "verge", "mihomo", "surge", "tailscale", "adguard", "cloudflare", "warp", "littlesnitch", "little snitch", "zscaler", "cisco", "globalprotect"}
	for _, raw := range strings.Split(res.Stdout, "\n") {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)
		if line == "" {
			continue
		}
		if !containsAny(lower, patterns) {
			continue
		}
		match := ResidueMatch{DisplayName: "Network/System extension", Risk: "network_extension_suspected", Kind: "systemExtension", Raw: safety.RedactSensitive(line), DetectOnly: true}
		for _, rule := range agentRules {
			if rules.MatchRule(rule, line) {
				match.RuleID = rule.ID
				match.DisplayName = rule.DisplayName
				match.Risk = rule.Risk
				match.AutoQuarantineAllowed = false
				match.DetectOnly = true
				break
			}
		}
		report.Residues.SystemExtensions = append(report.Residues.SystemExtensions, match)
	}
}

func (e Engine) collectProfiles(ctx context.Context, report *DiagnosticReport) {
	if !system.CommandExists("/usr/bin/profiles") {
		return
	}
	res := e.runWithTimeout(ctx, 8*time.Second, "/usr/bin/profiles", "status", "-type", "enrollment")
	if res.ExitCode != 0 {
		return
	}
	lower := strings.ToLower(res.Stdout)
	if strings.Contains(lower, "enrolled via dep: yes") || strings.Contains(lower, "mdm enrollment: yes") || strings.Contains(lower, "user approved mdm: yes") {
		report.Residues.Profiles = append(report.Residues.Profiles, ResidueMatch{DisplayName: "MDM/Profile enrollment", Risk: "profile_mdm_suspected", Kind: "profile", DetectOnly: true, Raw: safety.RedactSensitive(strings.TrimSpace(res.Stdout))})
	}
}

func (e Engine) collectTopology(ctx context.Context, report *DiagnosticReport, home string) {
	for _, item := range loadProtectedTopologyManifests(home) {
		report.Topology.Interfaces = append(report.Topology.Interfaces, item.Interfaces...)
		report.Topology.ProtectedConstraints = append(report.Topology.ProtectedConstraints, item.ProtectedConstraints...)
	}
	if !system.CommandExists("/usr/bin/dns-sd") {
		return
	}
	// dns-sd browse is intentionally bounded and read-only. Its output is
	// treated as weak role evidence only; interface-bound manifest or report
	// topology still owns mutation boundaries.
	for _, svc := range []string{"_netaudio-arc._udp", "_netaudio-cmc._udp", "_ravenna._udp", "_ndi._tcp"} {
		res := e.runWithTimeout(ctx, 1500*time.Millisecond, "/usr/bin/dns-sd", "-B", svc, "local.")
		text := strings.ToLower(res.Stdout + "\n" + res.Stderr)
		if text == "" || !containsAny(text, []string{"add", "local", "_netaudio", "ravenna", "ndi"}) {
			continue
		}
		report.Topology.RedHerrings = append(report.Topology.RedHerrings, TopologyRedHerring{
			Class:  "UNBOUND_SERVICE_DISCOVERY_SIGNAL",
			Reason: "read-only DNS-SD saw " + svc + "; use a protected-topology manifest or scoped evidence before mutating interfaces",
		})
	}
}

type topologyManifest struct {
	SchemaVersion        int                  `json:"schemaVersion"`
	Interfaces           []TopologyInterface  `json:"interfaces"`
	ProtectedConstraints []TopologyConstraint `json:"protectedConstraints"`
}

func loadProtectedTopologyManifests(home string) []topologyManifest {
	var paths []string
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".config", "agentlink", "protected-topology.json"),
			filepath.Join(home, "Library", "Application Support", "Cactus AgentLink Rescue", "protected-topology.json"),
		)
	}
	paths = append(paths, "/Library/Application Support/Cactus AgentLink Rescue/protected-topology.json")
	var out []topologyManifest
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			continue
		}
		var m topologyManifest
		if json.Unmarshal(data, &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
