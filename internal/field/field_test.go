package field

import (
	"testing"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
)

func TestMacBookNetworkRescueReportsClashTunStandard(t *testing.T) {
	rep := MacBookNetworkRescue("test", diagnose.DiagnosticReport{
		Classifications: []string{classify.NetworkExtensionSuspected},
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true},
			ProxySummary: diagnose.ProxySummary{
				Dirty: false,
			},
		},
		Reachability: diagnose.ReachabilityInfo{
			DNSNames: map[string]diagnose.ProbeResult{"apple.com": {Target: "apple.com", OK: true}},
		},
		Residues: diagnose.ResidueInfo{
			SystemExtensions: []diagnose.ResidueMatch{{RuleID: "clash_verge", DisplayName: "Clash Verge", Risk: "network_proxy_tun_residue"}},
		},
	})
	if rep.SchemaVersion != 1 || rep.ToolVersion != "test" || rep.FieldMode != "macbook-network-rescue" {
		t.Fatalf("bad report identity: %+v", rep)
	}
	if !rep.Diagnosis.ClashResidueDetected || !rep.Diagnosis.NetworkExtensionSuspected {
		t.Fatalf("expected clash/tun suspicion: %+v", rep.Diagnosis)
	}
	if !rep.Diagnosis.DefaultRouteOK || !rep.Diagnosis.DNSOK {
		t.Fatalf("expected route/dns ok summary: %+v", rep.Diagnosis)
	}
	if rep.RecommendedAction != "standard" {
		t.Fatalf("recommended action = %s", rep.RecommendedAction)
	}
	if rep.Commands["standard"] != "sudo ./bin/agentlink rescue --level standard --yes" {
		t.Fatalf("standard command = %q", rep.Commands["standard"])
	}
}

func TestMacBookNetworkRescueHealthyManual(t *testing.T) {
	rep := MacBookNetworkRescue("test", diagnose.DiagnosticReport{
		Classifications: []string{classify.OK},
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true},
		},
		Reachability: diagnose.ReachabilityInfo{
			DNSNames: map[string]diagnose.ProbeResult{"apple.com": {Target: "apple.com", OK: true}},
		},
	})
	if rep.RecommendedAction != "manual" {
		t.Fatalf("healthy field report should not recommend repair, got %s", rep.RecommendedAction)
	}
}
