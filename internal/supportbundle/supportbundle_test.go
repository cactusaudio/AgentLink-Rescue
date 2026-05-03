package supportbundle

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	var combined strings.Builder
	for _, f := range zr.File {
		names[f.Name] = true
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		combined.Write(data)
	}
	if !names["offline-readiness.json"] || !names["facts.json"] || !names["support-bundle-manifest.json"] {
		t.Fatalf("expected readiness and facts in bundle, got %v", names)
	}
	text := combined.String()
	for _, want := range []string{"tool presence", "local paths", "network/proxy status", "readiness reports", "latest session metadata"} {
		if !strings.Contains(text, want) {
			t.Fatalf("manifest missing category %q", want)
		}
	}
	for _, forbidden := range []string{"private key material", "browser cookies content", "shell history content", "Wi-Fi password value"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("bundle contains forbidden disclosure payload %q", forbidden)
		}
	}
}
