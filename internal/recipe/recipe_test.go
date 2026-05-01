package recipe

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

func testRegistry(t *testing.T) Registry {
	t.Helper()
	reg, err := LoadRegistry(filepath.Join("..", "..", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestRecipeLoadValidateAndUnknown(t *testing.T) {
	reg := testRegistry(t)
	if len(reg.List()) != 11 {
		t.Fatalf("recipe count=%d", len(reg.List()))
	}
	if _, ok := reg.Get("missing"); ok {
		t.Fatal("unexpected missing recipe")
	}
	if err := ValidateRecipe(Recipe{SchemaVersion: 1, ID: "bad", Title: "bad", Risk: "not_real"}); err == nil {
		t.Fatal("invalid risk accepted")
	}
	if err := ValidateRecipe(Recipe{SchemaVersion: 1, ID: "bad", Title: "bad", Risk: RiskReadOnly, Patches: []Patch{{Type: "raw_shell"}}}); err == nil {
		t.Fatal("raw_shell accepted")
	}
	if SupportsOS(Recipe{SupportedOS: []string{"linux"}}, "darwin") {
		t.Fatal("unsupported OS recipe was supported")
	}
}

func TestMacOSZshPathRepairDryRunAndRollback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("# original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	reg := testRegistry(t)
	dry := Run(context.Background(), &command.MockRunner{}, reg, "macos-zsh-path-repair", RunOptions{Home: home, DryRun: true})
	if dry.Status != StatusDryRun {
		t.Fatalf("unexpected dry status: %+v", dry)
	}
	data, _ := os.ReadFile(zshrc)
	if string(data) != "# original\n" {
		t.Fatal("dry-run changed file")
	}
	run := Run(context.Background(), nil, reg, "macos-zsh-path-repair", RunOptions{Home: home, Yes: true})
	if run.Status != StatusSuccess {
		t.Fatalf("run failed: %+v", run)
	}
	after, _ := os.ReadFile(zshrc)
	if strings.Count(string(after), "AGENTLINK_PATH_BLOCK") != 2 {
		t.Fatalf("managed block not added once:\n%s", after)
	}
	again := Run(context.Background(), nil, reg, "macos-zsh-path-repair", RunOptions{Home: home, Yes: true})
	if again.Status != StatusSuccess {
		t.Fatalf("second run failed: %+v", again)
	}
	afterAgain, _ := os.ReadFile(zshrc)
	if strings.Count(string(afterAgain), "AGENTLINK_PATH_BLOCK") != 2 {
		t.Fatalf("duplicate block added:\n%s", afterAgain)
	}
	rp, err := snapshot.Latest(system.UserRestorePointsDir(home))
	if err != nil {
		t.Fatal(err)
	}
	rp.SetMutationPolicy(system.MutationOptions{RealUserHome: home, ExtraAllowedPaths: []string{zshrc}})
	if err := rp.RestoreAll(); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(zshrc)
	if string(restored) != "# original\n" {
		t.Fatalf("rollback did not restore original: %q", restored)
	}
}

func TestCodexConfigParseRepairDoesNotWriteRealKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-secret-value")
	path := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[broken\n"), 0644); err != nil {
		t.Fatal(err)
	}
	reg := testRegistry(t)
	dry := Run(context.Background(), nil, reg, "codex-config-parse-repair", RunOptions{Home: home, DryRun: true})
	if dry.Status != StatusDryRun {
		t.Fatalf("unexpected dry result: %+v", dry)
	}
	unchanged, _ := os.ReadFile(path)
	if string(unchanged) != "[broken\n" {
		t.Fatal("dry-run changed malformed config")
	}
	run := Run(context.Background(), nil, reg, "codex-config-parse-repair", RunOptions{Home: home, Yes: true})
	if run.Status != StatusConfigTemplateGenerated {
		t.Fatalf("repair failed: %+v", run)
	}
	repaired, _ := os.ReadFile(path)
	if strings.Contains(string(repaired), "sk-test-secret-value") {
		t.Fatal("real key written to config")
	}
	if !strings.Contains(string(repaired), "model") {
		t.Fatalf("minimal template not written: %s", repaired)
	}
}

func TestVerifierFailureCausesRecipeFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	reg := Registry{recipes: map[string]Recipe{
		"fail-recipe": {
			SchemaVersion: 1,
			ID:            "fail-recipe",
			Title:         "Fail",
			SupportedOS:   []string{"darwin"},
			Risk:          RiskReadOnly,
			Verify:        []VerifierRef{{ID: "file_contains", Args: map[string]string{"path": "~/missing", "contains": "x"}}},
		},
	}}
	res := Run(context.Background(), nil, reg, "fail-recipe", RunOptions{Home: home})
	if res.Status != StatusVerifierFailed {
		t.Fatalf("expected verifier failure, got %+v", res)
	}
}

func TestGitNpmOriginalValuesRecorded(t *testing.T) {
	runner := &command.MockRunner{Results: map[string]command.Result{
		command.Render("/usr/bin/git", "config", "--global", "--get", "http.proxy"):   {ExitCode: 0, Stdout: "http://user:pass@127.0.0.1:7890\n"},
		command.Render("/usr/bin/git", "config", "--global", "--unset", "http.proxy"): {ExitCode: 0},
	}}
	reg := Registry{recipes: map[string]Recipe{
		"git-clean": {
			SchemaVersion: 1,
			ID:            "git-clean",
			Title:         "Git clean",
			SupportedOS:   []string{"darwin"},
			Risk:          RiskReversiblePatch,
			Patches:       []Patch{{ID: "unset", Type: "unset_git_config_key", Key: "http.proxy"}},
			Rollback:      []RollbackSpec{{Type: "restore_recorded_config_values", Key: "http.proxy"}},
		},
	}}
	home := t.TempDir()
	res := Run(context.Background(), runner, reg, "git-clean", RunOptions{Home: home, Yes: true})
	if res.Status != StatusSuccess {
		t.Fatalf("run failed: %+v", res)
	}
	if len(runner.Calls) < 2 {
		t.Fatalf("expected get and unset calls, got %v", runner.Calls)
	}
	rp, err := snapshot.Latest(system.UserRestorePointsDir(home))
	if err != nil {
		t.Fatal(err)
	}
	if len(rp.Manifest.ConfigEntries) != 1 {
		t.Fatalf("expected one config entry, got %+v", rp.Manifest.ConfigEntries)
	}
	entry := rp.Manifest.ConfigEntries[0]
	if !entry.Existed || entry.OldValue != "http://user:pass@127.0.0.1:7890" {
		t.Fatalf("bad recorded config entry: %+v", entry)
	}
}

func TestCodexDeepSeekRecipeUsesEnvKeyAndDefaults(t *testing.T) {
	reg := testRegistry(t)
	r, ok := reg.Get("codex-deepseek-provider-config")
	if !ok {
		t.Fatal("missing codex-deepseek-provider-config")
	}
	params, err := ResolveParams(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if params["baseURL"] != "https://api.deepseek.com" {
		t.Fatalf("baseURL=%s", params["baseURL"])
	}
	if params["model"] != "deepseek-v4-flash" {
		t.Fatalf("model=%s", params["model"])
	}
	content := Interpolate(r.Patches[0].Content, t.TempDir(), params)
	if !strings.Contains(content, `env_key = "DEEPSEEK_API_KEY"`) {
		t.Fatalf("env_key missing:\n%s", content)
	}
	if strings.Contains(content, "api_"+"key_env") {
		t.Fatalf("stale Codex API key env field found:\n%s", content)
	}
}

func TestCodexDeepSeekLegacyModelWarningAndSecretStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DEEPSEEK_API_KEY", "")
	reg := testRegistry(t)
	res := Run(context.Background(), nil, reg, "codex-deepseek-provider-config", RunOptions{
		Home: home,
		Yes:  true,
		Params: map[string]string{
			"model": "deepseek-chat",
		},
	})
	if res.Status != StatusNeedsUserSecret {
		t.Fatalf("status=%s result=%+v", res.Status, res)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "legacy DeepSeek compatibility model") {
		t.Fatalf("legacy warning missing: %+v", res.Warnings)
	}
	if !strings.Contains(joined, "provider_smoke_test_required") {
		t.Fatalf("smoke warning missing: %+v", res.Warnings)
	}
	path := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `env_key = "DEEPSEEK_API_KEY"`) || strings.Contains(text, "api_"+"key_env") {
		t.Fatalf("bad DeepSeek config:\n%s", text)
	}
}

