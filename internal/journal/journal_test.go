package journal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverMarksNoMutationTransactionAbandoned(t *testing.T) {
	m := New(t.TempDir())
	tx, err := m.Start(StartOptions{Target: "network", Recipe: "rescue:safe"})
	if err != nil {
		t.Fatal(err)
	}
	rep := m.Recover(false)
	if rep.Status != "abandoned" {
		t.Fatalf("expected abandoned, got %+v", rep)
	}
	loaded, err := m.Load(tx.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateAbandoned {
		t.Fatalf("state=%s", loaded.State)
	}
}

func TestRecoverRecommendsRollbackAfterMutation(t *testing.T) {
	m := New(t.TempDir())
	tx, err := m.Start(StartOptions{Target: "network", Recipe: "rescue:tun"})
	if err != nil {
		t.Fatal(err)
	}
	tx.RestorePoint = "restore-123"
	tx.RollbackAvailable = true
	if err := m.Save(tx); err != nil {
		t.Fatal(err)
	}
	if err := m.AppendMutation(tx, MutationEntry{ActionID: "test.mutate", State: StateMutating}); err != nil {
		t.Fatal(err)
	}
	rep := m.Recover(true)
	if rep.Status != "incomplete_transactions_found" {
		t.Fatalf("status=%s", rep.Status)
	}
	if rep.NextAction != "agentlink rollback --id restore-123" {
		t.Fatalf("next action=%q", rep.NextAction)
	}
	loaded, err := m.Load(tx.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateStarted {
		t.Fatalf("dry recover mutated transaction state: %s", loaded.State)
	}
}

func TestMalformedJournalReportsError(t *testing.T) {
	m := New(t.TempDir())
	tx, err := m.Start(StartOptions{Target: "network", Recipe: "rescue:safe"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.txDir(tx.TransactionID), "transaction.json"), []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Load(tx.TransactionID); err == nil {
		t.Fatalf("expected malformed journal load error")
	}
}
