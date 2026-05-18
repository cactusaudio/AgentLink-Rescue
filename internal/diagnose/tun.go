package diagnose

import (
	"net"
	"strings"

	"cactus-agentlink-rescue/internal/interference"
)

type TunReport struct {
	SchemaVersion        int                         `json:"schemaVersion"`
	UTunInterfaces       []TunInterface              `json:"utunInterfaces"`
	SuspiciousSignatures []string                    `json:"suspiciousTunSignatures"`
	Providers            []interference.ProviderFact `json:"providers"`
	RecommendedRepair    string                      `json:"recommendedRepair"`
	Warnings             []string                    `json:"warnings,omitempty"`
}

type TunInterface struct {
	Name       string   `json:"name"`
	Status     string   `json:"status,omitempty"`
	IPv4       []string `json:"ipv4,omitempty"`
	IPv6       []string `json:"ipv6,omitempty"`
	Suspicious bool     `json:"suspicious"`
	Reasons    []string `json:"reasons,omitempty"`
}

func DiagnoseTun(report DiagnosticReport) TunReport {
	AnalyzeTopology(&report)
	out := TunReport{SchemaVersion: 1}
	for _, iface := range report.Network.Interfaces {
		if !iface.IsUTun && !strings.HasPrefix(iface.Name, "utun") {
			continue
		}
		item := TunInterface{Name: iface.Name, Status: iface.Status, IPv4: append([]string(nil), iface.IPv4...), IPv6: append([]string(nil), iface.IPv6...)}
		for _, ip := range iface.IPv4 {
			if ip == "198.18.0.1" {
				item.Suspicious = true
				item.Reasons = append(item.Reasons, "Clash/Mihomo default TUN IPv4 198.18.0.1")
			} else if in19818Range(ip) {
				item.Suspicious = true
				item.Reasons = append(item.Reasons, "IPv4 in 198.18.0.0/15 benchmarking range often used by proxy TUN")
			}
		}
		for _, ip := range iface.IPv6 {
			if strings.EqualFold(strings.TrimSpace(ip), "fdfe:dcba:9876::1") {
				item.Suspicious = true
				item.Reasons = append(item.Reasons, "Clash/Mihomo default TUN IPv6 fdfe:dcba:9876::1")
			}
		}
		if item.Suspicious {
			for _, reason := range item.Reasons {
				out.SuspiciousSignatures = append(out.SuspiciousSignatures, item.Name+": "+reason)
			}
		}
		out.UTunInterfaces = append(out.UTunInterfaces, item)
	}
	out.Providers = interference.Facts(allResidueText(report.Residues), out.SuspiciousSignatures)
	if HasProtectedRouteOwnershipConflict(report) {
		out.Warnings = append(out.Warnings, "protected topology route ownership conflict detected; suppressing TUN repair recommendation until scoped ownership is proven")
		return out
	}
	if hasSuspiciousTun(out) {
		out.RecommendedRepair = "tun"
	}
	return out
}

func hasSuspiciousTun(report TunReport) bool {
	if len(report.SuspiciousSignatures) == 0 {
		return false
	}
	for _, p := range report.Providers {
		if (p.ID == interference.ProviderClashVergeRev || p.ID == interference.ProviderClashX || p.ID == interference.ProviderMihomoGeneric) && p.Detected {
			return true
		}
	}
	return false
}

func in19818Range(raw string) bool {
	ip := net.ParseIP(raw)
	if ip == nil {
		return false
	}
	v4 := ip.To4()
	return v4 != nil && v4[0] == 198 && (v4[1] == 18 || v4[1] == 19)
}

func allResidueText(res ResidueInfo) []string {
	var out []string
	add := func(items []ResidueMatch) {
		for _, item := range items {
			out = append(out, item.RuleID+" "+item.DisplayName+" "+item.Path+" "+item.Kind+" "+item.Raw)
		}
	}
	add(res.LaunchDaemons)
	add(res.LaunchAgents)
	add(res.PrivilegedHelpers)
	add(res.SystemExtensions)
	add(res.GroupContainers)
	add(res.AppSupport)
	add(res.Caches)
	add(res.Preferences)
	add(res.Profiles)
	return out
}
