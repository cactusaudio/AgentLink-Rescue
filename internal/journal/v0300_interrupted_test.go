package journal

import (
	"testing"
)

// V0300 W4: an interrupted (non-terminal, no-mutation) transaction must
// be recoverable to a consistent terminal state, and recovery must be
// idempotent (re-running Recover does not change anything).
func TestV0300InterruptedJournalRecoveryIsIdempotent(t *testing.T) {
	home := t.TempDir()
	m := Manager{Home: home}

	tx, err := m.Start(StartOptions{Target: "network", Recipe: "proxy-clean-stale-env"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	// simulate interruption: advance to a non-terminal state, no mutation log.
	if err := m.UpdateState(tx, StateCheckpointed); err != nil {
		t.Fatalf("update state: %v", err)
	}

	inc, err := m.Incomplete()
	if err != nil {
		t.Fatalf("incomplete: %v", err)
	}
	if len(inc) != 1 {
		t.Fatalf("expected 1 incomplete tx, got %d", len(inc))
	}

	// dry recovery must NOT change state, must report it as recoverable.
	dry := m.Recover(true)
	if dry.Status != "incomplete_transactions_found" {
		t.Fatalf("dry recover status = %q, want incomplete_transactions_found", dry.Status)
	}
	if inc2, _ := m.Incomplete(); len(inc2) != 1 {
		t.Fatalf("dry recover mutated state: incomplete now %d", len(inc2))
	}

	// real recovery must resolve the interrupted tx to terminal.
	got := m.Recover(false)
	if got.Status == "failed" {
		t.Fatalf("real recover failed: %+v", got.Warnings)
	}
	if inc3, _ := m.Incomplete(); len(inc3) != 0 {
		t.Fatalf("after real recover, still %d incomplete (not resolved)", len(inc3))
	}

	// idempotent: a second real recovery is a no-op ("none").
	again := m.Recover(false)
	if again.Status != "none" {
		t.Fatalf("second recover not idempotent: status=%q (want none)", again.Status)
	}

	// the recovered transaction must be in a terminal state on disk.
	rtx, err := m.Load(tx.TransactionID)
	if err != nil {
		t.Fatalf("load recovered tx: %v", err)
	}
	if !isTerminal(rtx.State) {
		t.Fatalf("recovered tx state %q is not terminal", rtx.State)
	}
}
