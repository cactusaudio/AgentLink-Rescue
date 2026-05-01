package planner

import (
	"testing"

	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/verifier"
)

func plannerRegistry() recipe.Registry {
	return recipe.RegistryForTest(map[string]recipe.Recipe{
		"r": {
			SchemaVersion: 1,
			ID:            "r",
			Title:         "R",
			SupportedOS:   []string{"darwin"},
			Risk:          recipe.RiskReversiblePatch,
			Patches:       []recipe.Patch{{Type: "append_managed_block_if_missing", Path: "~/.zshrc", Marker: "X", Content: "x"}},
			Rollback:      []recipe.RollbackSpec{{Type: "restore_file", Path: "~/.zshrc"}},
			Params:        []recipe.ParamSpec{{Name: "mode", Default: "a"}},
		},
	})
}

func TestPlannerInvalidCasesRejected(t *testing.T) {
	reg := plannerRegistry()
	vreg := verifier.NewRegistry()
	valid := Decision{SchemaVersion: 1, Intent: "repair", Confidence: 0.8, SelectedRecipe: SelectedRecipe{ID: "r", Params: map[string]string{"mode": "a"}}, Risk: recipe.RiskSafePatch, ExpectedVerifiers: []string{"managed_block_count"}}
	if err := ValidateDecision(valid, reg, vreg); err != nil {
		t.Fatalf("valid decision rejected: %v", err)
	}
	low := valid
	low.Confidence = 0.4
	if err := ValidateDecision(low, reg, vreg); err == nil {
		t.Fatal("low-confidence repair accepted")
	}
	unknown := valid
	unknown.SelectedRecipe.ID = "missing"
	if err := ValidateDecision(unknown, reg, vreg); err == nil {
		t.Fatal("unknown recipe accepted")
	}
	badVerifier := valid
	badVerifier.ExpectedVerifiers = []string{"missing_verifier"}
	if err := ValidateDecision(badVerifier, reg, vreg); err == nil {
		t.Fatal("unknown verifier accepted")
	}
}
