package supportbundle

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/installer"
)

func TestSupportBundleCreatesRedactedZip(t *testing.T) {
	home := t.TempDir()
	out := filepath.Join(home, "bundle.zip")
	catalog := installer.Catalog{}
	rep := Create(context.Background(), &command.MockRunner{}, home, out, catalog)
	if rep.Status != "created" {
		t.Fatalf("status = %s warnings=%v", rep.Status, rep.Warnings)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["offline-readiness.json"] || !names["facts.json"] {
		t.Fatalf("expected readiness and facts in bundle, got %v", names)
	}
}
