package lastgood

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestLastGoodSaveAndRestoreUserConfig(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("model = \"old\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	save := Save(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test"})
	if save.Status != "saved" {
		t.Fatalf("save status = %s warnings=%v", save.Status, save.Warnings)
	}
	if err := os.WriteFile(cfg, []byte("model = \"broken\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	restore := Restore(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test", Last: true, Yes: true})
	if restore.Status != "restored" {
		t.Fatalf("restore status = %s warnings=%v", restore.Status, restore.Warnings)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "model = \"old\"\n" {
		t.Fatalf("config not restored: %q", data)
	}
}
