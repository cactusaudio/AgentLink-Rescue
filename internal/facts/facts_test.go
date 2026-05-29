package facts

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

// TestCollectProducesStructuredFacts exercises fact collection with a mock
// runner and asserts the snapshot is populated and its maps are initialized
// (never nil) so downstream consumers can index safely.
func TestCollectProducesStructuredFacts(t *testing.T) {
	f := Collect(context.Background(), &command.MockRunner{}, t.TempDir(), false)
	if f.SchemaVersion == 0 || f.ToolVersion == "" {
		t.Fatalf("facts not populated: %+v", f)
	}
	if f.ProxyEnv == nil || f.Commands == nil || f.ConfigPaths == nil {
		t.Fatalf("facts maps must be initialized, got proxyEnv=%v commands=%v configPaths=%v", f.ProxyEnv, f.Commands, f.ConfigPaths)
	}
}
