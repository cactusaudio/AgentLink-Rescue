package planner

import (
	"testing"

	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/verifier"
)

func planRegistry() recipe.Registry {
	return recipe.RegistryForTest(map[string]recipe.Recipe{
		"reversible-proxy": {
			SchemaVersion:  1,
			ID:             "reversible-proxy",
			Title:          "Reversible proxy clean",
			FailureClasses: []string{"USER_PROXY_DIRTY"},
			Risk:           recipe.RiskReversiblePatch,
			Patches:        []recipe.Patch{{Type: "append_managed_block_if_missing", Path: "~/.zshrc", Marker: "X", Content: "x"}},
			Rollback:       []recipe.RollbackSpec{{Type: "restore_file", Path: "~/.zshrc"}},
			Verify:         []recipe.VerifierRef{{ID: "managed_block_count"}},
		},
		"privileged-net": {
			SchemaVersion:  1,
			ID:             "privileged-net",
			Title:          "Privileged net reset",
			FailureClasses: []string{"NO_DEFAULT_ROUTE"},
			Risk:           recipe.RiskPrivilegedAction,
			RequiresRoot:   true,
		},
		"readonly-keys": {
			SchemaVersion:  1,
			ID:             "readonly-keys",
			Title:          "Read-only keys",
			FailureClasses: []string{"KEYS_STATUS_UNKNOWN"},
			Risk:           recipe.RiskReadOnly,
			Verify:         []recipe.VerifierRef{{ID: "managed_block_count"}},
		},
	})
}

func TestPlanReversibleHighConfidenceRepairs(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "USER_PROXY_DIRTY", Confidence: 0.8}, reg)
	if d.Intent != "repair" {
		t.Fatalf("expected repair, got %q (%s)", d.Intent, d.StopReason)
	}
	if d.SelectedRecipe.ID != "reversible-proxy" {
		t.Fatalf("expected reversible-proxy, got %q", d.SelectedRecipe.ID)
	}
	if d.Risk != recipe.RiskReversiblePatch || !d.RequiresUserApproval {
		t.Fatalf("writable repair must carry reversible risk + user approval: %+v", d)
	}
	if len(d.ExpectedVerifiers) == 0 || d.ExpectedVerifiers[0] != "managed_block_count" {
		t.Fatalf("expected verifiers not propagated: %+v", d.ExpectedVerifiers)
	}
	// The distilled decision must survive the safety validator unchanged.
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("planned decision rejected by validator: %v", err)
	}
}

func TestPlanReversibleLowConfidenceProbes(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "USER_PROXY_DIRTY", Confidence: 0.4}, reg)
	if d.Intent != "probe" {
		t.Fatalf("low-confidence reversible class must probe, not %q", d.Intent)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("probe decision rejected: %v", err)
	}
}

func TestPlanPrivilegedOnlyClassDefersToReport(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "NO_DEFAULT_ROUTE", Confidence: 0.9}, reg)
	if d.Intent != "report" {
		t.Fatalf("privileged-only class must defer to report, got %q", d.Intent)
	}
	if d.StopReason == "" {
		t.Fatalf("deferral must carry a stop reason: %+v", d)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("report decision rejected: %v", err)
	}
}

func TestPlanReadOnlyClassProbes(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "KEYS_STATUS_UNKNOWN", Confidence: 0.9}, reg)
	if d.Intent != "probe" {
		t.Fatalf("read-only class must probe, got %q", d.Intent)
	}
}

func TestPlanNoRecipeReports(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "SYSTEM_PROXY_DIRTY", Confidence: 0.9}, reg)
	if d.Intent != "report" {
		t.Fatalf("class with no routed recipe must report, got %q", d.Intent)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("report decision rejected: %v", err)
	}
}

func TestPlanProtectedTopologyNeverMutates(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "USER_PROXY_DIRTY", Confidence: 0.95, ProtectedTopology: true}, reg)
	if d.Intent == "repair" {
		t.Fatalf("protected topology must never route to repair: %+v", d)
	}
	if d.Intent != "report" {
		t.Fatalf("protected topology must report, got %q", d.Intent)
	}
}

func TestPlanEmptyClassReports(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: "", Confidence: 0.9}, reg)
	if d.Intent != "report" {
		t.Fatalf("empty class must report, got %q", d.Intent)
	}
}
