package diagnose

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParseScutilProxyCleanDirty(t *testing.T) {
	clean := ParseScutilProxy(fixture(t, "scutil_proxy_clean.txt"))
	if clean.Dirty || clean.HTTPEnabled || clean.HTTPSEnabled || clean.SOCKSEnabled || clean.PACEnabled {
		t.Fatalf("expected clean proxy, got %+v", clean)
	}
	dirty := ParseScutilProxy(fixture(t, "scutil_proxy_dirty.txt"))
	if !dirty.Dirty || !dirty.HTTPSEnabled {
		t.Fatalf("expected dirty HTTPS proxy, got %+v", dirty)
	}
}

func TestParseNetworkServices(t *testing.T) {
	got := ParseNetworkServices("An asterisk (*) denotes that a network service is disabled.\nWi-Fi\n*Thunderbolt Bridge\nUSB 10/100/1000 LAN\n")
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Name != "Wi-Fi" || got[0].Disabled {
		t.Fatalf("bad first service: %+v", got[0])
	}
	if got[1].Name != "Thunderbolt Bridge" || !got[1].Disabled {
		t.Fatalf("bad disabled service: %+v", got[1])
	}
}

func TestParseHardwarePorts(t *testing.T) {
	raw := "Hardware Port: Wi-Fi\nDevice: en0\nEthernet Address: aa:bb:cc:dd:ee:ff\n\nHardware Port: Thunderbolt Bridge\nDevice: bridge0\nEthernet Address: none\n"
	got := ParseHardwarePorts(raw)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Port != "Wi-Fi" || got[0].Device != "en0" {
		t.Fatalf("bad port: %+v", got[0])
	}
}

func TestParseRouteDefault(t *testing.T) {
	ok := ParseRouteDefault(fixture(t, "route_default_ok.txt"), "")
	if !ok.Present || ok.Gateway != "192.168.1.1" || ok.Interface != "en0" {
		t.Fatalf("bad route: %+v", ok)
	}
	missing := ParseRouteDefault(fixture(t, "route_default_missing.txt"), "")
	if missing.Present {
		t.Fatalf("expected missing route: %+v", missing)
	}
}
