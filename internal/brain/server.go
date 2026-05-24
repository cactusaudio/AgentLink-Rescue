package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

type ServerReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	ToolVersion   string `json:"toolVersion"`
	Status        string `json:"status"`
	Port          int    `json:"port"`
	PID           int    `json:"pid,omitempty"`
	PIDFile       string `json:"pidFile,omitempty"`
	URL           string `json:"url,omitempty"`
	ServerPath    string `json:"serverPath,omitempty"`
	ModelPath     string `json:"modelPath,omitempty"`
	StdoutLog     string `json:"stdoutLog,omitempty"`
	StderrLog     string `json:"stderrLog,omitempty"`
	Error         string `json:"error,omitempty"`
}

func ServerStatus(home string, port int) ServerReport {
	if port <= 0 {
		port = 8080
	}
	rep := ServerReport{SchemaVersion: 1, ToolVersion: system.Version, Port: port, URL: fmt.Sprintf("http://127.0.0.1:%d/v1", port), PIDFile: serverPIDFile(home)}
	if pid := readPID(rep.PIDFile); pid > 0 {
		if processAlive(pid) {
			rep.PID = pid
		} else {
			_ = os.Remove(rep.PIDFile)
		}
	}
	if listening(port) {
		rep.Status = "running"
	} else {
		rep.Status = "stopped"
	}
	return rep
}

func StartServer(ctx context.Context, runner command.Runner, home string, port int) ServerReport {
	if port <= 0 {
		port = 8080
	}
	backend := NewLlamaCLIBackend(runner, home)
	avail := backend.Available(ctx)
	rep := ServerStatus(home, port)
	rep.ServerPath = avail.ServerPath
	rep.ModelPath = avail.ModelPath
	if rep.Status == "running" {
		return rep
	}
	if !avail.ModelExists || !avail.ModelSHA256OK {
		rep.Status = "failed"
		rep.Error = "Gemma model missing or checksum mismatch"
		return rep
	}
	if !avail.ServerExecutable {
		rep.Status = "failed"
		rep.Error = "llama-server missing or not executable"
		return rep
	}
	if err := os.MkdirAll(filepath.Dir(rep.PIDFile), 0700); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	logDir := serverLogDir(home)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	stdoutPath := filepath.Join(logDir, "llama-server.stdout.log")
	stderrPath := filepath.Join(logDir, "llama-server.stderr.log")
	stdoutLog, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	defer stdoutLog.Close()
	stderrLog, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	defer stderrLog.Close()
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	defer stdin.Close()
	cmd := exec.CommandContext(ctx, avail.ServerPath, "-m", avail.ModelPath, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "-c", "8192")
	cmd.Stdin = stdin
	cmd.Stdout = stdoutLog
	cmd.Stderr = stderrLog
	detachServer(cmd)
	if err := cmd.Start(); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	rep.PID = cmd.Process.Pid
	rep.StdoutLog = stdoutPath
	rep.StderrLog = stderrPath
	_ = os.WriteFile(rep.PIDFile, []byte(strconv.Itoa(rep.PID)), 0600)
	time.Sleep(500 * time.Millisecond)
	if listening(port) {
		rep.Status = "running"
	} else {
		rep.Status = "starting"
	}
	return rep
}

func StopServer(home string, port int) ServerReport {
	rep := ServerStatus(home, port)
	if rep.PID > 0 {
		if p, err := os.FindProcess(rep.PID); err == nil && processAlive(rep.PID) {
			_ = p.Signal(os.Interrupt)
			time.Sleep(300 * time.Millisecond)
			_ = p.Kill()
		}
		_ = os.Remove(rep.PIDFile)
	}
	rep.Status = "stopped"
	return rep
}

func MarshalServer(rep ServerReport) string {
	data, _ := json.MarshalIndent(rep, "", "  ")
	return string(data)
}

func serverPIDFile(home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, "Library", "Application Support", system.AppName, "brain-server", "llama-server.pid")
}

func serverLogDir(home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, "Library", "Application Support", system.AppName, "brain-server", "logs")
}

func readPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return pid
}

func listening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}
