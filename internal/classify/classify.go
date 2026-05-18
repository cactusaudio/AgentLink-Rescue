package classify

import (
	"strings"

	"cactus-agentlink-rescue/internal/diagnose"
)

const (
	OK                                    = "OK"
	NoActiveInterface                     = "NO_ACTIVE_INTERFACE"
	LinkLocalOnly                         = "LINK_LOCAL_ONLY"
	NoDHCPLease                           = "NO_DHCP_LEASE"
	NoDefaultRoute                        = "NO_DEFAULT_ROUTE"
	GatewayUnreachable                    = "GATEWAY_UNREACHABLE"
	RawIPUnreachable                      = "RAW_IP_UNREACHABLE"
	DNSFail                               = "DNS_FAIL"
	HTTPSFail                             = "HTTPS_FAIL"
	SystemProxyDirty                      = "SYSTEM_PROXY_DIRTY"
	UserProxyDirty                        = "USER_PROXY_DIRTY"
	GitProxyDirty                         = "GIT_PROXY_DIRTY"
	NpmProxyDirty                         = "NPM_PROXY_DIRTY"
	BrewProxyDirty                        = "BREW_PROXY_DIRTY"
	KnownAgentResidue                     = "KNOWN_AGENT_RESIDUE"
	NetworkExtensionSuspected             = "NETWORK_EXTENSION_SUSPECTED"
	ClashTunActiveOrStale                 = "CLASH_TUN_ACTIVE_OR_STALE"
	NetworkExtensionSessionStale          = "NETWORK_EXTENSION_SESSION_STALE"
	TunRouteOwnershipSuspected            = "TUN_ROUTE_OWNERSHIP_SUSPECTED"
	AirDropDiscoveryDegraded              = "AIRDROP_DISCOVERY_DEGRADED"
	ClashCleanReinstallRequired           = "CLASH_CLEAN_REINSTALL_REQUIRED"
	RestartGateRequired                   = "RESTART_GATE_REQUIRED"
	NetworkLocationSuspected              = "NETWORK_LOCATION_SUSPECTED"
	SysconfigSuspected                    = "SYSCONFIG_SUSPECTED"
	MDMProfileSuspected                   = "MDM_PROFILE_SUSPECTED"
	GeneralInternetOKAgentEndpointBlocked = "GENERAL_INTERNET_OK_AGENT_ENDPOINT_BLOCKED"
	ProtectedTopologyConstraint           = "PROTECTED_TOPOLOGY_CONSTRAINT"
	ProtectedAudioVLANRouteTrap           = "PROTECTED_AUDIO_VLAN_ROUTE_TRAP"
	Unknown                               = "UNKNOWN"
)

func Apply(r *diagnose.DiagnosticReport) {
	diagnose.AnalyzeTopology(r)
	classes := Classify(*r)
	r.Classifications = classes
	r.RecommendedRepairLevel = RecommendedRepairLevel(classes)
}

