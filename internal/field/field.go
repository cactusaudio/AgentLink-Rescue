package field

import (
	"strings"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
)

type Report struct {
	SchemaVersion     int               `json:"schemaVersion"`
	ToolVersion       string            `json:"toolVersion"`
	FieldMode         string            `json:"fieldMode"`
	Diagnosis         DiagnosisSummary  `json:"diagnosis"`
	RecommendedAction string            `json:"recommendedAction"`
	Commands          map[string]string `json:"commands"`
	Warnings          []string          `json:"warnings"`
}

type DiagnosisSummary struct {
	Classes                   []string `json:"classes"`
	ProxyDirty                bool     `json:"proxyDirty"`
	ClashResidueDetected      bool     `json:"clashResidueDetected"`
	NetworkExtensionSuspected bool     `json:"networkExtensionSuspected"`
	ClashTunDetected          bool     `json:"clashTunDetected"`
	DefaultRouteOK            bool     `json:"defaultRouteOK"`
	DNSOK                     bool     `json:"dnsOK"`
}

func MacBookNetworkRescue(version string, report diagnose.DiagnosticReport) Report {
	classes := append([]string(nil), report.Classifications...)
	if len(classes) == 0 {
		tmp := report
		classify.Apply(&tmp)
		classes = tmp.Classifications
	}
	diag := DiagnosisSummary{
		Classes:                   classes,
		ProxyDirty:                report.Network.ProxySummary.Dirty,
		ClashResidueDetected:      hasClashResidue(report.Residues),
		NetworkExtensionSuspected: hasClass(classes, classify.NetworkExtensionSuspected) || hasNetworkExtensionResidue(report.Residues),
		ClashTunDetected:          hasClass(classes, classify.ClashTunActiveOrStale) || diagnose.DiagnoseTun(report).RecommendedRepair == "tun",
		DefaultRouteOK:            report.Network.DefaultRoute.Present,
		DNSOK:                     dnsOK(report),
	}
	return Report{
		SchemaVersion:     1,
		ToolVersion:       version,
		FieldMode:         "macbook-network-rescue",
		Diagnosis:         diag,
		RecommendedAction: recommendedAction(classes, diag),
		Commands: map[string]string{
			"safe":                "sudo ./bin/agentlink rescue --level safe",
			"tun":                 "sudo ./bin/agentlink rescue --level tun --yes",
			"standard":            "sudo ./bin/agentlink rescue --level standard --yes",
			"cleanBaseline":       "sudo ./bin/agentlink rescue --level clean-baseline --yes",
			"standardSystemReset": "sudo ./bin/agentlink rescue --level standard-system-reset --yes",
			"deep":                "sudo ./bin/agentlink rescue --level deep --yes",
			"verifyNetwork":       "./bin/agentlink verify network --json",
			"ticket":              "./bin/agentlink ticket create --type clash-tun-fix --json",
			"supportBundle":       "./bin/agentlink support bundle",
		},
		Warnings: []string{
			"AgentLink will not enable Clash TUN automatically.",
			"Use clean-baseline, standard-system-reset, or deep only after targeted TUN repair and restart-gate verification fail.",
			"GUI mode does not run sudo or collect passwords.",
		},
	}
}

func recommendedAction(classes []string, diag DiagnosisSummary) string {
	if diag.ClashTunDetected || hasClass(classes, classify.ClashTunActiveOrStale) || hasClass(classes, classify.TunRouteOwnershipSuspected) {
		return "tun"
	}
	if hasClass(classes, classify.OK) && !diag.ProxyDirty && !diag.NetworkExtensionSuspected {
		return "manual"
	}
	if hasClass(classes, classify.SysconfigSuspected) {
		return "deep"
	}
	if hasClass(classes, classify.NoDefaultRoute) || hasClass(classes, classify.KnownAgentResidue) || diag.NetworkExtensionSuspected || diag.ClashResidueDetected {
		return "standard"
	}
	if diag.ProxyDirty || hasClass(classes, classify.SystemProxyDirty) || hasClass(classes, classify.UserProxyDirty) || hasClass(classes, classify.DNSFail) || hasClass(classes, classify.NoDHCPLease) {
		return "safe"
	}
	if len(classes) == 0 || hasClass(classes, classify.Unknown) {
		return "manual"
	}
	return "safe"
}

func dnsOK(report diagnose.DiagnosticReport) bool {
	if len(report.Reachability.DNSNames) == 0 {
		return false
	}
	for _, probe := range report.Reachability.DNSNames {
		if probe.OK {
			return true
		}
	}
	return false
}

func hasClass(classes []string, want string) bool {
	for _, c := range classes {
		if c == want {
			return true
		}
	}
	return false
}

func hasClashResidue(res diagnose.ResidueInfo) bool {
	for _, item := range allResidues(res) {
		text := strings.ToLower(item.RuleID + " " + item.DisplayName + " " + item.Risk + " " + item.Path + " " + item.Raw)
		if strings.Contains(text, "clash") || strings.Contains(text, "verge") || strings.Contains(text, "mihomo") {
			return true
		}
	}
	return false
}

func hasNetworkExtensionResidue(res diagnose.ResidueInfo) bool {
	for _, item := range res.SystemExtensions {
		text := strings.ToLower(item.RuleID + " " + item.DisplayName + " " + item.Risk + " " + item.Path + " " + item.Raw)
		if strings.Contains(text, "network") || strings.Contains(text, "tun") || strings.Contains(text, "proxy") || strings.Contains(text, "filter") || strings.Contains(text, "clash") || strings.Contains(text, "verge") || strings.Contains(text, "mihomo") {
			return true
		}
	}
	return false
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
