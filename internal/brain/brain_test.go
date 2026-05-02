package brain

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

type fakeBackend struct {
	texts []string
	calls int
}

func (f *fakeBackend) Name() string { return "fake" }
func (f *fakeBackend) Available(context.Context) BrainAvailability {
	return BrainAvailability{Backend: "fake", BrainPackAvailable: true, ModelExists: true, ModelSHA256OK: true, RuntimeExecutable: true}
}
func (f *fakeBackend) Generate(_ context.Context, _ BrainRequest) (BrainResponse, error) {
	text := f.texts[f.calls]
	f.calls++
	jsonText, _ := ExtractJSONObject(text)
	return BrainResponse{RawText: text, ExtractedJSON: jsonText, Backend: "fake", DurationMs: 1}, nil
}

func testRecipeRegistry() recipe.Registry {
	return recipe.RegistryForTest(map[string]recipe.Recipe{
		"macos-zsh-path-repair": {
			SchemaVersion: 1,
			ID:            "macos-zsh-path-repair",
			Title:         "PATH",
			SupportedOS:   []string{"darwin"},
			Risk:          recipe.RiskReversiblePatch,
			AutoAllowed:   true,
			Preconditions: []recipe.Precondition{{Type: "file_exists_or_creatable", Path: "~/.zshrc"}},
			Patches:       []recipe.Patch{{ID: "path", Type: "append_managed_block_if_missing", Path: "~/.zshrc", Marker: "PATH_BLOCK", Content: "export PATH=\"$HOME/bin:$PATH\""}},
			Verify:        []recipe.VerifierRef{{ID: "managed_block_count", Args: map[string]string{"path": "~/.zshrc", "marker": "PATH_BLOCK", "count": "1"}}},
			Rollback:      []recipe.RollbackSpec{{Type: "restore_file", Path: "~/.zshrc"}},
		},
		"api-key-detection-redaction": {
			SchemaVersion: 1,
			ID:            "api-key-detection-redaction",
			Title:         "Keys",
			SupportedOS:   []string{"darwin"},
			Risk:          recipe.RiskReadOnly,
			AutoAllowed:   true,
			Verify:        []recipe.VerifierRef{{ID: "env_key_present_redacted", Args: map[string]string{"env": "DEEPSEEK_API_KEY", "required": "false"}}},
		},
	})
}

func TestModelManifestAndAssetResolution(t *testing.T) {
	m, err := LoadModelManifest(filepath.Join("..", "..", "assets", "manifests", "qwen3-4b-instruct-2507-q4km.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != DefaultModelID || m.SHA256 == "" {
		t.Fatalf("bad manifest: %+v", m)
	}
	home := t.TempDir()
	model := filepath.Join(home, "model.gguf")
	if err := os.WriteFile(model, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTLINK_MODEL_PATH", model)
	loc := LocateAssets(home, m)
	if loc.ModelPath != model || !loc.ModelExists {
		t.Fatalf("env model override not used: %+v", loc)
	}
}

func TestLlamaCLICommandConstructionAndPrompt(t *testing.T) {
	args := LlamaCLIArgs("/m.gguf", "prompt", 64, 2048, 0)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-m /m.gguf", "-p prompt", "-n 64", "-c 2048", "--temp 0", "-st", "--no-display-prompt", "--simple-io"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing arg %s in %v", want, args)
		}
	}
	prompt := FormatChatML("sys", "user")
	if !strings.Contains(prompt, "<|im_start|>system\nsys<|im_end|>") || !strings.HasSuffix(prompt, "<|im_start|>assistant\n") {
		t.Fatalf("bad ChatML prompt: %q", prompt)
	}
}

func TestJSONExtraction(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "qwen_output_with_text_around_json.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ExtractJSONObject(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"intent": "report"`) {
		t.Fatalf("bad extraction: %s", got)
	}
	if _, err := ExtractJSONObject(`{"a":1} {"b":2}`); err == nil {
		t.Fatal("multiple JSON objects accepted")
	}
	if _, err := ExtractJSONObject("not json"); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	runtimeOutput := `tokenizer.chat_template str = {%- if tools %}
{"schemaVersion":1,"intent":"report","failureClass":"UNKNOWN","confidence":0.8,"selectedRecipe":{"id":"","params":{}},"risk":"read_only","requiresUserApproval":false,"expectedVerifiers":[],"fallbackRecipes":[],"explanationForUser":"ok","evidence":[],"stopReason":"done"} [end of text]`
	plannerJSON, err := ExtractPlannerJSONObject(runtimeOutput)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plannerJSON, `"stopReason":"done"`) {
		t.Fatalf("did not extract planner JSON from runtime output: %s", plannerJSON)
	}
}

func TestLlamaAssistantOutputIgnoresEchoedPrompt(t *testing.T) {
	raw := `prompt {"schemaVersion":0}
<|im_start|>assistant
{"schemaVersion":1,"intent":"report"}`
	got, err := ExtractJSONObject(llamaAssistantOutput(raw))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, `"schemaVersion":0`) || !strings.Contains(got, `"schemaVersion":1`) {
		t.Fatalf("did not extract assistant JSON: %s", got)
	}
}

