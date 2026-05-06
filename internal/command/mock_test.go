package command

import (
	"context"
	"strings"
	"testing"
	"time"
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

func TestExecRunnerCapsAndRedactsOutput(t *testing.T) {
	r := ExecRunner{Timeout: 5 * time.Second, OutputLimitBytes: 1024}
	res := r.Run(context.Background(), "/bin/sh", "-c", "yes A | head -c 4096; echo sk-test-SECRET12345 >&2")
	if res.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if !res.StdoutTruncated {
		t.Fatalf("stdout was not marked truncated")
	}
	if len(res.Stdout) > 1200 {
		t.Fatalf("stdout cap not applied: %d bytes", len(res.Stdout))
	}
	if !strings.Contains(res.Stderr, "SECRET12345") {
		t.Fatalf("raw stderr should remain available for internal rollback truth: %q", res.Stderr)
	}
	if strings.Contains(res.RedactedStderr, "SECRET12345") {
		t.Fatalf("redacted stderr leaked secret: %q", res.RedactedStderr)
	}
}

func TestExecRunnerTimeoutTerminatesCommand(t *testing.T) {
	r := ExecRunner{Timeout: 50 * time.Millisecond, OutputLimitBytes: 1024}
	res := r.Run(context.Background(), "/bin/sh", "-c", "sleep 2")
	if !res.TimedOut {
		t.Fatalf("expected timeout: %+v", res)
	}
	if res.ExitCode != -1 {
		t.Fatalf("expected timeout exit code -1, got %d", res.ExitCode)
	}
}
