package command

import (
	"context"
	"testing"
)

func TestMockRunner(t *testing.T) {
	m := &MockRunner{Results: map[string]Result{
		Render("/bin/echo", "ok"): {ExitCode: 0, Stdout: "ok\n"},
	}}
	res := m.Run(context.Background(), "/bin/echo", "ok")
	if res.ExitCode != 0 || res.Stdout != "ok\n" {
		t.Fatalf("bad result: %+v", res)
	}
	if len(m.Calls) != 1 {
		t.Fatalf("calls=%d", len(m.Calls))
	}
}
