package repair

import (
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/diagnose"
)

func TestBuildActionsSkipsDisabledServices(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services: []diagnose.NetworkService{
				{Name: "Wi-Fi"},
				{Name: "Thunderbolt Bridge", Disabled: true},
			},
		},
	}
	actions := buildActions(LevelSafe, report)
	for _, action := range actions {
		if strings.Contains(action.Description, "Thunderbolt Bridge") {
			t.Fatalf("disabled service produced action: %+v", action)
		}
		for _, arg := range action.Args {
			if arg == "Thunderbolt Bridge" {
				t.Fatalf("disabled service produced command args: %+v", action)
			}
		}
	}
}

func TestFormatRollbackCommandUsesAbsoluteQuotedExecutable(t *testing.T) {
	got := FormatRollbackCommand("/tmp/Cactus AgentLink Rescue/bin/agentlink", "20260501-021413")
	want := `sudo "/tmp/Cactus AgentLink Rescue/bin/agentlink" rollback --id "20260501-021413"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTunRepairPlanIsTargeted(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Wi-Fi"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
			DefaultRoute:  diagnose.DefaultRoute{Present: true, Gateway: "192.168.1.1", Interface: "en0"},
			Interfaces: []diagnose.NetworkInterface{
				{Name: "en0", Status: "active", IPv4: []string{"192.168.1.20"}},
				{Name: "utun0", IsUTun: true, IPv4: []string{"198.18.0.1"}, IPv6: []string{"fdfe:dcba:9876::1"}},
			},
		},
	}
	actions := buildActions(LevelTun, report)
	joined := ""
	for _, action := range actions {
		joined += action.ID + " " + action.Description + "\n"
	}
	for _, want := range []string{"tun.stop", "tun.networkextension.kick", "tun.ifconfig.down.utun0", "tun.wifi.dns", "tun.route.add.default", "tun.awdl.up"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in plan:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "standard.location") || strings.Contains(joined, "route.flush") {
		t.Fatalf("TUN plan used broad standard reset:\n%s", joined)
	}
}

func TestCriticalWorsenedDefaultRouteTriggersRollback(t *testing.T) {
	pre := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{DefaultRoute: diagnose.DefaultRoute{Present: true}}, Reachability: diagnose.ReachabilityInfo{RawIPs: map[string]diagnose.ProbeResult{"1.1.1.1": {OK: true}}}}
	post := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{DefaultRoute: diagnose.DefaultRoute{Present: false}}, Reachability: diagnose.ReachabilityInfo{RawIPs: map[string]diagnose.ProbeResult{"1.1.1.1": {OK: true}}}}
	if !criticalWorsened(pre, post) {
		t.Fatal("default route loss did not trigger worsened comparator")
	}
}
