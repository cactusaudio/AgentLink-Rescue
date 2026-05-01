package verifier

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/configfile"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type Result struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Evidence   string `json:"evidence,omitempty"`
	DurationMS int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

type Context struct {
	Context context.Context
	Runner  command.Runner
	Home    string
	Params  map[string]string
}

type Definition struct {
	ID   string
	Type string
	Run  func(Context, map[string]string) Result
}

type Registry struct {
	defs map[string]Definition
}

func NewRegistry() Registry {
	r := Registry{defs: map[string]Definition{}}
	for _, def := range builtins() {
		r.defs[def.ID] = def
	}
	return r
}

func (r Registry) Exists(id string) bool {
	_, ok := r.defs[id]
	return ok
}

func (r Registry) IDs() []string {
	out := make([]string, 0, len(r.defs))
	for id := range r.defs {
		out = append(out, id)
	}
	return out
}

func (r Registry) Run(ctx Context, id string, args map[string]string) Result {
	def, ok := r.defs[id]
	if !ok {
		return Result{ID: id, Type: "unknown", Status: "fail", Error: "unknown verifier"}
	}
	start := time.Now()
	res := def.Run(ctx, args)
	res.ID = id
	res.Type = def.Type
	res.DurationMS = time.Since(start).Milliseconds()
	res.Evidence = safety.RedactSensitive(res.Evidence)
	res.Error = safety.RedactSensitive(res.Error)
	return res
}

func builtins() []Definition {
	return []Definition{
		{ID: "shell_login_path_contains", Type: "shell", Run: shellLoginPathContains},
		{ID: "command_exists_in_login_shell", Type: "shell", Run: commandExistsInLoginShell},
		{ID: "shell_config_no_parse_error", Type: "shell", Run: shellConfigNoParseError},
		{ID: "managed_block_count", Type: "shell", Run: managedBlockCount},
		{ID: "env_proxy_absent", Type: "proxy", Run: envProxyAbsent},
		{ID: "env_proxy_matches", Type: "proxy", Run: envProxyMatches},
		{ID: "local_port_listening", Type: "proxy", Run: localPortListening},
		{ID: "git_proxy_absent", Type: "proxy", Run: gitProxyAbsent},
		{ID: "git_proxy_matches", Type: "proxy", Run: gitProxyMatches},
		{ID: "npm_proxy_absent", Type: "proxy", Run: npmProxyAbsent},
		{ID: "npm_proxy_matches", Type: "proxy", Run: npmProxyMatches},
		{ID: "curl_https_optional", Type: "proxy", Run: curlHTTPSOptional},
		{ID: "codex_command_exists", Type: "codex", Run: commandExists},
		{ID: "codex_version_runs", Type: "codex", Run: codexVersionRuns},
		{ID: "codex_config_exists", Type: "codex", Run: fileExists},
		{ID: "codex_config_basic_syntax_ok", Type: "codex", Run: tomlOK},
		{ID: "codex_provider_section_present", Type: "codex", Run: fileContains},
		{ID: "codex_provider_name_present", Type: "codex", Run: fileContains},
		{ID: "codex_provider_base_url_present", Type: "codex", Run: fileContains},
		{ID: "codex_model_present", Type: "codex", Run: fileContains},
		{ID: "codex_provider_env_key_present", Type: "codex", Run: fileContains},
		{ID: "codex_provider_key_env_present_redacted", Type: "codex", Run: envKeyPresent},
		{ID: "codex_provider_smoke_test_required", Type: "codex", Run: providerSmokeTestRequired},
		{ID: "env_key_present_redacted", Type: "keys", Run: envKeyPresent},
		{ID: "no_secret_leak_in_report", Type: "keys", Run: noSecretLeak},
		{ID: "key_not_written_to_config", Type: "keys", Run: fileNotContainsEnvValue},
		{ID: "json_basic_parse_ok", Type: "config", Run: jsonOK},
		{ID: "toml_basic_syntax_ok", Type: "config", Run: tomlOK},
		{ID: "file_contains", Type: "config", Run: fileContains},
		{ID: "file_not_contains", Type: "config", Run: fileNotContains},
		{ID: "managed_section_present", Type: "config", Run: managedSectionPresent},
	}
}

func pass(e string) Result    { return Result{Status: "pass", Evidence: e} }
func fail(e string) Result    { return Result{Status: "fail", Error: e} }
func warn(e string) Result    { return Result{Status: "warn", Evidence: e} }
func skipped(e string) Result { return Result{Status: "skipped", Evidence: e} }

func expand(ctx Context, path string) string {
	if path == "" {
		return ""
	}
	if path == "~" {
		return ctx.Home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(ctx.Home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func commandExists(ctx Context, args map[string]string) Result {
	name := args["command"]
	if name == "" {
		name = "codex"
	}
	if _, ok := findOnPath(name); ok {
		return pass(name + " exists")
	}
	return fail(name + " missing")
}

func findOnPath(name string) (string, bool) {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
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

func fileExists(ctx Context, args map[string]string) Result {
	path := expand(ctx, args["path"])
	if system.Exists(path) {
		return pass(path + " exists")
	}
	return fail(path + " missing")
}

func readFile(ctx Context, args map[string]string) (string, string, error) {
	path := expand(ctx, args["path"])
	data, err := os.ReadFile(path)
	return path, string(data), err
}

func fileContains(ctx Context, args map[string]string) Result {
	path, text, err := readFile(ctx, args)
	if err != nil {
		return fail(err.Error())
	}
	needle := args["contains"]
	if strings.Contains(text, needle) {
		return pass(path + " contains expected text")
	}
	return fail(path + " does not contain expected text")
}

func fileNotContains(ctx Context, args map[string]string) Result {
	path, text, err := readFile(ctx, args)
	if err != nil {
		return fail(err.Error())
	}
	needle := args["contains"]
	if !strings.Contains(text, needle) {
		return pass(path + " does not contain forbidden text")
	}
	return fail(path + " contains forbidden text")
}

func jsonOK(ctx Context, args map[string]string) Result {
	if err := configfile.JSONBasicParseOK(expand(ctx, args["path"])); err != nil {
		return fail(err.Error())
	}
	return pass("JSON parse OK")
}

func tomlOK(ctx Context, args map[string]string) Result {
	if err := configfile.TOMLBasicSyntaxOK(expand(ctx, args["path"])); err != nil {
		return fail(err.Error())
	}
	return pass("TOML syntax OK")
}

func managedSectionPresent(ctx Context, args map[string]string) Result {
	path := expand(ctx, args["path"])
	marker := args["marker"]
	if configfile.HasManagedBlock(path, marker) {
		return pass("managed section present")
	}
	return fail("managed section absent")
}

func resultFromBool(ok bool, evidence string) Result {
	if ok {
		return pass(evidence)
	}
	return fail(evidence)
}

func quoteShell(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func commandString(res command.Result) string {
	return fmt.Sprintf("%s exit=%d", res.Command, res.ExitCode)
}
