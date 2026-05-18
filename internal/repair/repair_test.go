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
		joined += action.Stage + " " + action.ID + " " + action.Description + " " + strings.Join(action.Args, " ") + "\n"
	}
	for _, want := range []string{"1_stop_runtime_processes tun.stop", "4_kick_network_extension_daemons tun.networkextension.kick", "5_down_stale_utun tun.ifconfig.down.utun0", "6_active_wifi_repair tun.wifi.dns", "tun.route.rebuild.safe_dhcp", "7_dns_airdrop_awdl_repair tun.awdl.up"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in plan:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "route add default 198.18.0.1") || strings.Contains(joined, "Add default route via DHCP gateway") {
		t.Fatalf("TUN plan re-used preflight default route:\n%s", joined)
	}
	if strings.Contains(joined, "standard.location") || strings.Contains(joined, "route.flush") {
		t.Fatalf("TUN plan used broad standard reset:\n%s", joined)
	}
}

func TestTunRepairStageOrder(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Wi-Fi"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
			Interfaces: []diagnose.NetworkInterface{
				{Name: "utun0", IsUTun: true, IPv4: []string{"198.18.0.1"}},
			},
		},
		Residues: diagnose.ResidueInfo{
			LaunchDaemons: []diagnose.ResidueMatch{{Path: "/Library/LaunchDaemons/io.github.clash-verge-rev.plist", Kind: "launchDaemon", RuleID: "clash-daemon", AutoQuarantineAllowed: true}},
			AppSupport:    []diagnose.ResidueMatch{{Path: "/Users/test/Library/Application Support/io.github.clash-verge-rev", Kind: "appSupport", RuleID: "clash-app", AutoQuarantineAllowed: true}},
		},
	}
	var stages []string
	for _, action := range buildTunPreActions(report) {
		stages = append(stages, action.Stage)
	}
	for _, action := range buildQuarantinePlan(report, LevelTun) {
		stages = append(stages, action.Stage)
	}
	for _, action := range buildTunPostActions(report) {
		stages = append(stages, action.Stage)
	}
	joined := strings.Join(stages, "\n")
	order := []string{
		"1_stop_runtime_processes",
		"2_bootout_launch_items",
		"3_quarantine_residue",
		"4_kick_network_extension_daemons",
		"5_down_stale_utun",
		"6_active_wifi_repair",
		"7_dns_airdrop_awdl_repair",
	}
	last := -1
	for _, stage := range order {
		idx := strings.Index(joined, stage)
		if idx < 0 {
			t.Fatalf("missing stage %s in:\n%s", stage, joined)
		}
		if idx < last {
			t.Fatalf("stage %s appeared out of order in:\n%s", stage, joined)
		}
		last = idx
	}
}

func TestSafeDHCPGatewayRejectsTunHijackGateways(t *testing.T) {
	for _, gateway := range []string{"", "198.18.0.1", "198.19.255.254", "fdfe:dcba:9876::1", "not-an-ip"} {
		if safeDHCPGateway(gateway) {
			t.Fatalf("gateway %q should be refused", gateway)
		}
	}
	for _, gateway := range []string{"192.168.1.1", "10.0.0.1", "fe80::1"} {
		if !safeDHCPGateway(gateway) {
			t.Fatalf("gateway %q should be accepted", gateway)
		}
	}
}

func TestRouteOutputSuspiciousDetectsUtunGateway(t *testing.T) {
	if !routeOutputSuspicious("gateway: 198.18.0.1\ninterface: utun0\n") {
		t.Fatal("did not detect suspicious utun route output")
	}
	if routeOutputSuspicious("gateway: 192.168.1.1\ninterface: en0\n") {
		t.Fatal("safe Wi-Fi route output marked suspicious")
	}
}

func TestStandardDoesNotUseBroadSystemReset(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Wi-Fi"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
			DefaultRoute:  diagnose.DefaultRoute{Present: false},
		},
	}
	standard := buildActions(LevelStandard, report)
	for _, action := range standard {
		if strings.Contains(action.ID, "location.") || strings.Contains(action.ID, "route.flush") {
			t.Fatalf("standard used broad reset action: %+v", action)
		}
	}
	systemReset := buildActions(LevelStandardSystemReset, report)
	foundLocation := false
	for _, action := range systemReset {
		if strings.Contains(action.ID, "location.create") {
			foundLocation = true
		}
	}
	if !foundLocation {
		t.Fatal("standard-system-reset did not create clean location")
	}
}

func TestProtectedAudioVLANTrapPlansNoMutation(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Classifications: []string{classify.ProtectedAudioVLANRouteTrap},
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Ethernet"}, {Name: "USB LAN"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Ethernet", Device: "en0"}, {Port: "USB LAN", Device: "en1"}},
			DefaultRoute:  diagnose.DefaultRoute{Present: true, Gateway: "192.168.0.1", Interface: "en0"},
		},
	}
	for _, level := range []string{LevelSafe, LevelStandard, LevelTun, LevelCleanBaseline, LevelStandardSystemReset, LevelDeep} {
		if actions := buildActions(level, report); len(actions) != 0 {
			t.Fatalf("protected route trap level %s produced mutating actions: %+v", level, actions)
		}
	}
}

