package lastgood

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

func TestLastGoodSaveAndRestoreUserConfig(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("model = \"old\"\n"), 0600); err != nil {
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
	info, err := os.Stat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %o, want 0600", got)
	}
}

func TestLastGoodRestorePreservesOriginalMode0640(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("mode = \"0640\"\n"), 0640); err != nil {
		t.Fatal(err)
	}
	save := Save(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test"})
	if save.Status != "saved" {
		t.Fatalf("save status = %s warnings=%v", save.Status, save.Warnings)
	}
	if err := os.WriteFile(cfg, []byte("broken\n"), 0600); err != nil {
		t.Fatal(err)
	}
	restore := Restore(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test", Last: true, Yes: true})
	if restore.Status != "restored" {
		t.Fatalf("restore status = %s warnings=%v", restore.Status, restore.Warnings)
	}
	info, err := os.Stat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0640 {
		t.Fatalf("mode = %o, want 0640", got)
	}
}

func TestLastGoodRestoreDefaultsMissingModeTo0600(t *testing.T) {
	home := t.TempDir()
	profile := writeManualProfile(t, home, []SavedItem{manualItem(t, home, filepath.Join(home, ".codex", "config.toml"), "model = \"safe\"\n", "")})
	restore := Restore(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test", ID: profile, Yes: true})
	if restore.Status != "restored" {
		t.Fatalf("restore status = %s warnings=%v", restore.Status, restore.Warnings)
	}
	info, err := os.Stat(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %o, want 0600", got)
	}
}

func TestLastGoodRestoreRefusesOutOfPolicyManifestPaths(t *testing.T) {
	home := t.TempDir()
	valid := manualItem(t, home, filepath.Join(home, ".codex", "config.toml"), "model = \"safe\"\n", "0600")
	badTmp := manualItem(t, home, "/tmp/evil", "evil\n", "0600")
	badSSH := manualItem(t, home, filepath.Join(home, ".ssh", "config"), "Host *\n", "0600")
	traversal := manualItem(t, home, filepath.Join(home, "..", "somewhere"), "bad\n", "0600")
	profile := writeManualProfile(t, home, []SavedItem{valid, badTmp, badSSH, traversal})
	restore := Restore(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test", ID: profile, Yes: true})
	if restore.Status != "restored" {
		t.Fatalf("restore status = %s warnings=%v", restore.Status, restore.Warnings)
	}
	if len(restore.Warnings) < 3 {
		t.Fatalf("expected skipped warnings, got %v", restore.Warnings)
	}
	for _, warning := range restore.Warnings {
		if strings.Contains(warning, "private") {
			t.Fatalf("unexpected warning: %s", warning)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); err != nil {
		t.Fatal("valid codex config was not restored")
	}
}

func TestLastGoodRestoreRefusesStoredPathOutsideProfile(t *testing.T) {
	home := t.TempDir()
	outside := filepath.Join(home, "outside-copy")
	if err := os.WriteFile(outside, []byte("bad\n"), 0600); err != nil {
		t.Fatal(err)
	}
	item := SavedItem{ID: "codex-config", Kind: "config", OriginalPath: filepath.Join(home, ".codex", "config.toml"), StoredPath: outside, Exists: true, Restorable: true, Mode: "0600"}
	profile := writeManualProfile(t, home, []SavedItem{item})
	restore := Restore(context.Background(), &command.MockRunner{}, Options{Home: home, Version: "test", ID: profile, Yes: true})
	if restore.Status != "failed" {
		t.Fatalf("status = %s warnings=%v", restore.Status, restore.Warnings)
	}
	if len(restore.Warnings) == 0 || !strings.Contains(strings.Join(restore.Warnings, "\n"), "stored path outside selected profile") {
		t.Fatalf("missing stored path warning: %v", restore.Warnings)
	}
}

func manualItem(t *testing.T, home, original, content, mode string) SavedItem {
	t.Helper()
	id := "manual-" + strings.ReplaceAll(filepath.Base(original), ".", "-")
	stored := filepath.Join(system.UserLastGoodDir(home), "manual", "files", id)
	if err := os.MkdirAll(filepath.Dir(stored), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stored, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return SavedItem{ID: id, Kind: "config", OriginalPath: original, StoredPath: stored, Exists: true, Restorable: true, Mode: mode}
}

func writeManualProfile(t *testing.T, home string, items []SavedItem) string {
	t.Helper()
	id := "manual"
	root := filepath.Join(system.UserLastGoodDir(home), id)
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	m := Manifest{SchemaVersion: 1, ToolVersion: "test", ID: id, Home: home, Items: items}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return id
}
