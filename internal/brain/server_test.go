package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerStatusRemovesStalePID(t *testing.T) {
	home := t.TempDir()
	pidFile := serverPIDFile(home)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte("999999"), 0644); err != nil {
		t.Fatal(err)
	}
	rep := ServerStatus(home, 65534)
	if rep.PID != 0 {
		t.Fatalf("stale pid reported as active: %+v", rep)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("stale pid file was not removed: %v", err)
	}
}

func TestServerLogDirUnderApplicationSupport(t *testing.T) {
	home := t.TempDir()
	got := serverLogDir(home)
	if !filepath.IsAbs(got) || !strings.HasPrefix(got, filepath.Join(home, "Library", "Application Support")) {
		t.Fatalf("unexpected server log dir: %s", got)
	}
}