func TestProtectedTopologyConstraintPlansNoMutationWithoutFailureClass(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Ethernet"}, {Name: "USB LAN"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Ethernet", Device: "en0"}, {Port: "USB LAN", Device: "en1"}},
			Interfaces:    []diagnose.NetworkInterface{{Name: "en0", Status: "active", IPv4: []string{"192.168.0.103"}}, {Name: "en1", Status: "active", IPv4: []string{"192.168.0.104"}}},
			DefaultRoute:  diagnose.DefaultRoute{Present: true, Gateway: "192.168.0.1", Interface: "en0"},
		},
		Reachability: diagnose.ReachabilityInfo{Gateway: diagnose.ProbeResult{Target: "192.168.0.1", OK: false}},
		Topology: diagnose.TopologyInfo{Interfaces: []diagnose.TopologyInterface{{
			Name: "en0", Protected: true, Roles: []string{"protected_media"}, RoleEvidence: []string{"Dante audio VLAN"},
		}}},
	}
	for _, level := range []string{LevelSafe, LevelStandard, LevelTun, LevelCleanBaseline, LevelStandardSystemReset, LevelDeep} {
		if actions := plannedActions(level, report); len(actions) != 0 {
			t.Fatalf("protected topology level %s produced planned actions: %+v", level, actions)
		}
	}
}

func TestCleanBaselineRequiresYesForMutation(t *testing.T) {
	result := Run(context.Background(), &command.MockRunner{}, Options{Level: LevelCleanBaseline})
	if result.ExitCode != 50 || !strings.Contains(result.Error, "requires --yes") {
		t.Fatalf("clean-baseline without --yes got exit=%d err=%q", result.ExitCode, result.Error)
	}
}

func TestCleanBaselinePlanIsLastResortButNotDeepReset(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services:      []diagnose.NetworkService{{Name: "Wi-Fi"}},
			HardwarePorts: []diagnose.HardwarePort{{Port: "Wi-Fi", Device: "en0"}},
		},
	}
	actions := buildActions(LevelCleanBaseline, report)
	joined := ""
	for _, action := range actions {
		joined += action.Stage + " " + action.ID + " " + action.Description + " " + strings.Join(action.Args, " ") + "\n"
	}
	for _, want := range []string{"clean.stop", "clean.location.automatic", "clean.service.wi_fi.webproxy", "clean.ipconfig.dhcp.en0", "clean.route.rebuild.safe_dhcp", "clean.awdl.up"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing clean baseline action %s in:\n%s", want, joined)
		}
	}
	for _, forbidden := range []string{"route.flush", "deep.quarantine", "enable", "tun on"} {
		if strings.Contains(strings.ToLower(joined), forbidden) {
			t.Fatalf("clean baseline contains forbidden broad/proxy-enabling action %q:\n%s", forbidden, joined)
		}
	}
}

func TestCriticalWorsenedDefaultRouteTriggersRollback(t *testing.T) {
	pre := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{DefaultRoute: diagnose.DefaultRoute{Present: true}}, Reachability: diagnose.ReachabilityInfo{RawIPs: map[string]diagnose.ProbeResult{"1.1.1.1": {OK: true}}}}
	post := diagnose.DiagnosticReport{Network: diagnose.NetworkInfo{DefaultRoute: diagnose.DefaultRoute{Present: false}}, Reachability: diagnose.ReachabilityInfo{RawIPs: map[string]diagnose.ProbeResult{"1.1.1.1": {OK: true}}}}
	if !criticalWorsened(pre, post) {
		t.Fatal("default route loss did not trigger worsened comparator")
	}
}

