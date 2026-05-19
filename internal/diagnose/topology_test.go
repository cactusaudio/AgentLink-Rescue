package diagnose

import "testing"

func TestAnalyzeTopologyHonorsStructuredAES67WithoutRaw(t *testing.T) {
	r := routeTrapReport()
	r.Topology.Interfaces = []TopologyInterface{{
		Name:         "en0",
		Roles:        []string{"protected_media"},
		RoleEvidence: []string{"AES67 PTP protected audio VLAN"},
		Protected:    true,
	}}
	AnalyzeTopology(&r)
	if len(r.Topology.ProtectedConstraints) != 1 {
		t.Fatalf("constraints=%+v", r.Topology.ProtectedConstraints)
	}
	if r.Topology.ProtectedConstraints[0].Class != TopologyClassProtectedAudioVLAN {
		t.Fatalf("class=%s", r.Topology.ProtectedConstraints[0].Class)
	}
	if !ProtectedAudioRouteTrap(r) {
		t.Fatalf("expected protected audio route trap: %+v", r.Topology)
	}
	if r.Topology.RepairCorridor.MutationAllowed {
		t.Fatalf("protected topology allowed mutation: %+v", r.Topology.RepairCorridor)
	}
}

func TestAnalyzeTopologyRequiresInterfaceBoundRaw(t *testing.T) {
	r := routeTrapReport()
	r.Raw = map[string]string{
		"route.en0": "default route interface en0 gateway unreachable",
		"audio.en3": "Dante and MTRX evidence on unrelated interface en3",
	}
	AnalyzeTopology(&r)
	if len(r.Topology.ProtectedConstraints) != 0 {
		t.Fatalf("unbound raw produced constraints: %+v", r.Topology.ProtectedConstraints)
	}

	r.Raw["audio.en0"] = "Dante _netaudio service evidence bound to en0"
	AnalyzeTopology(&r)
	if len(r.Topology.ProtectedConstraints) == 0 || !ProtectedAudioRouteTrap(r) {
		t.Fatalf("bound raw did not produce protected trap: %+v", r.Topology)
	}
}

func TestInternetCandidateDoesNotMatchNDISubstring(t *testing.T) {
	r := routeTrapReport()
	r.Topology.Interfaces = []TopologyInterface{{
		Name:                "en1",
		Roles:               []string{"management", "internet_candidate"},
		RoleEvidence:        []string{"alternate internet candidate over USB LAN"},
		InternetCandidate:   true,
		ManagementCandidate: true,
	}}
	AnalyzeTopology(&r)
	for _, c := range r.Topology.ProtectedConstraints {
		if c.Interface == "en1" {
			t.Fatalf("internet_candidate produced protected constraint on en1: %+v", r.Topology.ProtectedConstraints)
		}
	}
}

func TestNDIWordTokenStillCreatesVideoBoundary(t *testing.T) {
	r := routeTrapReport()
	r.Topology.Interfaces = []TopologyInterface{{
		Name:         "en0",
		Roles:        []string{"protected_video"},
		RoleEvidence: []string{"NDI production VLAN multicast video"},
		Protected:    true,
	}}
	AnalyzeTopology(&r)
	if len(r.Topology.ProtectedConstraints) != 1 {
		t.Fatalf("constraints=%+v", r.Topology.ProtectedConstraints)
	}
	if r.Topology.ProtectedConstraints[0].Class != TopologyClassProtectedVideoVLAN {
		t.Fatalf("class=%s constraints=%+v", r.Topology.ProtectedConstraints[0].Class, r.Topology.ProtectedConstraints)
	}
}

func TestProtectedDefaultWithInternetCandidateDoesNotProtectAlternate(t *testing.T) {
	r := routeTrapReport()
	r.Topology.Interfaces = []TopologyInterface{
		{
			Name:         "en0",
			Roles:        []string{"protected_media"},
			RoleEvidence: []string{"Dante _netaudio production VLAN"},
			Protected:    true,
		},
		{
			Name:                "en1",
			Roles:               []string{"management", "internet_candidate"},
			RoleEvidence:        []string{"alternate internet candidate over USB LAN"},
			InternetCandidate:   true,
			ManagementCandidate: true,
		},
	}
	AnalyzeTopology(&r)
	for _, c := range r.Topology.ProtectedConstraints {
		if c.Interface == "en1" {
			t.Fatalf("alternate internet candidate was marked protected: %+v", r.Topology.ProtectedConstraints)
		}
	}
	if !ProtectedAudioRouteTrap(r) {
		t.Fatalf("protected default route trap was lost: %+v", r.Topology.ProtectedConstraints)
	}
}

func TestMDMProfileCreatesPolicyBoundary(t *testing.T) {
	r := routeTrapReport()
	r.Residues.Profiles = []ResidueMatch{{DisplayName: "Managed Proxy Profile", Risk: "mdm_profile_network_policy", Kind: "profile", DetectOnly: true}}
	AnalyzeTopology(&r)
	if len(r.Topology.ProtectedConstraints) != 1 {
		t.Fatalf("constraints=%+v", r.Topology.ProtectedConstraints)
	}
	if r.Topology.ProtectedConstraints[0].Class != TopologyClassProtectedPolicy {
		t.Fatalf("class=%s", r.Topology.ProtectedConstraints[0].Class)
	}
	if r.Topology.RepairCorridor.MutationAllowed {
		t.Fatal("MDM policy boundary allowed mutation")
	}
}

func routeTrapReport() DiagnosticReport {
	return DiagnosticReport{
		Network: NetworkInfo{
			HardwarePorts: []HardwarePort{{Port: "Ethernet", Device: "en0"}, {Port: "USB LAN", Device: "en1"}},
			Interfaces: []NetworkInterface{
				{Name: "en0", Status: "active", IPv4: []string{"192.168.0.103"}},
				{Name: "en1", Status: "active", IPv4: []string{"192.168.0.104"}},
				{Name: "utun4", Status: "active", IPv4: []string{"198.18.0.1"}, IsUTun: true},
			},
			DefaultRoute: DefaultRoute{Present: true, Gateway: "192.168.0.1", Interface: "en0"},
			ProxySummary: ProxySummary{Dirty: false},
		},
		Reachability: ReachabilityInfo{
			Gateway:      ProbeResult{Target: "192.168.0.1", OK: false},
			RawIPs:       map[string]ProbeResult{"1.1.1.1": {Target: "1.1.1.1", OK: false}, "1.1.1.1%en1": {Target: "1.1.1.1", OK: true, Status: "forced over en1"}},
			DNSNames:     map[string]ProbeResult{"apple.com": {Target: "apple.com", OK: false}},
			HTTPSTargets: map[string]ProbeResult{"https://www.apple.com/": {Target: "https://www.apple.com/", OK: false}},
		},
	}
}
