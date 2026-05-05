package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigureLocalGemmaStatusHonesty(t *testing.T) {
	home := t.TempDir()
	rep := ConfigureLocalGemma(home, true)
	if rep.Status != "template_generated" {
		t.Fatalf("generated template reported %q, want template_generated", rep.Status)
	}
	home = t.TempDir()
	path := configPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	rep = ConfigureLocalGemma(home, true)
	if rep.Status != "manual_merge_required" {
		t.Fatalf("existing config reported %q, want manual_merge_required", rep.Status)
	}
}

func TestInstallPluginDryRunIsScaffoldStatus(t *testing.T) {
	rep := InstallPluginDryRun()
	if rep.Status != "scaffold_available" {
		t.Fatalf("plugin scaffold reported %q", rep.Status)
	}
}