func Classify(r diagnose.DiagnosticReport) []string {
	diagnose.AnalyzeTopology(&r)
	added := map[string]bool{}
	var out []string
	add := func(c string) {
		if c != "" && !added[c] {
			added[c] = true
			out = append(out, c)
		}
	}

	activeWithIP, anyLinkLocal, anyDHCPNoLease := interfaceState(r)
	if !activeWithIP {
		add(NoActiveInterface)
	}
	if anyLinkLocal && !hasRoutableIPv4(r.Network.Interfaces) {
		add(LinkLocalOnly)
	}
	if anyDHCPNoLease {
		add(NoDHCPLease)
	}
	if !r.Network.DefaultRoute.Present {
		add(NoDefaultRoute)
	}

	rawOK := anyProbeOK(r.Reachability.RawIPs)
	dnsOK := anyProbeOK(r.Reachability.DNSNames)
	httpsOK := anyProbeOK(r.Reachability.HTTPSTargets)
	agentOK := allProbeOK(r.Reachability.AgentTargets)
	internetBroken := !rawOK || !dnsOK || !httpsOK || !r.Network.DefaultRoute.Present

	if r.Network.DefaultRoute.Present && r.Reachability.Gateway.Target != "" && !r.Reachability.Gateway.OK && !rawOK {
		add(GatewayUnreachable)
	}
	if len(r.Reachability.RawIPs) > 0 && !rawOK {
		add(RawIPUnreachable)
	}
	if rawOK && len(r.Reachability.DNSNames) > 0 && !dnsOK {
		add(DNSFail)
	}
	if dnsOK && len(r.Reachability.HTTPSTargets) > 0 && !httpsOK {
		add(HTTPSFail)
	}
	if r.Network.ProxySummary.Dirty {
		add(SystemProxyDirty)
	}
	if len(nonEmptyValues(r.UserConfig.EnvProxy)) > 0 {
		add(UserProxyDirty)
	}
	if len(nonEmptyValues(r.UserConfig.GitProxy)) > 0 {
		add(GitProxyDirty)
	}
	if len(nonEmptyValues(r.UserConfig.NpmProxy)) > 0 {
		add(NpmProxyDirty)
	}
	if len(nonEmptyValues(r.UserConfig.BrewProxy)) > 0 {
		add(BrewProxyDirty)
	}
	if hasAutoResidue(r.Residues) && internetBroken {
		add(KnownAgentResidue)
	}
	protectedTrap := diagnose.ProtectedAudioRouteTrap(r)
	protectedConstraint := diagnose.HasProtectedTopologyConstraint(r)
	tun := diagnose.DiagnoseTun(r)
	if tun.RecommendedRepair == "tun" && !protectedTrap && !protectedConstraint {
		add(ClashTunActiveOrStale)
	}
	if proxyClean(r) && internetBroken && hasNetworkFilterResidue(r.Residues) {
		add(NetworkExtensionSuspected)
	}
	if hasClassValue(out, ClashTunActiveOrStale) && (hasClassValue(out, RawIPUnreachable) || hasClassValue(out, HTTPSFail) || hasClassValue(out, GatewayUnreachable)) {
		add(TunRouteOwnershipSuspected)
	}
	if awdlDegraded(r) && (hasClassValue(out, ClashTunActiveOrStale) || hasClassValue(out, NetworkExtensionSuspected)) {
		add(AirDropDiscoveryDegraded)
	}
	if hasProfiles(r.Residues) {
		add(MDMProfileSuspected)
	}
	if rawOK && dnsOK && httpsOK && len(r.Reachability.AgentTargets) > 0 && !agentOK {
		add(GeneralInternetOKAgentEndpointBlocked)
	}
	if protectedTrap {
		add(ProtectedAudioVLANRouteTrap)
	} else if protectedConstraint {
		add(ProtectedTopologyConstraint)
	}
	if len(out) == 0 {
		add(OK)
	}
	return out
}

func RecommendedRepairLevel(classes []string) string {
	has := func(c string) bool {
		for _, x := range classes {
			if x == c {
				return true
			}
		}
		return false
	}
	if has(GeneralInternetOKAgentEndpointBlocked) {
		return "none"
	}
	if has(ProtectedAudioVLANRouteTrap) {
		return "safe"
	}
	if has(ProtectedTopologyConstraint) {
		return "safe"
	}
	if has(ClashTunActiveOrStale) || has(TunRouteOwnershipSuspected) {
		return "tun"
	}
	if has(SysconfigSuspected) {
		return "deep"
	}
	if has(NoDefaultRoute) || has(NetworkLocationSuspected) || has(KnownAgentResidue) {
		return "standard"
	}
	if has(SystemProxyDirty) || has(DNSFail) || has(NoDHCPLease) || has(UserProxyDirty) || has(GitProxyDirty) || has(NpmProxyDirty) || has(BrewProxyDirty) {
		return "safe"
	}
	if len(classes) == 1 && classes[0] == OK {
		return "none"
	}
	return "safe"
}

func hasClassValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func interfaceState(r diagnose.DiagnosticReport) (activeWithIP bool, anyLinkLocal bool, anyDHCPNoLease bool) {
	hardwareDevices := map[string]bool{}
	for _, hp := range r.Network.HardwarePorts {
		if hp.Device != "" {
			hardwareDevices[hp.Device] = true
		}
	}
	relevantDevices := map[string]bool{}
	for _, iface := range r.Network.Interfaces {
		if !isRelevantInterface(iface, r.Network.DefaultRoute.Interface, hardwareDevices) {
			continue
		}
		if iface.LinkLocalOnly {
			anyLinkLocal = true
		}
		if hasRoutableIP(iface) && (strings.EqualFold(iface.Status, "active") || iface.Name == r.Network.DefaultRoute.Interface || hardwareDevices[iface.Name]) {
			activeWithIP = true
		}
		if strings.EqualFold(iface.Status, "active") || iface.Name == r.Network.DefaultRoute.Interface || hasRoutableIP(iface) {
			relevantDevices[iface.Name] = true
		}
	}
	for _, dhcp := range r.Network.DHCP {
		if dhcp.Device != "" && relevantDevices[dhcp.Device] && !dhcp.HasLease {
			anyDHCPNoLease = true
		}
		if dhcp.Device != "" && relevantDevices[dhcp.Device] && dhcp.LinkLocalOnly {
			anyLinkLocal = true
		}
	}
	return
}

