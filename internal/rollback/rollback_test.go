package rollback

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/repair"
	"cactus-agentlink-rescue/internal/snapshot"
)

func TestLoadNetworkRollbackStateFromRestorePoint(t *testing.T) {
	rp, err := snapshot.NewRestorePoint(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	snap := repair.NetworkStateSnapshot{
		WiFiService:           "Wi-Fi",
		DNSServers:            []string{"1.1.1.1"},
		DefaultRouteGateway:   "192.168.1.1",
		DefaultRouteInterface: "en0",
	}
	mutations := repair.NetworkMutations{DNSChanged: true, DefaultRouteChanged: true, Actions: []string{"tun.wifi.dns", "tun.route.rebuild.safe_dhcp"}}
	if _, err := rp.WriteJSON("network-preflight.json", snap); err != nil {
		t.Fatal(err)
	}
	if _, err := rp.WriteJSON("network-mutations.json", mutations); err != nil {
		t.Fatal(err)
	}
	loadedSnap, loadedMutations, ok, warnings := loadNetworkRollbackState(rp)
	if !ok || len(warnings) != 0 {
		t.Fatalf("network rollback state not loaded: ok=%v warnings=%v", ok, warnings)
	}
	if loadedSnap.WiFiService != "Wi-Fi" || loadedSnap.DefaultRouteGateway != "192.168.1.1" {
		t.Fatalf("bad snapshot: %+v", loadedSnap)
	}
	if !loadedMutations.DNSChanged || !loadedMutations.DefaultRouteChanged || loadedMutations.ProxyChanged {
		t.Fatalf("bad mutations: %+v", loadedMutations)
	}
	runner := &command.MockRunner{Results: map[string]command.Result{
		`/usr/sbin/networksetup -setdnsservers Wi-Fi 1.1.1.1`: {ExitCode: 0},
		`/sbin/route delete default`:                          {ExitCode: 0},
		`/sbin/route add default 192.168.1.1`:                 {ExitCode: 0},
	}}
	restored := repair.RestoreNetworkSnapshot(context.Background(), runner, loadedSnap, loadedMutations)
	if restored.Status != "restored" {
		t.Fatalf("loaded network rollback state did not restore: %+v calls=%v", restored, runner.Calls)
	}
}

func TestLoadNetworkRollbackStateRefusesPathOutsideRestorePoint(t *testing.T) {
	rp, err := snapshot.NewRestorePoint(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "network-preflight.json")
	data, _ := json.Marshal(repair.NetworkStateSnapshot{WiFiService: "Wi-Fi"})
	if err := os.WriteFile(outside, data, 0600); err != nil {
		t.Fatal(err)
	}
	rp.Manifest.NetworkSnapshotPath = outside
	_, _, ok, warnings := loadNetworkRollbackState(rp)
	if ok || len(warnings) == 0 {
		t.Fatalf("outside network snapshot was accepted: ok=%v warnings=%v", ok, warnings)
	}
}
