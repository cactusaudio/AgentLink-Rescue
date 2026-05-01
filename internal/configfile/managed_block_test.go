package configfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagedBlockAppendIdempotentAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(path, []byte("export FOO=bar\n"), 0644); err != nil {
		t.Fatal(err)
	}
	first := AppendManagedBlockIfMissing(path, "TEST", "export PATH=\"$HOME/bin:$PATH\"")
	if !first.Changed {
		t.Fatalf("first append did not change: %+v", first)
	}
	second := AppendManagedBlockIfMissing(path, "TEST", "export PATH=\"$HOME/bin:$PATH\"")
	if second.Changed {
		t.Fatalf("second append changed: %+v", second)
	}
	if got := CountManagedBlocks(path, "TEST"); got != 1 {
		t.Fatalf("block count=%d", got)
	}
	removed := RemoveManagedBlock(path, "TEST")
	if !removed.Changed {
		t.Fatalf("remove did not change: %+v", removed)
	}
	if HasManagedBlock(path, "TEST") {
		t.Fatal("block still present")
	}
}

func TestDuplicateManagedBlockDetectedByCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zshrc")
	text := StartMarker("DUP") + "\nA\n" + EndMarker("DUP") + "\n" + StartMarker("DUP") + "\nB\n" + EndMarker("DUP") + "\n"
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	if got := CountManagedBlocks(path, "DUP"); got != 2 {
		t.Fatalf("block count=%d", got)
	}
}
