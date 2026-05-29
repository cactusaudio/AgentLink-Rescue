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
	pd, kind := plannerRepairPlan(g, reg, verifier.NewRegistry())
	if kind != "recipe" {
		t.Fatalf("expected reversible recipe route, got kind=%q intent=%q stop=%q", kind, pd.Intent, pd.StopReason)
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

// TestPlannerRepairPlanRoutesNetworkStateClassToLevel proves the executor split:
// a system network-state class (NO_DEFAULT_ROUTE) routes to the privileged
// repair.Run level executor (not the reversible recipe.Run path), and the
// graduated tier (standard) is chosen rather than a heavy clean-baseline recipe.
func TestPlannerRepairPlanRoutesNetworkStateClassToLevel(t *testing.T) {
	reg, err := recipe.LoadRegistry(repoRecipesDir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	g := diagnosisgraph.Build(diagnose.DiagnosticReport{
		Classifications: []string{classify.NoDefaultRoute},
	}, true)
	pd, kind := plannerRepairPlan(g, reg, verifier.NewRegistry())
	if kind != "level" {
		t.Fatalf("NO_DEFAULT_ROUTE must route to the repair-level executor, got kind=%q", kind)
	}
	if pd.RepairLevel != planner.LevelStandard {
		t.Fatalf("expected standard tier, got %q", pd.RepairLevel)
	}
	if pd.SelectedRecipe.ID != "" {
		t.Fatalf("level decision must not also name a recipe: %+v", pd)
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

// TestRunLevelRepairDryRunReportsPlanned proves the network-state executor:
// runLevelRepair drives the privileged repair engine in dry-run for a
// network-state class and reports a planned, admin-gated, snapshot-reversible
// repair with planner provenance.
func TestRunLevelRepairDryRunReportsPlanned(t *testing.T) {
	pd := planner.Decision{
		SchemaVersion: 1, Intent: "repair", FailureClass: classify.SystemProxyDirty,
		Confidence: 0.85, RepairLevel: planner.LevelSafe, Risk: recipe.RiskNetworkAction, RequiresUserApproval: true,
	}
	base := Report{SchemaVersion: 1, Status: "failed", Cycles: []Cycle{}, FailureClasses: []string{classify.SystemProxyDirty}}
	rep := runLevelRepair(context.Background(), &command.MockRunner{}, pd, Options{DryRun: true, Home: t.TempDir()}, base)
	if rep.Status != "planned" {
		t.Fatalf("dry-run level repair status = %q, want planned", rep.Status)
	}
	if rep.PlanSource != "planner" || !rep.RequiresAdmin {
		t.Fatalf("level repair provenance wrong: source=%q admin=%v", rep.PlanSource, rep.RequiresAdmin)
	}
	if !hasCycle(rep, "PlannerDecision") || !hasCycle(rep, "DryRunOrExecute") {
		t.Fatalf("missing planner/execute cycles: %+v", rep.Cycles)
	}
}

// TestGenomeAdvisoryAttachesForNetworkClass proves knowledge integration: the
// orchestrator surfaces distilled genome cards for the primary class as
// read-only report context (genome in the loop), best-effort.
func TestGenomeAdvisoryAttachesForNetworkClass(t *testing.T) {
	g := diagnosisgraph.Build(diagnose.DiagnosticReport{
		Classifications: []string{classify.SystemProxyDirty},
		Network:         diagnose.NetworkInfo{ProxySummary: diagnose.ProxySummary{Dirty: true}},
	}, true)
	adv := genomeAdvisory(g, "")
	if adv == nil {
		t.Skip("genome corpus not reachable from test cwd; advisory is best-effort")
	}
	if adv.Layer != "L05_proxy" || len(adv.CardIDs) == 0 {
		t.Fatalf("SYSTEM_PROXY_DIRTY advisory wrong: %+v", adv)
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
