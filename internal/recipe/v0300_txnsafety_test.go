package recipe

import (
	"path/filepath"
	"runtime"
	"testing"

	"cactus-agentlink-rescue/internal/toolmanifest"
)

// repoRecipesDir resolves the real repo recipes/ dir regardless of the
// test working directory (the package-relative FindRecipesDir() does
// not resolve from `go test ./internal/recipe/`).
func repoRecipesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	// internal/recipe/v0300_txnsafety_test.go -> repo root is ../../
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "recipes")
}

// V0300 W4: every mutating recipe must carry transaction-safety
// metadata (preconditions + postverify), and reversible-class recipes
// MUST declare rollback. Higher-risk classes may omit automatic
// rollback only because their risk class already signals that.
func TestV0300AllMutatingRecipesAreTransactionSafe(t *testing.T) {
	reg, err := LoadRegistry(repoRecipesDir(t))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	list := reg.List()
	if len(list) == 0 {
		t.Fatal("no recipes loaded")
	}
	validRisk := map[string]bool{
		RiskReadOnly: true, RiskSafePatch: true, RiskReversiblePatch: true,
		RiskNetworkAction: true, RiskPrivilegedAction: true,
		RiskDestructiveAction: true,
	}
	for _, r := range list {
		if r.ID == "" || r.Title == "" || r.Description == "" {
			t.Errorf("recipe %q: missing id/title/description", r.ID)
		}
		if !validRisk[r.Risk] {
			t.Errorf("recipe %s: invalid risk %q", r.ID, r.Risk)
		}
		if len(r.SupportedOS) == 0 {
			t.Errorf("recipe %s: empty supportedOS", r.ID)
		}
		if r.Risk == RiskReadOnly {
			continue // read-only recipes need no rollback
		}
		// mutating recipe
		if len(r.Preconditions) == 0 {
			t.Errorf("recipe %s (%s): mutating recipe has no preconditions (preflight)", r.ID, r.Risk)
		}
		if len(r.Verify) == 0 {
			t.Errorf("recipe %s (%s): mutating recipe has no verify (postverify)", r.ID, r.Risk)
		}
		switch r.Risk {
		case RiskSafePatch, RiskReversiblePatch:
			if len(r.Rollback) == 0 {
				t.Errorf("recipe %s (%s): reversible-class recipe MUST declare rollback", r.ID, r.Risk)
			}
		case RiskNetworkAction, RiskPrivilegedAction, RiskDestructiveAction:
			// rollback may be impossible; the high-risk class is the
			// declared signal. If it DOES claim rollback, fine.
		}
	}
}

// V0300 W4: the W2 manifest's recipe-backed execution tools must bind
// to REAL recipes in the registry (no dangling — closes the W2
// integration gap caught in W4 by truth-vs-harness).
func TestV0300ManifestExecToolsBindRealRecipes(t *testing.T) {
	reg, err := LoadRegistry(repoRecipesDir(t))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	for toolID, recipeID := range toolmanifest.RecipeBackedExecTools {
		if !reg.Exists(recipeID) {
			t.Errorf("manifest exec tool %s references non-existent recipe %q", toolID, recipeID)
		}
	}
	// CLI-backed execution tools must reference a real AgentLink
	// transactional subcommand (journal/snapshot-backed, not raw shell).
	realSub := map[string]bool{"last-good": true, "rollback": true, "journal": true, "package": true}
	for toolID, argv := range toolmanifest.CLIBackedExecTools {
		if len(argv) == 0 || !realSub[argv[0]] {
			t.Errorf("CLI-backed exec tool %s has non-real subcommand argv=%v", toolID, argv)
		}
	}
	// every execution ToolCard whose argv is `recipe run <id> ...`
	// must point at a real recipe (skip the generic placeholder).
	m := toolmanifest.Build("test")
	for _, tc := range m.Tools {
		if tc.MutationClass != toolmanifest.MutationHostTxn {
			continue
		}
		if len(tc.Argv) >= 3 && tc.Argv[0] == "recipe" && tc.Argv[1] == "run" {
			id := tc.Argv[2]
			if id == "<recipe_id>" {
				continue // generic executor; recipe_id supplied at call time
			}
			if !reg.Exists(id) {
				t.Errorf("exec tool %s argv recipe %q not in registry", tc.ID, id)
			}
		}
	}
}
