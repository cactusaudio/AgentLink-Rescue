package repair

import (
	"context"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
)

// TestNuclearBypassesProtectedTopology proves the connectivity-first override:
// every graduated level refuses to mutate a protected topology, but the nuclear
// button runs anyway — restoring internet outranks preserving an audio VLAN.
func TestNuclearBypassesProtectedTopology(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Classifications: []string{classify.ProtectedAudioVLANRouteTrap},
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Ethernet"}, {Name: "USB LAN"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
		},
		Topology: diagnose.TopologyInfo{Interfaces: []diagnose.TopologyInterface{{
			Name: "en0", Protected: true, Roles: []string{"protected_media"}, RoleEvidence: []string{"Dante audio VLAN"},
		}}},
	}
	if actions := buildActions(LevelCleanBaseline, report); len(actions) != 0 {
		t.Fatalf("clean-baseline must refuse under protected topology, got %d actions", len(actions))
	}
	if actions := buildActions(LevelNuclear, report); len(actions) == 0 {
		t.Fatal("nuclear button must produce actions even under protected topology")
	}
	if planned := plannedActions(LevelNuclear, report); len(planned) == 0 {
		t.Fatal("nuclear plannedActions must produce a plan under protected topology")
	}
}

// TestNuclearPlanCoversOnlineKillers proves the nuclear reset is topologized to
// the extreme for the "NIC up but no internet" scenario: it covers every common
// killer — dirty proxy, stale DNS, stale TUN route, dead-NE claim, missing
// default route — in one universal plan.
func TestNuclearPlanCoversOnlineKillers(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Wi-Fi"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
			ProxySummary:  diagnose.ProxySummary{Dirty: true},
			Interfaces: []diagnose.NetworkInterface{
				{Name: "en0", Status: "active", IPv4: []string{"192.168.1.20"}},
				{Name: "utun4", Status: "active", IsUTun: true, IPv4: []string{"198.18.0.1"}},
			},
		},
	}
	var ids strings.Builder
	for _, a := range buildActions(LevelNuclear, report) {
		ids.WriteString(a.ID)
		ids.WriteByte(' ')
	}
	got := ids.String()
	for _, want := range []string{".webproxy", ".dns", "clean.route.rebuild", "dns.flushcache"} {
		if !strings.Contains(got, want) {
			t.Fatalf("nuclear plan missing killer coverage %q; ids=%s", want, got)
		}
	}
	// TUN/NetworkExtension teardown must be present (release a hijacked route).
	if !strings.Contains(got, "tun.stop") && !strings.Contains(got, "nuclear.ifconfig.down") && !strings.Contains(got, "nuclear.ne.kick") {
		t.Fatalf("nuclear plan missing TUN/NE teardown; ids=%s", got)
	}
}

// TestNuclearSuccessRequiresOnlineCapablePostState proves the reset's success
// is gated on actually getting online: a healthy post-state is "restored",
// while an unchanged-broken post-state is never reported as success.
func TestNuclearSuccessRequiresOnlineCapablePostState(t *testing.T) {
	brokenPre := diagnose.DiagnosticReport{Classifications: []string{classify.NoDefaultRoute, classify.DNSFail}}
	onlinePost := diagnose.DiagnosticReport{Classifications: []string{classify.OK}}
	stillBroken := diagnose.DiagnosticReport{Classifications: []string{classify.NoDefaultRoute, classify.DNSFail}}
	if code, status := compare(brokenPre, onlinePost, 0); code != 0 {
		t.Fatalf("online-capable post-state must be success, got code=%d status=%q", code, status)
	}
	if code, _ := compare(brokenPre, stillBroken, 0); code == 0 {
		t.Fatal("unchanged-broken post-state must not be reported as success")
	}
}

// TestNuclearRequiresYesForMutation keeps the safety gate: the nuclear button
// will not mutate without explicit consent.
func TestNuclearRequiresYesForMutation(t *testing.T) {
	res := Run(context.Background(), &command.MockRunner{}, Options{Level: LevelNuclear})
	if res.ExitCode == 0 || !strings.Contains(res.Status, "nuclear") {
		t.Fatalf("nuclear without --yes must be refused, got status=%q exit=%d", res.Status, res.ExitCode)
	}
}