func TestNetworkRollbackRestoresSafeState(t *testing.T) {
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/usr/sbin/networksetup -setdnsservers Wi-Fi 1.1.1.1 8.8.8.8`:  {ExitCode: 0},
		`/usr/sbin/networksetup -setsearchdomains Wi-Fi lan`:           {ExitCode: 0},
		`/usr/sbin/networksetup -setwebproxystate Wi-Fi off`:           {ExitCode: 0},
		`/usr/sbin/networksetup -setsecurewebproxystate Wi-Fi off`:     {ExitCode: 0},
		`/usr/sbin/networksetup -setsocksfirewallproxystate Wi-Fi off`: {ExitCode: 0},
		`/usr/sbin/networksetup -setautoproxystate Wi-Fi off`:          {ExitCode: 0},
		`/sbin/route delete default`:                                   {ExitCode: 0},
		`/sbin/route add default 192.168.1.1`:                          {ExitCode: 0},
		`/sbin/ifconfig awdl0 up`:                                      {ExitCode: 0},
	}}
	var result Result
	status := restoreNetworkSnapshot(context.Background(), runner, NetworkStateSnapshot{
		WiFiService:           "Wi-Fi",
		DNSServers:            []string{"1.1.1.1", "8.8.8.8"},
		SearchDomains:         []string{"lan"},
		DefaultRouteGateway:   "192.168.1.1",
		DefaultRouteInterface: "en0",
		AWDLUp:                true,
	}, &result)
	if status != "restored" {
		t.Fatalf("network rollback status %q, actions=%+v warnings=%+v", status, result.NetworkRollbackActions, result.PartialRollbackWarnings)
	}
}

func TestNetworkRollbackRefusesSuspiciousTunGateway(t *testing.T) {
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/usr/sbin/networksetup -setdnsservers Wi-Fi empty`:            {ExitCode: 0},
		`/usr/sbin/networksetup -setsearchdomains Wi-Fi empty`:         {ExitCode: 0},
		`/usr/sbin/networksetup -setwebproxystate Wi-Fi off`:           {ExitCode: 0},
		`/usr/sbin/networksetup -setsecurewebproxystate Wi-Fi off`:     {ExitCode: 0},
		`/usr/sbin/networksetup -setsocksfirewallproxystate Wi-Fi off`: {ExitCode: 0},
		`/usr/sbin/networksetup -setautoproxystate Wi-Fi off`:          {ExitCode: 0},
	}}
	var result Result
	status := restoreNetworkSnapshot(context.Background(), runner, NetworkStateSnapshot{
		WiFiService:           "Wi-Fi",
		DefaultRouteGateway:   "198.18.0.1",
		DefaultRouteInterface: "utun0",
	}, &result)
	if status != "partial_rollback_failed" {
		t.Fatalf("suspicious gateway rollback status %q, want partial_rollback_failed", status)
	}
	for _, call := range runner.Calls {
		if strings.Contains(call, "route add default 198.18.0.1") {
			t.Fatalf("rollback reinstalled suspicious gateway: %v", runner.Calls)
		}
	}
}

func TestSafeDefaultRouteRebuildSkipsWithoutSafeGateway(t *testing.T) {
	rp, err := snapshot.NewRestorePoint(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/sbin/route -n get default`:              {ExitCode: 1},
		`/usr/sbin/ipconfig getoption en0 router`: {ExitCode: 0, Stdout: "198.18.0.1\n"},
	}}
	got := runSafeDefaultRouteFromDHCP(context.Background(), runner, &rp, action{ID: "tun.route.rebuild.safe_dhcp", Dynamic: "safe-default-route-from-dhcp", Args: []string{"en0"}}, ActionResult{ID: "tun.route.rebuild.safe_dhcp"})
	if got.Error != "" || got.SkippedReason != "route_rebuild_skipped_no_safe_gateway" || got.RouteDeleteAttempted || got.RouteAddAttempted {
		t.Fatalf("unexpected route rebuild result: %+v calls=%v", got, runner.Calls)
	}
	for _, call := range runner.Calls {
		if strings.Contains(call, "route delete default") || strings.Contains(call, "route add default") {
			t.Fatalf("route mutated despite unsafe gateway: %v", runner.Calls)
		}
	}
}

func TestSafeDefaultRouteRebuildFailsIfDeleteThenAddFails(t *testing.T) {
	rp, err := snapshot.NewRestorePoint(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/sbin/route -n get default`:              {ExitCode: 0, Stdout: "gateway: 198.18.0.1\ninterface: utun0\n"},
		`/usr/sbin/ipconfig getoption en0 router`: {ExitCode: 0, Stdout: "192.168.1.1\n"},
		`/sbin/route delete default`:              {ExitCode: 0},
		`/sbin/route add default 192.168.1.1`:     {ExitCode: 1, Stderr: "add failed"},
	}}
	got := runSafeDefaultRouteFromDHCP(context.Background(), runner, &rp, action{ID: "tun.route.rebuild.safe_dhcp", Dynamic: "safe-default-route-from-dhcp", Args: []string{"en0"}}, ActionResult{ID: "tun.route.rebuild.safe_dhcp"})
	if got.Error == "" || !got.RouteDeleteAttempted || !got.RouteAddAttempted || got.RouteAddGateway != "192.168.1.1" {
		t.Fatalf("route add failure was not reported as fatal: %+v calls=%v", got, runner.Calls)
	}
}

func TestSafeDefaultRouteRebuildAddsSafeDHCPGateway(t *testing.T) {
	rp, err := snapshot.NewRestorePoint(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/sbin/route -n get default`:              {ExitCode: 1},
		`/usr/sbin/ipconfig getoption en0 router`: {ExitCode: 0, Stdout: "192.168.1.1\n"},
		`/sbin/route add default 192.168.1.1`:     {ExitCode: 0},
	}}
	got := runSafeDefaultRouteFromDHCP(context.Background(), runner, &rp, action{ID: "tun.route.rebuild.safe_dhcp", Dynamic: "safe-default-route-from-dhcp", Args: []string{"en0"}}, ActionResult{ID: "tun.route.rebuild.safe_dhcp"})
	if got.Error != "" || !got.RouteAddAttempted || got.RouteAddGateway != "192.168.1.1" {
		t.Fatalf("safe DHCP gateway was not used correctly: %+v calls=%v", got, runner.Calls)
	}
}
