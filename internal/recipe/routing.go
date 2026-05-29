package recipe

import (
	"runtime"
	"sort"
)

// RecipesForClass is the reverse index of Recipe.FailureClasses: given a
// deterministic failure class emitted by classify/diagnosisgraph, it returns
// the OS-supported, non-destructive recipes that declare a repair path for
// that class.
//
// Ordering is least-invasive-first: ascending risk rank (read_only <
// safe_patch < reversible_patch < network_action < privileged_action), then
// stable by recipe ID. This is the kernel's class -> recipe routing surface
// (architecture review P0.3): the planner consumes it instead of the
// orchestrator string-equalling a single hardcoded recipe.
func (r Registry) RecipesForClass(class string) []Recipe {
	if class == "" {
		return nil
	}
	var out []Recipe
	for _, rec := range r.recipes {
		if rec.Risk == RiskDestructiveAction {
			continue
		}
		if !SupportsOS(rec, runtime.GOOS) {
			continue
		}
		for _, fc := range rec.FailureClasses {
			if fc == class {
				out = append(out, rec)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if RiskRank(out[i].Risk) != RiskRank(out[j].Risk) {
			return RiskRank(out[i].Risk) < RiskRank(out[j].Risk)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// LowestRiskRecipeForClass returns the single least-invasive recipe routed to
// the class, or ok=false if none is declared.
func (r Registry) LowestRiskRecipeForClass(class string) (Recipe, bool) {
	candidates := r.RecipesForClass(class)
	if len(candidates) == 0 {
		return Recipe{}, false
	}
	return candidates[0], true
}
