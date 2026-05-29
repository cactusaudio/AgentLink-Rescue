package networkverify

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

// TestRunProducesStructuredReport exercises the full verify path with a mock
// runner (all probes "missing") and asserts a well-formed report comes back
// without panicking and is classified.
func TestRunProducesStructuredReport(t *testing.T) {
	rep := Run(context.Background(), &command.MockRunner{}, "", false)
	if rep.SchemaVersion == 0 || rep.ToolVersion == "" {
		t.Fatalf("report not populated: %+v", rep)
	}
	// With no working network (mock), it must not falsely report OK.
	if rep.OK {
		t.Fatalf("mock/no-network run should not report OK: %+v", rep)
	}
	if len(rep.FailureClasses) == 0 {
		t.Fatalf("broken network must produce failure classes: %+v", rep)
	}
}
