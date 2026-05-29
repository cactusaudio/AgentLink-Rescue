package verifier

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cactus-agentlink-rescue/internal/configfile"
)

// TestVerifierDeltaManagedSectionFailThenPass is the verifier-delta harness:
// it proves a verifier genuinely OWNS truth by discriminating the pre-repair
// state (managed block absent -> fail) from the post-repair state (block
// present -> pass). A verifier that cannot flip on a real state change would
// rubber-stamp any repair.
func TestVerifierDeltaManagedSectionFailThenPass(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("export PATH=/usr/bin\n"), 0o644); err != nil {
		t.Fatalf("seed zshrc: %v", err)
	}
	reg := NewRegistry()
	ctx := Context{Context: context.Background(), Home: home}
	args := map[string]string{"path": "~/.zshrc", "marker": "PROXY_CLEAN_BLOCK"}

	if r := reg.Run(ctx, "managed_section_present", args); r.Status != "fail" {
		t.Fatalf("pristine state (no managed block) must fail, got %q", r.Status)
	}
	// Simulate the repair the recipe would apply.
	configfile.AppendManagedBlockIfMissing(zshrc, "PROXY_CLEAN_BLOCK", "unset HTTP_PROXY HTTPS_PROXY ALL_PROXY")
	if r := reg.Run(ctx, "managed_section_present", args); r.Status != "pass" {
		t.Fatalf("repaired state (managed block present) must pass, got %q", r.Status)
	}
}

// TestVerifierDeltaUnknownVerifierFails ensures an unknown verifier id is a
// hard fail, never a silent pass.
func TestVerifierDeltaUnknownVerifierFails(t *testing.T) {
	reg := NewRegistry()
	r := reg.Run(Context{Context: context.Background()}, "no_such_verifier", nil)
	if r.Status != "fail" {
		t.Fatalf("unknown verifier must fail, got %q", r.Status)
	}
}
