package verify

import (
	"encoding/json"
	"fmt"
	"time"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/rules"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/snapshot"
)

func Selftest() []error {
	var errs []error
	proxy := diagnose.ParseScutilProxy("HTTPEnable : 0\nHTTPSEnable : 1\nSOCKSEnable : 0\nProxyAutoConfigEnable : 0\n")
	if !proxy.Dirty || !proxy.HTTPSEnabled {
		errs = append(errs, fmt.Errorf("proxy parser failed dirty sample"))
	}
	services := diagnose.ParseNetworkServices("An asterisk (*) denotes that a network service is disabled.\nWi-Fi\n*Thunderbolt Bridge\n")
	if len(services) != 2 || !services[1].Disabled {
		errs = append(errs, fmt.Errorf("service parser failed"))
	}
	route := diagnose.ParseRouteDefault("   route to: default\ndestination: default\n       mask: default\n    gateway: 192.168.1.1\n  interface: en0\n", "")
	if !route.Present || route.Gateway != "192.168.1.1" {
		errs = append(errs, fmt.Errorf("route parser failed"))
	}
	if safety.RedactURLCredentials("http://user:pass@example.com:8080") != "http://REDACTED@example.com:8080" {
		errs = append(errs, fmt.Errorf("redaction failed"))
	}
	if !rules.MatchRule(rules.AgentRule{Patterns: []string{"*clash*"}}, "/Library/PrivilegedHelperTools/io.github.clashverge.helper") {
		errs = append(errs, fmt.Errorf("rule matching failed"))
	}
	r := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true},
			Interfaces:   []diagnose.NetworkInterface{{Name: "en0", Status: "active", IPv4: []string{"192.168.1.2"}}},
		},
		Reachability: diagnose.ReachabilityInfo{
			RawIPs:   map[string]diagnose.ProbeResult{"1.1.1.1": {OK: true}},
			DNSNames: map[string]diagnose.ProbeResult{"apple.com": {OK: false}},
		},
	}
	classes := classify.Classify(r)
	if !contains(classes, classify.DNSFail) {
		errs = append(errs, fmt.Errorf("classifier DNS_FAIL failed: %v", classes))
	}
	if snapshot.GenerateID(time.Date(2026, 5, 1, 2, 14, 13, 0, time.UTC)) != "20260501-021413" {
		errs = append(errs, fmt.Errorf("restore point id generation failed"))
	}
	if _, err := json.Marshal(diagnose.DiagnosticReport{SchemaVersion: 1}); err != nil {
		errs = append(errs, fmt.Errorf("report JSON marshal failed: %v", err))
	}
	return errs
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
