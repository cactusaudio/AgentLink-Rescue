package ticket

import (
	"context"
	"os"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestCreateClashTunTicket(t *testing.T) {
	home := t.TempDir()
	packageRoot := "/tmp/Cactus MacBook Network Rescue/Cactus AgentLink Rescue.app/Contents/Resources/agentlink"
	rep := Create(context.Background(), &command.MockRunner{}, Options{Home: home, Type: "clash-tun-fix", Version: "test", PackageRoot: packageRoot})
	if rep.Status != "created" {
		t.Fatalf("ticket failed: %+v", rep)
	}
	path := rep.Files["Run-Clash-TUN-Fix.command"]
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "sudo ./bin/agentlink rescue --level tun --yes") {
		t.Fatalf("ticket missing tun repair command:\n%s", text)
	}
	if !strings.Contains(text, "cd '"+packageRoot+"'") {
		t.Fatalf("ticket did not use explicit quoted package root:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "password") && strings.Contains(text, "sudo -S") {
		t.Fatalf("ticket contains unsafe password handling:\n%s", text)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("ticket command is not executable: %v", info.Mode())
	}
}
