package networkverify

import (
	"context"
	"time"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion          int                       `json:"schemaVersion"`
	ToolVersion            string                    `json:"toolVersion"`
	CreatedAt              string                    `json:"createdAt"`
	OK                     bool                      `json:"ok"`
	FailureClasses         []string                  `json:"failureClasses"`
	NextRecommendation     string                    `json:"nextRecommendation"`
	SupportBundleSuggested bool                      `json:"supportBundleSuggested"`
	ActiveInterface        string                    `json:"activeInterface,omitempty"`
	CurrentLocation        string                    `json:"currentLocation,omitempty"`
	DHCPGateway            string                    `json:"dhcpGateway,omitempty"`
	DefaultRouteOK         bool                      `json:"defaultRouteOK"`
	GatewayOK              bool                      `json:"gatewayOK"`
	RawIPOK                bool                      `json:"rawIpOk"`
	DNSOK                  bool                      `json:"dnsOk"`
	HTTPSOK                bool                      `json:"httpsOk"`
	SystemProxyDirty       bool                      `json:"systemProxyDirty"`
	Tun                    diagnose.TunReport        `json:"tun"`
	AWDLUp                 bool                      `json:"awdlUp"`
	FirewallBlockAll       string                    `json:"firewallBlockAll,omitempty"`
	Diagnostic             diagnose.DiagnosticReport `json:"diagnostic,omitempty"`
}

func Run(ctx context.Context, runner command.Runner, rulesDir string, includeDiagnostic bool) Report {
	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: rulesDir})
	diag := engine.Run(ctx)
	classify.Apply(&diag)
	tun := diagnose.DiagnoseTun(diag)
	rep := Report{
		SchemaVersion:          1,
		ToolVersion:            system.Version,
		CreatedAt:              time.Now().Format(time.RFC3339),
		FailureClasses:         diag.Classifications,
		NextRecommendation:     nextRecommendation(diag, tun),
		CurrentLocation:        diag.Network.CurrentLocation,
		DHCPGateway:            diag.Network.DefaultRoute.Gateway,
		DefaultRouteOK:         diag.Network.DefaultRoute.Present,
		GatewayOK:              diag.Reachability.Gateway.OK,
		RawIPOK:                anyProbeOK(diag.Reachability.RawIPs),
		DNSOK:                  anyProbeOK(diag.Reachability.DNSNames),
		HTTPSOK:                anyProbeOK(diag.Reachability.HTTPSTargets),
		SystemProxyDirty:       diag.Network.ProxySummary.Dirty,
		Tun:                    tun,
		AWDLUp:                 awdlUp(diag),
		SupportBundleSuggested: true,
	}
	rep.ActiveInterface = diag.Network.DefaultRoute.Interface
	rep.OK = len(diag.Classifications) == 1 && diag.Classifications[0] == classify.OK && tun.RecommendedRepair == ""
	rep.SupportBundleSuggested = !rep.OK
	if includeDiagnostic {
		rep.Diagnostic = diag
	}
	return rep
}

func nextRecommendation(diag diagnose.DiagnosticReport, tun diagnose.TunReport) string {
	if tun.RecommendedRepair == "tun" {
		return "tun"
	}
	if diag.RecommendedRepairLevel != "" {
		return diag.RecommendedRepairLevel
	}
	return "none"
}

func anyProbeOK(m map[string]diagnose.ProbeResult) bool {
	for _, p := range m {
		if p.OK {
			return true
		}
	}
	return false
}

func awdlUp(diag diagnose.DiagnosticReport) bool {
	for _, iface := range diag.Network.Interfaces {
		if iface.Name == "awdl0" && iface.Status == "active" {
			return true
		}
	}
	return false
}