func TestAPIKeyDetectionOptionalMissingDoesNotFail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "OPENROUTER_API_KEY"} {
		t.Setenv(key, "")
	}
	reg := testRegistry(t)
	res := Run(context.Background(), nil, reg, "api-key-detection-redaction", RunOptions{Home: home})
	if res.Status != StatusSuccessWithWarnings {
		t.Fatalf("status=%s result=%+v", res.Status, res)
	}
	for _, vr := range res.VerifierResults {
		if vr.ID == "env_key_present_redacted" && vr.Status != "warn" {
			t.Fatalf("expected optional key warning, got %+v", vr)
		}
	}
}

func TestProxyClashSetGitNpmRecordsAndRestoresValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("# original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &command.MockRunner{Results: map[string]command.Result{
		command.Render("/usr/bin/git", "config", "--global", "--get", "http.proxy"):                  {ExitCode: 0, Stdout: "http://old.example:7890\n"},
		command.Render("/usr/bin/git", "config", "--global", "--get", "https.proxy"):                 {ExitCode: 1},
		command.Render("/usr/bin/npm", "config", "get", "proxy"):                                     {ExitCode: 0, Stdout: "null\n"},
		command.Render("/usr/bin/npm", "config", "get", "https-proxy"):                               {ExitCode: 0, Stdout: "https://old.example:7890\n"},
		command.Render("/usr/bin/git", "config", "--global", "http.proxy", "http://127.0.0.1:7897"):  {ExitCode: 0},
		command.Render("/usr/bin/git", "config", "--global", "https.proxy", "http://127.0.0.1:7897"): {ExitCode: 0},
		command.Render("/usr/bin/npm", "config", "set", "proxy", "http://127.0.0.1:7897"):            {ExitCode: 0},
		command.Render("/usr/bin/npm", "config", "set", "https-proxy", "http://127.0.0.1:7897"):      {ExitCode: 0},
	}}
	reg := testRegistry(t)
	r, ok := reg.Get("proxy-clash-7897-apply")
	if !ok {
		t.Fatal("missing proxy-clash recipe")
	}
	r.Preconditions = []Precondition{{Type: "os_is", Value: "darwin"}, {Type: "file_exists_or_creatable", Path: "~/.zshrc"}}
	r.Verify = nil
	reg = RegistryForTest(map[string]Recipe{r.ID: r})
	res := Run(context.Background(), runner, reg, r.ID, RunOptions{Home: home, Yes: true, Params: map[string]string{"setGitNpm": "true"}})
	if res.Status != StatusSuccess {
		t.Fatalf("run failed: %+v", res)
	}
	rp, err := snapshot.Latest(system.UserRestorePointsDir(home))
	if err != nil {
		t.Fatal(err)
	}
	if len(rp.Manifest.ConfigEntries) != 4 {
		t.Fatalf("config entries=%d %+v", len(rp.Manifest.ConfigEntries), rp.Manifest.ConfigEntries)
	}
	restoreRunner := &command.MockRunner{Results: map[string]command.Result{
		command.Render("/usr/bin/npm", "config", "set", "https-proxy", "https://old.example:7890"):    {ExitCode: 0},
		command.Render("/usr/bin/npm", "config", "delete", "proxy"):                                   {ExitCode: 0},
		command.Render("/usr/bin/git", "config", "--global", "--unset", "https.proxy"):                {ExitCode: 0},
		command.Render("/usr/bin/git", "config", "--global", "http.proxy", "http://old.example:7890"): {ExitCode: 0},
	}}
	if err := rp.RestoreConfigEntries(context.Background(), restoreRunner); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(restoreRunner.Calls, "\n")
	for _, want := range []string{
		"/usr/bin/git config --global http.proxy http://old.example:7890",
		"/usr/bin/git config --global --unset https.proxy",
		"/usr/bin/npm config delete proxy",
		"/usr/bin/npm config set https-proxy https://old.example:7890",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing restore command %q in\n%s", want, joined)
		}
	}
}
