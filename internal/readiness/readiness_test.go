package readiness

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/installer"
)

// TestRunProducesReadinessReport exercises the readiness aggregation (facts +
// brain + installer + dev doctors) with a mock runner and asserts a structured,
// checked report without panicking.
func TestRunProducesReadinessReport(t *testing.T) {
	rep := Run(context.Background(), &command.MockRunner{}, t.TempDir(), installer.Catalog{})
	if rep.SchemaVersion != 1 || rep.ToolVersion == "" {
		t.Fatalf("readiness report not populated: %+v", rep)
	}
	if rep.Status == "" {
		t.Fatalf("readiness report missing status: %+v", rep)
	}
	if len(rep.Checks) == 0 {
		t.Fatalf("readiness report produced no checks: %+v", rep)
	}
}
