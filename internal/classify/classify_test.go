package classify

import (
	"testing"

	"cactus-agentlink-rescue/internal/diagnose"
)

func TestClassifierDirtyProxyRecommendsSafe(t *testing.T) {
	r := healthyReport()
	r.Network.ProxySummary.Dirty = true
	classes := Classify(r)
	if !has(classes, SystemProxyDirty) {
		t.Fatalf("missing SYSTEM_PROXY_DIRTY: %v", classes)
	}
	if got := RecommendedRepairLevel(classes); got != "safe" {
		t.Fatalf("level=%s", got)
	}
}

func TestClassifierRawIPOKDNSFail(t *testing.T) {
	r := healthyReport()
	r.Reachability.DNSNames = map[string]diagnose.ProbeResult{"apple.com": {Target: "apple.com", OK: false}}
	classes := Classify(r)
	if !has(classes, DNSFail) {
		t.Fatalf("missing DNS_FAIL: %v", classes)
	}
}

func TestClassifierNoDefaultRouteRecommendsStandard(t *testing.T) {
	r := healthyReport()
	r.Network.DefaultRoute.Present = false
	classes := Classify(r)
	if !has(classes, NoDefaultRoute) {
		t.Fatalf("missing NO_DEFAULT_ROUTE: %v", classes)
	}
	if got := RecommendedRepairLevel(classes); got != "standard" {
		t.Fatalf("level=%s", got)
	}
}

func TestClassifierClashResidueNetworkBroken(t *testing.T) {
	r := healthyReport()
	r.Reachability.RawIPs = map[string]diagnose.ProbeResult{"1.1.1.1": {Target: "1.1.1.1", OK: false}}
	r.Reachability.DNSNames = map[string]diagnose.ProbeResult{"apple.com": {Target: "apple.com", OK: false}}
	r.Reachability.HTTPSTargets = map[string]diagnose.ProbeResult{"https://github.com/": {Target: "https://github.com/", OK: false}}
	r.Residues.PrivilegedHelpers = []diagnose.ResidueMatch{{
		RuleID:                "clash_verge",
		DisplayName:           "Clash Verge / Clash Verge Rev",
		Risk:                  "network_proxy_tun_residue",
		Path:                  "/Library/PrivilegedHelperTools/io.github.clashverge.helper",
		Kind:                  "privilegedHelper",
		AutoQuarantineAllowed: true,
	}}
	classes := Classify(r)
	if !has(classes, KnownAgentResidue) || !has(classes, NetworkExtensionSuspected) {
		t.Fatalf("missing residue classes: %v", classes)
	}
}

func TestClassifierClashTunRecommendsTun(t *testing.T) {
	r := healthyReport()
	r.Reachability.RawIPs = map[string]diagnose.ProbeResult{"1.1.1.1": {Target: "1.1.1.1", OK: false}}
	r.Network.Interfaces = append(r.Network.Interfaces, diagnose.NetworkInterface{Name: "utun0", Status: "active", IsUTun: true, IPv4: []string{"198.18.0.1"}, IPv6: []string{"fdfe:dcba:9876::1"}})
	r.Residues.PrivilegedHelpers = []diagnose.ResidueMatch{{
		RuleID:                "clash_verge",
		DisplayName:           "Clash Verge Rev",
		Risk:                  "network_proxy_tun_residue",
		Path:                  "/Library/PrivilegedHelperTools/io.github.clash-verge-rev.clash-verge-rev.service.bundle",
		Kind:                  "privilegedHelper",
		AutoQuarantineAllowed: true,
	}}
	classes := Classify(r)
	if !has(classes, ClashTunActiveOrStale) {
		t.Fatalf("missing TUN class: %v", classes)
	}
	if got := RecommendedRepairLevel(classes); got != "tun" {
		t.Fatalf("level=%s classes=%v", got, classes)
	}
}

