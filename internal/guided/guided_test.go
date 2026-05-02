package guided

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

type fakeBackend struct {
	text      string
	available bool
}

func (f fakeBackend) Name() string { return "fake" }

func (f fakeBackend) Available(context.Context) brain.BrainAvailability {
	return brain.BrainAvailability{BrainPackAvailable: f.available, Backend: "fake", ModelID: brain.DefaultModelID, RuntimeExecutable: f.available, ModelExists: f.available, ModelSHA256OK: f.available}
}

func (f fakeBackend) Generate(context.Context, brain.BrainRequest) (brain.BrainResponse, error) {
	return brain.BrainResponse{RawText: f.text, Backend: "fake", ModelPath: "/tmp/fake.gguf"}, nil
}

func testRegistry(t *testing.T) recipe.Registry {
	t.Helper()
	reg, err := recipe.LoadRegistry(filepath.Join("..", "..", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func healthyFacts() facts.Facts {
	return facts.Facts{
		OS:          "darwin",
		Arch:        "arm64",
		ConfigPaths: map[string]facts.PathFact{"codex": {Path: "~/.codex/config.toml"}},
		Network: diagnose.DiagnosticReport{
			Classifications:        []string{classify.OK},
			RecommendedRepairLevel: "none",
		},
	}
}

func unknownFacts() facts.Facts {
	f := healthyFacts()
	f.Network.Classifications = []string{classify.Unknown}
	f.Network.RecommendedRepairLevel = ""
	return f
}

func TestGuidedReportJSONMarshals(t *testing.T) {
	rep := Report{SchemaVersion: 1, ToolVersion: system.Version, Target: "path", Status: StatusDryRunComplete}
	if err := ValidateReport(rep); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(MarshalReport(rep), `"status": "dry_run_complete"`) {
		t.Fatal("guided JSON did not contain status")
	}
}

func TestGuidedHealthyAutoStopsWithoutRepair(t *testing.T) {
	home := t.TempDir()
	rep := RunForTest(context.Background(), nil, testRegistry(t), Options{Home: home, Target: "auto", DryRun: true}, healthyFacts(), fakeBackend{})
	if rep.Status != StatusHealthy {
		t.Fatalf("status=%s report=%+v", rep.Status, rep)
	}
	if rep.SelectedRecipe != "" {
		t.Fatalf("healthy auto selected recipe: %s", rep.SelectedRecipe)
	}
}

func TestGuidedPathDryRunDoesNotMutate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("# original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	rep := RunForTest(context.Background(), nil, testRegistry(t), Options{Home: home, Target: "path", DryRun: true}, healthyFacts(), fakeBackend{})
	if rep.Status != StatusDryRunComplete {
		t.Fatalf("status=%s report=%+v", rep.Status, rep)
	}
	data, _ := os.ReadFile(zshrc)
	if string(data) != "# original\n" {
		t.Fatalf("dry-run mutated zshrc: %q", data)
	}
	if rep.SelectedRecipe != "macos-zsh-path-repair" {
		t.Fatalf("recipe=%s", rep.SelectedRecipe)
	}
}

func TestGuidedPathYesSnapshotsAndIsRollbackable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("# original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	rep := RunForTest(context.Background(), nil, testRegistry(t), Options{Home: home, Target: "path", Yes: true}, healthyFacts(), fakeBackend{})
	if rep.Status != StatusRepaired {
		t.Fatalf("status=%s report=%+v", rep.Status, rep)
	}
	if !rep.RollbackAvailable || rep.SnapshotID == "" {
		t.Fatalf("rollback not available: %+v", rep)
	}
	after, _ := os.ReadFile(zshrc)
	if strings.Count(string(after), "AGENTLINK_PATH_BLOCK") != 2 {
		t.Fatalf("managed block missing or duplicated:\n%s", after)
	}
	rp, err := snapshot.Load(filepath.Join(system.UserRestorePointsDir(home), rep.SnapshotID))
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

func TestGuidedNetworkTargetNeverRunsRescue(t *testing.T) {
	home := t.TempDir()
	rep := RunForTest(context.Background(), nil, testRegistry(t), Options{Home: home, Target: "network", DryRun: true}, unknownFacts(), fakeBackend{available: true})
	if rep.Status != StatusManualActionRequired {
		t.Fatalf("status=%s report=%+v", rep.Status, rep)
	}
	if rep.SelectedRecipe != "" || !strings.Contains(rep.NextSafeCommand, "rescue --level safe") {
		t.Fatalf("network target should only report copyable command: %+v", rep)
	}
}

func TestGuidedRejectsLowConfidenceBrainRepair(t *testing.T) {
	decision := `{"schemaVersion":1,"intent":"repair","failureClass":"UNKNOWN","confidence":0.2,"selectedRecipe":{"id":"macos-zsh-path-repair","params":{}},"risk":"reversible_patch","requiresUserApproval":true,"expectedVerifiers":["managed_block_count"],"fallbackRecipes":[],"explanationForUser":"low confidence","evidence":[],"stopReason":""}`
	rep := RunForTest(context.Background(), nil, testRegistry(t), Options{Home: t.TempDir(), Target: "auto", DryRun: true}, unknownFacts(), fakeBackend{text: decision, available: true})
	if rep.Status != StatusManualActionRequired {
		t.Fatalf("low-confidence decision should be refused, got %+v", rep)
	}
	if !rep.PlannerUsed {
		t.Fatal("expected planner to be used")
	}
}