func TestBrainPlannerValidationAndCorrection(t *testing.T) {
	valid, _ := os.ReadFile(filepath.Join("testdata", "qwen_planner_valid.json"))
	fb := &fakeBackend{texts: []string{"invalid", string(valid)}}
	p := NewPlanner(fb, testRecipeRegistry())
	plan, err := p.Plan(context.Background(), PlanInput{Target: "path", Facts: facts.Facts{OS: "darwin"}})
	if err != nil {
		t.Fatal(err)
	}
	if fb.calls != 2 || len(plan.Calls) != 2 {
		t.Fatalf("expected bounded correction attempt, calls=%d plan=%+v", fb.calls, plan)
	}
	if plan.Decision.SelectedRecipe.ID != "macos-zsh-path-repair" {
		t.Fatalf("bad decision: %+v", plan.Decision)
	}
	low, _ := os.ReadFile(filepath.Join("testdata", "qwen_planner_low_confidence.json"))
	fb = &fakeBackend{texts: []string{string(low), string(low)}}
	_, err = NewPlanner(fb, testRecipeRegistry()).Plan(context.Background(), PlanInput{Target: "path"})
	if err == nil {
		t.Fatal("low confidence repair accepted")
	}
}

func TestBrainLoopDryRunAndYesPathRepair(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	valid, _ := os.ReadFile(filepath.Join("testdata", "qwen_planner_valid.json"))
	reg := testRecipeRegistry()
	backend := &fakeBackend{texts: []string{string(valid)}}
	pl := NewPlanner(backend, reg)
	plan, err := pl.Plan(context.Background(), PlanInput{Target: "path", Home: home})
	if err != nil || plan.Decision.SelectedRecipe.ID == "" {
		t.Fatalf("plan failed: %+v %v", plan, err)
	}
	runner := &command.MockRunner{}
	dry := recipe.Run(context.Background(), runner, reg, plan.Decision.SelectedRecipe.ID, recipe.RunOptions{Home: home, DryRun: true})
	if dry.Status != recipe.StatusDryRun {
		t.Fatalf("dry run failed: %+v", dry)
	}
	if system.Exists(filepath.Join(home, ".zshrc")) {
		t.Fatal("dry-run modified temp HOME")
	}
	yesBackend := &fakeBackend{texts: []string{string(valid)}}
	yes := RunLoopWithBackendForTest(context.Background(), runner, reg, yesBackend, LoopOptions{Home: home, Target: "path", Yes: true, CommandLine: []string{"repair", "--auto", "--brain"}})
	if yes.Status != recipe.StatusSuccess {
		t.Fatalf("brain loop yes failed: %+v", yes)
	}
	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "AGENTLINK_PATH_BLOCK") != 2 {
		t.Fatalf("managed block not written once:\n%s", data)
	}
	rp, err := snapshot.Latest(system.UserRestorePointsDir(home))
	if err != nil {
		t.Fatal(err)
	}
	rp.SetMutationPolicy(system.MutationOptions{RealUserHome: home, ExtraAllowedPaths: []string{filepath.Join(home, ".zshrc")}})
	if err := rp.RestoreAllWithRunner(context.Background(), runner); err != nil {
		t.Fatal(err)
	}
	if system.Exists(filepath.Join(home, ".zshrc")) {
		t.Fatal("rollback did not restore absent .zshrc state")
	}
}

func TestBrainAutoPolicyRefusesWithoutYesAndRisk(t *testing.T) {
	rec, _ := testRecipeRegistry().Get("macos-zsh-path-repair")
	dec := planner.Decision{SchemaVersion: 1, Intent: "repair", SelectedRecipe: planner.SelectedRecipe{ID: rec.ID}, Risk: recipe.RiskReversiblePatch, Confidence: 0.9}
	rec.Risk = recipe.RiskReversiblePatch
	if err := ValidateAutoExecution(dec, rec, AutoPolicy{}); err == nil {
		t.Fatal("reversible patch accepted without --yes")
	}
	rec.Risk = recipe.RiskPrivilegedAction
	if err := ValidateAutoExecution(dec, rec, AutoPolicy{Yes: true}); err == nil {
		t.Fatal("privileged action accepted")
	}
	rec.Risk = recipe.RiskNetworkAction
	if err := ValidateAutoExecution(dec, rec, AutoPolicy{Yes: true}); err == nil {
		t.Fatal("network action accepted without online")
	}
}

func TestBrainDoctorMissingAssetsWarns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AGENTLINK_MODEL_PATH", filepath.Join(home, "missing.gguf"))
	t.Setenv("AGENTLINK_LLAMA_CLI", filepath.Join(home, "missing-llama"))
	doc := Doctor(context.Background(), &command.MockRunner{}, home)
	if doc.BrainPackAvailable || len(doc.MissingAssets) == 0 || len(doc.FetchCommands) == 0 {
		t.Fatalf("expected missing assets warning: %+v", doc)
	}
}

func TestPlannerOutputRedaction(t *testing.T) {
	secret := "sk-test-THIS_SHOULD_NOT_LEAK"
	t.Setenv("DEEPSEEK_API_KEY", secret)
	raw := `{"schemaVersion":1,"intent":"report","failureClass":"UNKNOWN","confidence":0.8,"selectedRecipe":{"id":"","params":{}},"risk":"read_only","requiresUserApproval":false,"expectedVerifiers":[],"fallbackRecipes":[],"explanationForUser":"REDACTED","evidence":["Authorization: Bearer ` + secret + `"],"stopReason":"report"}`
	fb := &fakeBackend{texts: []string{raw, raw}}
	_, err := NewPlanner(fb, testRecipeRegistry()).Plan(context.Background(), PlanInput{Target: "keys"})
	if err == nil {
		t.Fatal("secret-bearing decision accepted")
	}
}

func TestBrainRequestTimeoutType(t *testing.T) {
	req := BrainRequest{Timeout: 2 * time.Second}
	if req.Timeout <= 0 {
		t.Fatal("timeout not set")
	}
}
