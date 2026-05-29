package planner

import (
	"testing"

	"cactus-agentlink-rescue/internal/classify"
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

func TestPlanNetworkStateClassRoutesToSafeLevel(t *testing.T) {
	reg := planRegistry()
	// SYSTEM_PROXY_DIRTY has no user-config recipe -> privileged safe-tier repair.
	d := Plan(Inputs{PrimaryClass: classify.SystemProxyDirty, Confidence: 0.85}, reg)
	if d.Intent != "repair" || d.RepairLevel != LevelSafe {
		t.Fatalf("SYSTEM_PROXY_DIRTY must route to safe-tier repair, got intent=%q level=%q", d.Intent, d.RepairLevel)
	}
	if d.Risk != recipe.RiskNetworkAction || !d.RequiresUserApproval || d.SelectedRecipe.ID != "" {
		t.Fatalf("safe network repair shape wrong: %+v", d)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("safe-level decision rejected by validator: %v", err)
	}
}

func TestPlanPrivilegedNetworkClassRoutesToStandardLevel(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: classify.NoDefaultRoute, Confidence: 0.9}, reg)
	if d.Intent != "repair" || d.RepairLevel != LevelStandard {
		t.Fatalf("NO_DEFAULT_ROUTE must route to standard-tier repair, got intent=%q level=%q", d.Intent, d.RepairLevel)
	}
	if d.Risk != recipe.RiskPrivilegedAction || !d.RequiresUserApproval {
		t.Fatalf("standard network repair must be privileged + consented: %+v", d)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("standard-level decision rejected by validator: %v", err)
	}
}

func TestPlanReportOnlyClassStaysReadOnly(t *testing.T) {
	reg := planRegistry()
	// MDM profiles need user/admin action; never auto-routed to a network repair.
	d := Plan(Inputs{PrimaryClass: classify.MDMProfileSuspected, Confidence: 0.95}, reg)
	if d.Intent != "report" {
		t.Fatalf("MDM_PROFILE_SUSPECTED must stay read-only, got %q (level %q)", d.Intent, d.RepairLevel)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("report decision rejected: %v", err)
	}
}

func TestPlanLowConfidenceNetworkClassDoesNotRepair(t *testing.T) {
	reg := planRegistry()
	d := Plan(Inputs{PrimaryClass: classify.SystemProxyDirty, Confidence: 0.4}, reg)
	if d.Intent == "repair" {
		t.Fatalf("low-confidence network class must not repair: %+v", d)
	}
}

func TestPlanNetworkTierSpreadAndProtectedRefusal(t *testing.T) {
	reg, err := recipe.LoadRegistry("../../recipes")
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	cases := []struct {
		class, level, intent string
	}{
		{classify.SystemProxyDirty, LevelSafe, "repair"},
		{classify.DNSFail, LevelSafe, "repair"},
		{classify.NoDefaultRoute, LevelStandard, "repair"},
		{classify.SysconfigSuspected, LevelDeep, "repair"},
		{classify.ProtectedAudioVLANRouteTrap, "", "report"},
		{classify.ProtectedTopologyConstraint, "", "report"},
		{classify.MDMProfileSuspected, "", "report"},
	}
	for _, c := range cases {
		d := Plan(Inputs{PrimaryClass: c.class, Confidence: 0.9}, reg)
		if d.Intent != c.intent || d.RepairLevel != c.level {
			t.Fatalf("%s: got intent=%q level=%q, want intent=%q level=%q", c.class, d.Intent, d.RepairLevel, c.intent, c.level)
		}
		if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
			t.Fatalf("%s: decision rejected: %v", c.class, err)
		}
	}
}

func TestPlanBrewProxyRoutesToReversibleRecipe(t *testing.T) {
	reg, err := recipe.LoadRegistry("../../recipes")
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	d := Plan(Inputs{PrimaryClass: classify.BrewProxyDirty, Confidence: 0.8}, reg)
	if d.Intent != "repair" || d.SelectedRecipe.ID != "proxy-clean-stale-env" {
		t.Fatalf("BREW_PROXY_DIRTY must route to the reversible env-proxy recipe, got intent=%q recipe=%q", d.Intent, d.SelectedRecipe.ID)
	}
	if d.RepairLevel != "" {
		t.Fatalf("config-recipe decision must not also set a repair level: %+v", d)
	}
	if err := ValidateDecision(d, reg, verifier.NewRegistry()); err != nil {
		t.Fatalf("brew proxy decision rejected: %v", err)
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
	// An upstream agent-endpoint block has no host-side repair (escalate
	// upstream); it must stay read-only.
	d := Plan(Inputs{PrimaryClass: classify.GeneralInternetOKAgentEndpointBlocked, Confidence: 0.9}, reg)
	if d.Intent != "report" {
		t.Fatalf("class with no routed repair must report, got %q (level %q)", d.Intent, d.RepairLevel)
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

// TestPlanNoOrphanClassAcrossTaxonomy proves routing-topology completeness:
// every deterministic class classify can emit resolves to a valid,
// class-labeled decision against the REAL recipe catalog (recipe-backed classes
// route to repair; the rest get an explicit class-labeled report) — never a
// generic UNKNOWN orphan.
func TestPlanNoOrphanClassAcrossTaxonomy(t *testing.T) {
	reg, err := recipe.LoadRegistry("../../recipes")
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	vreg := verifier.NewRegistry()
	for _, c := range classify.AllClasses() {
		d := Plan(Inputs{PrimaryClass: c, Classes: []string{c}, Confidence: 0.85}, reg)
		switch d.Intent {
		case "repair", "probe", "report":
		default:
			t.Fatalf("class %s orphaned: unrecognized intent %q", c, d.Intent)
		}
		if d.FailureClass != c {
			t.Fatalf("class %s lost its label in the decision: %q", c, d.FailureClass)
		}
		if err := ValidateDecision(d, reg, vreg); err != nil {
			t.Fatalf("class %s produced an invalid decision: %v", c, err)
		}
	}
}
