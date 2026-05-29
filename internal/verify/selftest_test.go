package verify

import "testing"

// TestSelftestPasses runs the built-in self-test that validates the diagnostic
// parsers, redaction, and rule matching against frozen samples. It must report
// zero errors — a failure here means a core parser regressed.
func TestSelftestPasses(t *testing.T) {
	if errs := Selftest(); len(errs) != 0 {
		t.Fatalf("selftest reported %d error(s): %v", len(errs), errs)
	}
}
