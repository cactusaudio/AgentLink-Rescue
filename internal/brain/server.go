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
	Error         string `json:"error,omitempty"`
}

func ServerStatus(home string, port int) ServerReport {
	if port <= 0 {
		port = 8080
	}
	rep := ServerReport{SchemaVersion: 1, ToolVersion: system.Version, Port: port, URL: fmt.Sprintf("http://127.0.0.1:%d/v1", port), PIDFile: serverPIDFile(home)}
	if pid := readPID(rep.PIDFile); pid > 0 {
		rep.PID = pid
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
	if err := os.MkdirAll(filepath.Dir(rep.PIDFile), 0755); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	cmd := exec.CommandContext(ctx, avail.ServerPath, "-m", avail.ModelPath, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "-c", "8192")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		rep.Status = "failed"
		rep.Error = err.Error()
		return rep
	}
	rep.PID = cmd.Process.Pid
	_ = os.WriteFile(rep.PIDFile, []byte(strconv.Itoa(rep.PID)), 0644)
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
		if p, err := os.FindProcess(rep.PID); err == nil {
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
