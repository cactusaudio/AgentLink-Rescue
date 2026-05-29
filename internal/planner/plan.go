package planner

import (
	"cactus-agentlink-rescue/internal/recipe"
)

// Inputs is the compact, redacted evidence the planner decides from. It is
// produced by the diagnose -> classify -> diagnosisgraph chain (the kernel
// builds it from diagnosisgraph.Graph), never from raw host dumps.
type Inputs struct {
	PrimaryClass      string
	Classes           []string
	Confidence        float64
	ProtectedTopology bool
}

// RepairConfidenceFloor mirrors the validator: a repair is only ever emitted
// at or above this confidence. Below it the planner stays read-only.
const RepairConfidenceFloor = 0.55

// Plan is the distilled judgment of the kernel: given ranked, classified
// evidence and the recipe catalog, it produces a single bounded Decision.
// It is a pure function (no host access, no I/O) so it is fully test-stable.
//
// The judgment it encodes (architecture review P0.3 + distillation goal):
//   - protected topology is never mutated (read-only report);
//   - select the least-invasive recipe routed to the primary class
//     (recipe.RecipesForClass is already risk-ordered);
//   - only reversible/safe writable recipes auto-route to "repair", and only
//     at/above the confidence floor — they are snapshot+rollback gated by
//     recipe.Run;
//   - read-only / network recipes route to "probe" (no host mutation without
//     explicit execution);
//   - privileged recipes are NOT executed on this path — they belong to the
//     deterministic TUN supervisor / last-resort consent path, so the planner
//     reports and defers;
//   - with no routed recipe, or below the confidence floor, stay read-only.
//
// The returned Decision is intended to be gated by ValidateDecision against
// the same recipe + verifier registries before any execution.
func Plan(in Inputs, reg recipe.Registry) Decision {
	if in.ProtectedTopology {
		return Decision{
			SchemaVersion:      1,
			Intent:             "report",
			FailureClass:       in.PrimaryClass,
			Confidence:         clamp01(in.Confidence),
			ExplanationForUser: "Protected topology detected; planner stays read-only and will not mutate protected interfaces or routes.",
			StopReason:         "protected topology repair corridor",
		}
	}
	if in.PrimaryClass == "" {
		return Decision{
			SchemaVersion:      1,
			Intent:             "report",
			Confidence:         clamp01(in.Confidence),
			ExplanationForUser: "No deterministic failure class; gather more evidence before any repair.",
			StopReason:         "no deterministic failure class",
		}
	}

	candidates := reg.RecipesForClass(in.PrimaryClass)
	if len(candidates) == 0 {
		return Decision{
			SchemaVersion:      1,
			Intent:             "report",
			FailureClass:       in.PrimaryClass,
			Confidence:         clamp01(in.Confidence),
			ExplanationForUser: "Class detected but no bounded recipe is routed to it; staying read-only.",
			StopReason:         "no recipe routed for class " + in.PrimaryClass,
		}
	}

	best := candidates[0]
	fallbacks := recipeIDs(candidates[1:])
	conf := clamp01(in.Confidence)

	switch best.Risk {
	case recipe.RiskSafePatch, recipe.RiskReversiblePatch:
		if conf < RepairConfidenceFloor {
			return probeDecision(in.PrimaryClass, conf, best, "confidence below repair floor; probe before any reversible repair")
		}
		return Decision{
			SchemaVersion:        1,
			Intent:               "repair",
			FailureClass:         in.PrimaryClass,
			Confidence:           conf,
			SelectedRecipe:       SelectedRecipe{ID: best.ID, Params: map[string]string{}},
			Risk:                 best.Risk,
			RequiresUserApproval: recipe.WritableRisk(best.Risk),
			ExpectedVerifiers:    verifierIDs(best),
			FallbackRecipes:      fallbacks,
			ExplanationForUser:   "Least-invasive reversible recipe routed to " + in.PrimaryClass + "; snapshot + verifier + rollback are enforced by the runner.",
		}
	case recipe.RiskReadOnly, recipe.RiskNetworkAction:
		return probeDecision(in.PrimaryClass, conf, best, "least-invasive routed recipe is read-only/network probe; no host mutation without explicit execution")
	default: // privileged_action and anything stricter
		return Decision{
			SchemaVersion:      1,
			Intent:             "report",
			FailureClass:       in.PrimaryClass,
			Confidence:         conf,
			FallbackRecipes:    recipeIDs(candidates),
			ExplanationForUser: "Only privileged repairs are routed to this class; planner defers to the privileged supervisor / user-approved ticket path.",
			StopReason:         "privileged repair requires terminal ticket or last-resort consent",
		}
	}
}

func probeDecision(class string, conf float64, rec recipe.Recipe, reason string) Decision {
	return Decision{
		SchemaVersion:      1,
		Intent:             "probe",
		FailureClass:       class,
		Confidence:         conf,
		SelectedRecipe:     SelectedRecipe{ID: rec.ID, Params: map[string]string{}},
		Risk:               rec.Risk,
		ExpectedVerifiers:  verifierIDs(rec),
		ExplanationForUser: reason,
	}
}

func verifierIDs(r recipe.Recipe) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range r.Verify {
		if v.ID == "" || seen[v.ID] {
			continue
		}
		seen[v.ID] = true
		out = append(out, v.ID)
	}
	return out
}

func recipeIDs(recs []recipe.Recipe) []string {
	if len(recs) == 0 {
		return nil
	}
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.ID)
	}
	return out
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
