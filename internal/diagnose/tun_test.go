package diagnose

import "testing"

func TestDiagnoseTunDetectsClashSignatures(t *testing.T) {
	report := DiagnosticReport{
		Network: NetworkInfo{Interfaces: []NetworkInterface{
			{Name: "utun0", IsUTun: true, IPv4: []string{"198.18.0.1"}, IPv6: []string{"fdfe:dcba:9876::1"}},
		}},
		Residues: ResidueInfo{LaunchAgents: []ResidueMatch{{
			RuleID: "clash_verge_rev", DisplayName: "Clash Verge Rev", Path: "/Users/me/Library/LaunchAgents/io.github.clash-verge-rev.plist",
		}}},
	}
	tun := DiagnoseTun(report)
	if tun.RecommendedRepair != "tun" {
		t.Fatalf("recommended=%q report=%+v", tun.RecommendedRepair, tun)
	}
	if len(tun.SuspiciousSignatures) == 0 || len(tun.Providers) == 0 {
		t.Fatalf("missing evidence: %+v", tun)
	}
}

func TestDiagnoseTunIgnoresUTunWithoutProviderResidue(t *testing.T) {
	report := DiagnosticReport{Network: NetworkInfo{Interfaces: []NetworkInterface{
		{Name: "utun0", IsUTun: true, IPv4: []string{"10.8.0.2"}},
	}}}
	tun := DiagnoseTun(report)
	if tun.RecommendedRepair == "tun" {
		t.Fatalf("unexpected recommendation: %+v", tun)
	}
}
