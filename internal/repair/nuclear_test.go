package repair

import (
	"context"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/snapshot"
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

// TestStaticFallbackGatewayDerivation covers the static-IP fallback used when a
// network has no DHCP router: prefer a known safe gateway, else derive the
// conventional .1 gateway from the active interface, rejecting TUN-hijack ones.
func TestStaticFallbackGatewayDerivation(t *testing.T) {
	known := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{DefaultRoute: diagnose.DefaultRoute{Gateway: "10.0.0.1"}}}
	if g := staticFallbackGateway(known); g != "10.0.0.1" {
		t.Fatalf("known safe gateway not preferred: %q", g)
	}
	staticNet := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{Interfaces: []diagnose.NetworkInterface{
		{Name: "en0", Status: "active", IPv4: []string{"192.168.50.42"}},
	}}}
	if g := staticFallbackGateway(staticNet); g != "192.168.50.1" {
		t.Fatalf("static .1 fallback wrong: %q", g)
	}
	tunTrap := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{
		DefaultRoute: diagnose.DefaultRoute{Gateway: "198.18.0.1"},
		Interfaces: []diagnose.NetworkInterface{
			{Name: "en0", Status: "active", IPv4: []string{"192.168.1.20"}},
			{Name: "utun4", IsUTun: true, IPv4: []string{"198.18.0.1"}},
		},
	}}
	if g := staticFallbackGateway(tunTrap); g != "192.168.1.1" {
		t.Fatalf("TUN gateway must be rejected and utun skipped, got %q", g)
	}
}

// TestSafeDefaultRouteRebuildUsesStaticFallback proves the route rebuild falls
// back to the static gateway (Args[1]) when there is no DHCP router.
func TestSafeDefaultRouteRebuildUsesStaticFallback(t *testing.T) {
	rp, err := snapshot.NewRestorePoint(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/sbin/route -n get default`:              {ExitCode: 1},
		`/usr/sbin/ipconfig getoption en0 router`: {ExitCode: 0, Stdout: "\n"},
		`/sbin/route add default 192.168.1.1`:     {ExitCode: 0},
	}}
	got := runSafeDefaultRouteFromDHCP(context.Background(), runner, &rp,
		action{ID: "clean.route.rebuild.safe_dhcp", Dynamic: "safe-default-route-from-dhcp", Args: []string{"en0", "192.168.1.1"}},
		ActionResult{ID: "clean.route.rebuild.safe_dhcp"})
	if got.Error != "" || !got.RouteAddAttempted || got.RouteAddGateway != "192.168.1.1" {
		t.Fatalf("static fallback gateway not used: %+v calls=%v", got, runner.Calls)
	}
}

// TestNuclearPlanCarriesStaticFallbackGateway proves the nuclear plan embeds the
// derived static-IP gateway so the route can be rebuilt on DHCP-less networks.
func TestNuclearPlanCarriesStaticFallbackGateway(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Wi-Fi"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
			Interfaces:    []diagnose.NetworkInterface{{Name: "en0", Status: "active", IPv4: []string{"192.168.7.33"}}},
		},
	}
	for _, a := range buildActions(LevelNuclear, report) {
		if a.Dynamic == "safe-default-route-from-dhcp" {
			if len(a.Args) < 2 || a.Args[1] != "192.168.7.1" {
				t.Fatalf("nuclear route rebuild missing static fallback gateway: %+v", a.Args)
			}
			return
		}
	}
	t.Fatal("nuclear plan has no route-rebuild action")
}

// TestNuclearRequiresYesForMutation keeps the safety gate: the nuclear button
// will not mutate without explicit consent.
func TestNuclearRequiresYesForMutation(t *testing.T) {
	res := Run(context.Background(), &command.MockRunner{}, Options{Level: LevelNuclear})
	if res.ExitCode == 0 || !strings.Contains(res.Status, "nuclear") {
		t.Fatalf("nuclear without --yes must be refused, got status=%q exit=%d", res.Status, res.ExitCode)
	}
}
