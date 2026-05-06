package packagehealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestDoctorIsReadOnlyForMissingSupportDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := packageRootFixture(t)
	rep := Doctor(context.Background(), &command.MockRunner{}, Options{PackageRoot: root})
	if rep.Status != "needs_repair" {
		t.Fatalf("expected needs_repair for missing support dirs, got %s: %+v", rep.Status, rep.Checks)
	}
	if _, err := os.Stat(filepath.Join(home, "Library", "Application Support")); !os.IsNotExist(err) {
		t.Fatalf("doctor created support dirs or got unexpected error: %v", err)
	}
}

func TestRepairFixesExecutableBitAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := packageRootFixture(t)
	bin := filepath.Join(root, "bin", "agentlink")
	if err := os.Chmod(bin, 0644); err != nil {
		t.Fatal(err)
	}
	rep := Repair(context.Background(), &command.MockRunner{}, Options{PackageRoot: root, Yes: true})
	if rep.Status != "repaired" && rep.Status != "warning" {
		t.Fatalf("unexpected repair status: %s %+v", rep.Status, rep.Warnings)
	}
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("executable bit was not restored: %v", info.Mode())
	}
	again := Repair(context.Background(), &command.MockRunner{}, Options{PackageRoot: root, Yes: true})
	if again.Status != "repaired" && again.Status != "warning" {
		t.Fatalf("second repair should be idempotent, got %s", again.Status)
	}
}

func TestDoctorMissingRequiredBinFailsClearly(t *testing.T) {
	root := packageRootFixture(t)
	if err := os.Remove(filepath.Join(root, "bin", "agentlink")); err != nil {
		t.Fatal(err)
	}
	rep := Doctor(context.Background(), &command.MockRunner{}, Options{PackageRoot: root})
	if rep.Status != "broken" {
		t.Fatalf("expected broken, got %s", rep.Status)
	}
	found := false
	for _, check := range rep.Checks {
		if check.ID == "embedded_agentlink" && check.Status == "missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing bin check not reported: %+v", rep.Checks)
	}
}

func packageRootFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "bin"))
	mustMkdir(t, filepath.Join(root, "recipes"))
	mustMkdir(t, filepath.Join(root, "rules"))
	mustMkdir(t, filepath.Join(root, "docs", "offline"))
	mustMkdir(t, filepath.Join(root, "assets", "manifests"))
	mustWrite(t, filepath.Join(root, "bin", "agentlink"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(root, "rescue.sh"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(root, "agentlink.command"), "#!/bin/sh\n")
	return root
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0755); err != nil {
		t.Fatal(err)
	}
}
