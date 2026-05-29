package orchestrator

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/diagnosisgraph"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/verifier"
)

// repoRecipesDir is the real recipe catalog, relative to this package dir.
const repoRecipesDir = "../../recipes"

// TestGraphChainProjectsRankedPrimaryClass proves diagnose -> classify ->
// graph is wired: a classified report projects into a diagnosis graph that
// exposes a ranked primary class and a repair-grade confidence (not a flat,
// unordered class set).
func TestGraphChainProjectsRankedPrimaryClass(t *testing.T) {
	diag := diagnose.DiagnosticReport{
		Classifications: []string{classify.UserProxyDirty},
		Network:         diagnose.NetworkInfo{ProxySummary: diagnose.ProxySummary{Dirty: true}},
	}
	g := diagnosisgraph.Build(diag, true)
	if g.PrimaryClass != classify.UserProxyDirty {
		t.Fatalf("graph primary = %q, want %q", g.PrimaryClass, classify.UserProxyDirty)
	}
	if g.Confidence < planner.RepairConfidenceFloor {
		t.Fatalf("graph confidence %.2f below repair floor %.2f", g.Confidence, planner.RepairConfidenceFloor)
	}
}

// TestReversibleRepairPlanRoutesReversibleClass proves recipe -> planner ->
// orchestrator -> verifier -> safety end to end against the REAL recipe
// catalog: a reversible orphan class (USER_PROXY_DIRTY) that the deterministic
// TUN supervisor cannot handle now routes through the registry-driven planner
// to its bounded reversible recipe, and the decision survives the
// registry+verifier safety validator.
func TestReversibleRepairPlanRoutesReversibleClass(t *testing.T) {
	reg, err := recipe.LoadRegistry(repoRecipesDir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	diag := diagnose.DiagnosticReport{
		Classifications: []string{classify.UserProxyDirty},
		Network:         diagnose.NetworkInfo{ProxySummary: diagnose.ProxySummary{Dirty: true}},
	}
	g := diagnosisgraph.Build(diag, true)
	pd, ok := reversibleRepairPlan(g, reg, verifier.NewRegistry())
	if !ok {
		t.Fatalf("expected executable reversible repair, got intent=%q stop=%q", pd.Intent, pd.StopReason)
	}
	if pd.SelectedRecipe.ID != "proxy-clean-stale-env" {
		t.Fatalf("routed to %q, want proxy-clean-stale-env", pd.SelectedRecipe.ID)
	}
	if pd.Risk != recipe.RiskReversiblePatch || !pd.RequiresUserApproval {
		t.Fatalf("reversible repair must carry reversible risk + user approval: %+v", pd)
	}
	if planner.ValidateDecision(pd, reg, verifier.NewRegistry()) != nil {
		t.Fatal("routed decision did not survive the safety validator")
	}
}

// TestReversibleRepairPlanRefusesPrivilegedClass proves the executor split is
// preserved: a class routed only to a privileged recipe (NO_DEFAULT_ROUTE ->
// clean-network-baseline-reset) must NOT be sent to the reversible recipe.Run
// executor — it belongs to the deterministic supervisor / ticket path.
func TestReversibleRepairPlanRefusesPrivilegedClass(t *testing.T) {
	reg, err := recipe.LoadRegistry(repoRecipesDir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	g := diagnosisgraph.Build(diagnose.DiagnosticReport{
		Classifications: []string{classify.NoDefaultRoute},
	}, true)
	if pd, ok := reversibleRepairPlan(g, reg, verifier.NewRegistry()); ok {
		t.Fatalf("privileged-only class routed to reversible executor: %+v", pd)
	}
}

// TestRunReversibleRepairDryRunReportsPlanned proves the dispatch glue fires:
// runReversibleRepair drives recipe.Run (the verifier+rollback executor) in
// dry-run and maps the result onto a coherent orchestrator report with planner
// provenance and both decision + execute cycles.
func TestRunReversibleRepairDryRunReportsPlanned(t *testing.T) {
	reg, err := recipe.LoadRegistry(repoRecipesDir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	pd := planner.Decision{
		SchemaVersion:        1,
		Intent:               "repair",
		FailureClass:         classify.UserProxyDirty,
		Confidence:           0.85,
		SelectedRecipe:       planner.SelectedRecipe{ID: "proxy-clean-stale-env", Params: map[string]string{}},
		Risk:                 recipe.RiskReversiblePatch,
		RequiresUserApproval: true,
	}
	base := Report{SchemaVersion: 1, Status: "failed", Cycles: []Cycle{}, FailureClasses: []string{classify.UserProxyDirty}}
	rep := runReversibleRepair(context.Background(), &command.MockRunner{}, reg, pd, Options{DryRun: true, Home: t.TempDir()}, base)
	if rep.Status != "planned" {
		t.Fatalf("dry-run status = %q, want planned", rep.Status)
	}
	if rep.PlanSource != "planner" || rep.SelectedRecipe != "proxy-clean-stale-env" {
		t.Fatalf("planner dispatch metadata wrong: source=%q recipe=%q", rep.PlanSource, rep.SelectedRecipe)
	}
	if !hasCycle(rep, "PlannerDecision") || !hasCycle(rep, "DryRunOrExecute") {
		t.Fatalf("missing planner/execute cycles: %+v", rep.Cycles)
	}
}

func hasCycle(rep Report, state string) bool {
	for _, c := range rep.Cycles {
		if c.State == state {
			return true
		}
	}
	return false
}