func TestInactiveHardwareNoDHCPDoesNotTriggerNoLease(t *testing.T) {
	r := healthyReport()
	r.Network.HardwarePorts = []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}, {Port: "USB 10/100/1000 LAN", Device: "en6"}}
	r.Network.Interfaces = []diagnose.NetworkInterface{
		{Name: "en0", Status: "active", IPv4: []string{"192.168.1.20"}},
		{Name: "en6", Status: "inactive"},
	}
	r.Network.DHCP = []diagnose.DHCPInfo{
		{Device: "en0", IPv4: "192.168.1.20", HasLease: true},
		{Device: "en6", HasLease: false},
	}
	classes := Classify(r)
	if has(classes, NoDHCPLease) {
		t.Fatalf("inactive hardware generated NO_DHCP_LEASE: %v", classes)
	}
}

func TestBridgeNoLeaseDoesNotTriggerNoLease(t *testing.T) {
	r := healthyReport()
	r.Network.HardwarePorts = []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}, {Port: "Thunderbolt Bridge", Device: "bridge0"}}
	r.Network.Interfaces = []diagnose.NetworkInterface{
		{Name: "en0", Status: "active", IPv4: []string{"192.168.1.20"}},
		{Name: "bridge0", Status: "active"},
	}
	r.Network.DHCP = []diagnose.DHCPInfo{
		{Device: "en0", IPv4: "192.168.1.20", HasLease: true},
		{Device: "bridge0", HasLease: false},
	}
	classes := Classify(r)
	if has(classes, NoDHCPLease) {
		t.Fatalf("bridge0 generated NO_DHCP_LEASE: %v", classes)
	}
}

func TestUTunDoesNotSatisfyActivePhysicalConnectivity(t *testing.T) {
	r := healthyReport()
	r.Network.DefaultRoute = diagnose.DefaultRoute{Present: true, Interface: "utun0"}
	r.Network.HardwarePorts = []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}}
	r.Network.Interfaces = []diagnose.NetworkInterface{
		{Name: "en0", Status: "inactive"},
		{Name: "utun0", Status: "active", IPv4: []string{"10.8.0.2"}},
	}
	r.Network.DHCP = nil
	classes := Classify(r)
	if !has(classes, NoActiveInterface) {
		t.Fatalf("utun0 made machine look physically connected: %v", classes)
	}
}

func TestDefaultRouteHardwareWithRoutableIPAndNoStatusIsConnected(t *testing.T) {
	r := healthyReport()
	r.Network.DefaultRoute = diagnose.DefaultRoute{Present: true, Gateway: "192.168.1.1", Interface: "en0"}
	r.Network.HardwarePorts = []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}}
	r.Network.Interfaces = []diagnose.NetworkInterface{{Name: "en0", IPv4: []string{"192.168.1.20"}}}
	classes := Classify(r)
	if has(classes, NoActiveInterface) {
		t.Fatalf("default-route hardware interface with IPv4 was not connected: %v", classes)
	}
}

func healthyReport() diagnose.DiagnosticReport {
	return diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true, Gateway: "192.168.1.1", Interface: "en0"},
			Interfaces:   []diagnose.NetworkInterface{{Name: "en0", Status: "active", IPv4: []string{"192.168.1.20"}}},
		},
		Reachability: diagnose.ReachabilityInfo{
			Gateway:      diagnose.ProbeResult{Target: "192.168.1.1", OK: true},
			RawIPs:       map[string]diagnose.ProbeResult{"1.1.1.1": {Target: "1.1.1.1", OK: true}},
			DNSNames:     map[string]diagnose.ProbeResult{"apple.com": {Target: "apple.com", OK: true}},
			HTTPSTargets: map[string]diagnose.ProbeResult{"https://github.com/": {Target: "https://github.com/", OK: true}},
			AgentTargets: map[string]diagnose.ProbeResult{"https://api.openai.com/": {Target: "https://api.openai.com/", OK: true}},
		},
		UserConfig: diagnose.UserConfigInfo{
			EnvProxy:  map[string]string{},
			GitProxy:  map[string]string{},
			NpmProxy:  map[string]string{},
			BrewProxy: map[string]string{},
		},
	}
}

func has(classes []string, target string) bool {
	for _, c := range classes {
		if c == target {
			return true
		}
	}
	return false
}
