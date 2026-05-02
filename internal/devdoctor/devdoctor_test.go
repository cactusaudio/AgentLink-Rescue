package devdoctor

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestDevDoctorReportsBaseTools(t *testing.T) {
	report := Run(context.Background(), &command.MockRunner{Results: map[string]command.Result{}}, "test")
	if report.SchemaVersion != 1 {
		t.Fatalf("schema version = %d", report.SchemaVersion)
	}
	if len(report.Tools) == 0 {
		t.Fatal("expected tool checks")
	}
	foundGit := false
	for _, tool := range report.Tools {
		if tool.ID == "git" {
			foundGit = true
		}
	}
	if !foundGit {
		t.Fatal("git check missing")
	}
}