func isRelevantInterface(iface diagnose.NetworkInterface, defaultRouteInterface string, hardwareDevices map[string]bool) bool {
	if isNoisyVirtualInterface(iface.Name) {
		return false
	}
	if hardwareDevices[iface.Name] && strings.EqualFold(iface.Status, "active") {
		return true
	}
	if iface.Name == defaultRouteInterface {
		return true
	}
	return hasRoutableIPv4Address(iface)
}

func isNoisyVirtualInterface(name string) bool {
	name = strings.ToLower(name)
	if name == "lo0" {
		return true
	}
	for _, prefix := range []string{"utun", "awdl", "llw", "bridge", "stf", "gif", "ap"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func awdlDegraded(r diagnose.DiagnosticReport) bool {
	for _, iface := range r.Network.Interfaces {
		if iface.Name == "awdl0" {
			return !strings.EqualFold(iface.Status, "active")
		}
	}
	return false
}

func hasRoutableIP(iface diagnose.NetworkInterface) bool {
	for _, ip := range iface.IPv4 {
		if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
			return true
		}
	}
	for _, ip := range iface.IPv6 {
		lower := strings.ToLower(ip)
		if !strings.HasPrefix(lower, "fe80:") && lower != "::1" {
			return true
		}
	}
	return false
}

func hasRoutableIPv4Address(iface diagnose.NetworkInterface) bool {
	for _, ip := range iface.IPv4 {
		if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
			return true
		}
	}
	return false
}

func hasRoutableIPv4(ifaces []diagnose.NetworkInterface) bool {
	for _, iface := range ifaces {
		if isNoisyVirtualInterface(iface.Name) {
			continue
		}
		for _, ip := range iface.IPv4 {
			if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
				return true
			}
		}
	}
	return false
}

func anyProbeOK(m map[string]diagnose.ProbeResult) bool {
	for _, p := range m {
		if p.OK {
			return true
		}
	}
	return false
}

func allProbeOK(m map[string]diagnose.ProbeResult) bool {
	if len(m) == 0 {
		return true
	}
	for _, p := range m {
		if !p.OK {
			return false
		}
	}
	return true
}

func nonEmptyValues(m map[string]string) []string {
	var vals []string
	for _, v := range m {
		if strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "null" && strings.TrimSpace(v) != "undefined" {
			vals = append(vals, v)
		}
	}
	return vals
}

func proxyClean(r diagnose.DiagnosticReport) bool {
	return !r.Network.ProxySummary.Dirty && len(nonEmptyValues(r.UserConfig.EnvProxy)) == 0 && len(nonEmptyValues(r.UserConfig.GitProxy)) == 0 && len(nonEmptyValues(r.UserConfig.NpmProxy)) == 0 && len(nonEmptyValues(r.UserConfig.BrewProxy)) == 0
}

func hasAutoResidue(res diagnose.ResidueInfo) bool {
	for _, item := range allResidues(res) {
		if item.AutoQuarantineAllowed {
			return true
		}
	}
	return false
}

func hasNetworkFilterResidue(res diagnose.ResidueInfo) bool {
	for _, item := range allResidues(res) {
		text := strings.ToLower(item.Risk + " " + item.DisplayName + " " + item.Raw + " " + item.Path)
		if strings.Contains(text, "network") || strings.Contains(text, "proxy") || strings.Contains(text, "tun") || strings.Contains(text, "vpn") || strings.Contains(text, "filter") {
			return true
		}
	}
	return false
}

func hasProfiles(res diagnose.ResidueInfo) bool {
	return len(res.Profiles) > 0
}

func allResidues(res diagnose.ResidueInfo) []diagnose.ResidueMatch {
	var out []diagnose.ResidueMatch
	out = append(out, res.LaunchDaemons...)
	out = append(out, res.LaunchAgents...)
	out = append(out, res.PrivilegedHelpers...)
	out = append(out, res.SystemExtensions...)
	out = append(out, res.GroupContainers...)
	out = append(out, res.AppSupport...)
	out = append(out, res.Caches...)
	out = append(out, res.Preferences...)
	out = append(out, res.Profiles...)
	return out
}
