package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 10 * time.Second

type Result struct {
	Path           string   `json:"path"`
	Args           []string `json:"args"`
	Command        string   `json:"command"`
	ExitCode       int      `json:"exitCode"`
	Stdout         string   `json:"stdout,omitempty"`
	Stderr         string   `json:"stderr,omitempty"`
	Error          string   `json:"error,omitempty"`
	Missing        bool     `json:"missing,omitempty"`
	TimedOut       bool     `json:"timedOut,omitempty"`
	DurationMillis int64    `json:"durationMillis"`
}

type Runner interface {
	Run(ctx context.Context, path string, args ...string) Result
}

type ExecRunner struct {
	Timeout time.Duration
}

func NewExecRunner() ExecRunner {
	return ExecRunner{Timeout: DefaultTimeout}
}

func (r ExecRunner) Run(ctx context.Context, path string, args ...string) Result {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	start := time.Now()
	res := Result{
		Path:    path,
		Args:    append([]string(nil), args...),
		Command: Render(path, args...),
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res.DurationMillis = time.Since(start).Milliseconds()
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()
	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		res.Error = "command timed out"
		return res
	}
	if err == nil {
		res.ExitCode = 0
		return res
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
		res.Error = err.Error()
		return res
	}
	res.ExitCode = 127
	res.Error = err.Error()
	if errors.Is(err, exec.ErrNotFound) || strings.Contains(err.Error(), "no such file") {
		res.Missing = true
	}
	return res
}

func Render(path string, args ...string) string {
	parts := append([]string{path}, args...)
	for i, p := range parts {
		if p == "" || strings.ContainsAny(p, " \t\n'\"\\$") {
			parts[i] = fmt.Sprintf("%q", p)
		}
	}
	return strings.Join(parts, " ")
}

func Success(r Result) bool {
	return r.ExitCode == 0 && !r.TimedOut && !r.Missing
}
