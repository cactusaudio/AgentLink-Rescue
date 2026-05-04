package proxyapp

import (
	"context"
	"strings"
	"testing"
)

func TestCleanReinstallDryRunIsFinalResort(t *testing.T) {
	rep := CleanReinstall(context.Background(), nil, t.TempDir(), "clash-verge-rev", false)
	if rep.Status != "dry_run" {
		t.Fatalf("status = %s", rep.Status)
	}
	joined := strings.Join(append(rep.Actions, rep.Warnings...), "\n")
	if !strings.Contains(joined, "Final resort") {
		t.Fatalf("missing final resort warning: %s", joined)
	}
	if !strings.Contains(joined, "will not enable Clash proxy or TUN automatically") {
		t.Fatalf("missing no auto-enable warning: %s", joined)
	}
}

func TestCleanReinstallRejectsUnknownApp(t *testing.T) {
	rep := CleanReinstall(context.Background(), nil, t.TempDir(), "random-proxy", false)
	if rep.Status != "failed" {
		t.Fatalf("status = %s", rep.Status)
	}
}
