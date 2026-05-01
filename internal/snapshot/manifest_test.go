package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

func TestRestorePointPathGeneration(t *testing.T) {
	if got := GenerateID(time.Date(2026, 5, 1, 2, 14, 13, 0, time.UTC)); got != "20260501-021413" {
		t.Fatalf("id=%s", got)
	}
}

func TestManifestCreateUpdate(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(home, "Library", "LaunchAgents", "source.plist")
	if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
		t.Fatal(err)
	}
	rp, err := NewRestorePointWithPolicy(filepath.Join(dir, "restore-points"), "test", system.MutationOptions{RealUserHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if rp.Manifest.ID == "" {
		t.Fatal("missing id")
	}
	if err := os.WriteFile(source, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := rp.BackupPath(source); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(rp.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Manifest.Entries) != 1 || !loaded.Manifest.Entries[0].RollbackAvailable {
		t.Fatalf("bad entries: %+v", loaded.Manifest.Entries)
	}
}

func TestQuarantinePathSingleManifestEntryAndRestore(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(home, "Library", "LaunchAgents", "com.example.agent.plist")
	if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("plist"), 0644); err != nil {
		t.Fatal(err)
	}
	rp, err := NewRestorePointWithPolicy(filepath.Join(dir, "restore-points"), "test", system.MutationOptions{RealUserHome: home})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := rp.QuarantinePath(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(rp.Manifest.Entries) != 1 {
		t.Fatalf("entries=%d", len(rp.Manifest.Entries))
	}
	if entry.BackupPath == "" || entry.QuarantinePath == "" {
		t.Fatalf("entry missing backup or quarantine path: %+v", entry)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source should have moved to quarantine, err=%v", err)
	}
	if err := rp.RestoreAll(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "plist" {
		t.Fatalf("restored data=%q", data)
	}
}

func TestRestoreAllRefusesToOverwriteExistingQuarantineOriginal(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	source := filepath.Join(home, "Library", "LaunchAgents", "com.example.agent.plist")
	if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	rp, err := NewRestorePointWithPolicy(filepath.Join(dir, "restore-points"), "test", system.MutationOptions{RealUserHome: home})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rp.QuarantinePath(source); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := rp.RestoreAll(); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected overwrite refusal, got %v", err)
	}
}

func TestRestoreConfigEntriesRestoresExistedAndUnsetsMissing(t *testing.T) {
	dir := t.TempDir()
	rp, err := NewRestorePointWithPolicy(filepath.Join(dir, "restore-points"), "test", system.MutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := rp.RecordConfigValue("git", "global", "http.proxy", true, "http://old.example:7890"); err != nil {
		t.Fatal(err)
	}
	if err := rp.RecordConfigValue("npm", "user", "proxy", false, ""); err != nil {
		t.Fatal(err)
	}
	gitPath := restoreConfigToolPath("git")
	npmPath := restoreConfigToolPath("npm")
	runner := &command.MockRunner{Results: map[string]command.Result{
		command.Render(npmPath, "config", "delete", "proxy"):                                   {ExitCode: 0},
		command.Render(gitPath, "config", "--global", "http.proxy", "http://old.example:7890"): {ExitCode: 0},
	}}
	if err := rp.RestoreConfigEntries(context.Background(), runner); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(runner.Calls, "\n")
	if !strings.Contains(joined, command.Render(gitPath, "config", "--global", "http.proxy", "http://old.example:7890")) {
		t.Fatalf("git restore missing: %s", joined)
	}
	if !strings.Contains(joined, command.Render(npmPath, "config", "delete", "proxy")) {
		t.Fatalf("npm delete missing: %s", joined)
	}
}
