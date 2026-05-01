package facts

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type Facts struct {
	SchemaVersion  int                       `json:"schemaVersion"`
	ToolVersion    string                    `json:"toolVersion"`
	CreatedAt      string                    `json:"createdAt"`
	OS             string                    `json:"os"`
	Arch           string                    `json:"arch"`
	Shell          string                    `json:"shell"`
	Path           string                    `json:"path"`
	RealUser       string                    `json:"realUser"`
	Home           string                    `json:"home"`
	ProxyEnv       map[string]string         `json:"proxyEnv"`
	Commands       map[string]ToolFact       `json:"commands"`
	ConfigPaths    map[string]PathFact       `json:"configPaths"`
	APIKeys        map[string]KeyFact        `json:"apiKeys"`
	Ports          map[string]PortFact       `json:"ports"`
	Network        diagnose.DiagnosticReport `json:"network"`
	Warnings       []string                  `json:"warnings"`
	LikelyFailures []string                  `json:"likelyFailureClasses"`
}

type ToolFact struct {
	Name    string `json:"name"`
	Exists  bool   `json:"exists"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

type PathFact struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type KeyFact struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
}

type PortFact struct {
	Port      string `json:"port"`
	Listening bool   `json:"listening"`
	Error     string `json:"error,omitempty"`
}

func Collect(ctx context.Context, runner command.Runner, home string, includeNetwork bool) Facts {
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	userInfo := system.RealConsoleUser(ctx, runner)
	if userInfo.Home != "" {
		home = userInfo.Home
	}
	f := Facts{
		SchemaVersion:  1,
		ToolVersion:    system.Version,
		CreatedAt:      time.Now().Format(time.RFC3339),
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		Shell:          os.Getenv("SHELL"),
		Path:           os.Getenv("PATH"),
		RealUser:       userInfo.Name,
		Home:           home,
		ProxyEnv:       collectProxyEnv(),
		Commands:       map[string]ToolFact{},
		ConfigPaths:    map[string]PathFact{},
		APIKeys:        map[string]KeyFact{},
		Ports:          map[string]PortFact{},
		Warnings:       []string{},
		LikelyFailures: []string{},
	}
	for _, name := range []string{"codex", "claude", "opencode", "git", "npm", "node", "brew", "curl", "jq", "tmux", "ssh"} {
		f.Commands[name] = commandFact(name)
	}
	f.ConfigPaths["codex"] = pathFact(filepath.Join(home, ".codex", "config.toml"))
	f.ConfigPaths["claude"] = pathFact(filepath.Join(home, ".claude"))
	f.ConfigPaths["opencode"] = pathFact(filepath.Join(home, ".config", "opencode"))
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "OPENROUTER_API_KEY"} {
		val, ok := os.LookupEnv(key)
		k := KeyFact{Name: key, Present: ok && val != ""}
		if k.Present {
			k.Value = "REDACTED"
		}
		f.APIKeys[key] = k
	}
	for _, port := range []string{"7890", "7897", "1080", "8080"} {
		f.Ports[port] = portFact(port)
	}
	if includeNetwork {
		engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: findRulesDir()})
		report := engine.Run(ctx)
		classify.Apply(&report)
		f.Network = report
		f.LikelyFailures = append(f.LikelyFailures, report.Classifications...)
	}
	if len(f.ProxyEnv) > 0 {
		f.LikelyFailures = appendIfMissing(f.LikelyFailures, classify.UserProxyDirty)
	}
	return f
}

func collectProxyEnv() map[string]string {
	out := map[string]string{}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"} {
		if val := os.Getenv(key); val != "" {
			out[key] = safety.RedactProxyValue(val)
		}
	}
	return out
}

func commandFact(name string) ToolFact {
	path, ok := findOnPath(name)
	return ToolFact{Name: name, Exists: ok, Path: path}
}

func findOnPath(name string) (string, bool) {
	if strings.Contains(name, "/") {
		if system.CommandExists(name) {
			return name, true
		}
		return "", false
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, name)
		if system.CommandExists(path) {
			return path, true
		}
	}
	for _, dir := range []string{"/usr/bin", "/bin", "/usr/sbin", "/sbin", "/opt/homebrew/bin", "/usr/local/bin"} {
		path := filepath.Join(dir, name)
		if system.CommandExists(path) {
			return path, true
		}
	}
	return "", false
}

func pathFact(path string) PathFact {
	return PathFact{Path: path, Exists: system.Exists(path)}
}

func portFact(port string) PortFact {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 250*time.Millisecond)
	if err != nil {
		return PortFact{Port: port, Listening: false, Error: safety.RedactSensitive(err.Error())}
	}
	_ = conn.Close()
	return PortFact{Port: port, Listening: true}
}

func appendIfMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func findRulesDir() string {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "rules"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "rules"), filepath.Join(dir, "..", "rules"), filepath.Join(dir, "..", "..", "rules"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}
