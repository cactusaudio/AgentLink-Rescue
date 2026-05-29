package planner

import (
	"cactus-agentlink-rescue/internal/classify"
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
		// No user-config recipe routes this class. If it mutates system network
		// state, route it to the privileged repair tier (reversible via captured
		// network snapshot + rollback); otherwise stay read-only.
		return planNetworkLevel(in)
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
	default: // privileged_action: a registry recipe exists but mutates system
		// network state. Route through the graduated network-level path (which
		// picks the appropriate tier from RecommendedRepairLevel) rather than
		// jumping to a heavy privileged recipe.
		return planNetworkLevel(in)
	}
}

// networkStateClasses are the classes whose remedy is a bounded system
// network-state mutation handled by the privileged repair engine (reversible
// via the captured network snapshot + rollback). The repair TIER for each is
// taken from classify.RecommendedRepairLevel — the single source of truth — so
// this set never drifts from the level policy. Classes deliberately absent are
// report-only by design: MDM profiles (need user/admin), agent-endpoint blocks
// (escalate upstream), AirDrop discovery (cosmetic), and the TUN-tier classes
// (owned by the deterministic supervisor).
var networkStateClasses = map[string]bool{
	classify.SystemProxyDirty:          true,
	classify.DNSFail:                   true,
	classify.NoDHCPLease:               true,
	classify.NoDefaultRoute:            true,
	classify.GatewayUnreachable:        true,
	classify.RawIPUnreachable:          true,
	classify.LinkLocalOnly:             true,
	classify.NoActiveInterface:         true,
	classify.HTTPSFail:                 true,
	classify.SysconfigSuspected:        true,
	classify.NetworkLocationSuspected:  true,
	classify.KnownAgentResidue:         true,
	classify.NetworkExtensionSuspected: true,
}

// planNetworkLevel routes a class that has no user-config recipe. If the class
// is a known network-state class it goes to the privileged repair tier (tier
// from classify.RecommendedRepairLevel); everything else stays read-only.
func planNetworkLevel(in Inputs) Decision {
	conf := clamp01(in.Confidence)
	if isProtectedClass(in.PrimaryClass) {
		return reportDecision(in.PrimaryClass, conf, "protected topology class; read-only, no network mutation")
	}
	if !networkStateClasses[in.PrimaryClass] {
		return reportDecision(in.PrimaryClass, conf, "no bounded repair routed for class "+in.PrimaryClass)
	}
	level := classify.RecommendedRepairLevel([]string{in.PrimaryClass})
	if level == LevelTun {
		return reportDecision(in.PrimaryClass, conf, "TUN-tier class deferred to the deterministic supervisor")
	}
	if !validRepairLevel(level) { // "none" or unknown
		return reportDecision(in.PrimaryClass, conf, "no bounded repair routed for class "+in.PrimaryClass)
	}
	if conf < RepairConfidenceFloor {
		return reportDecision(in.PrimaryClass, conf, "confidence below repair floor for privileged network repair")
	}
	risk := recipe.RiskNetworkAction
	if level == LevelStandard || level == LevelDeep {
		risk = recipe.RiskPrivilegedAction
	}
	return Decision{
		SchemaVersion:        1,
		Intent:               "repair",
		FailureClass:         in.PrimaryClass,
		Confidence:           conf,
		RepairLevel:          level,
		Risk:                 risk,
		RequiresUserApproval: true,
		ExplanationForUser:   "System network-state repair (" + level + " tier) routed to " + in.PrimaryClass + "; privileged and reversible via the captured network snapshot + rollback.",
	}
}

func reportDecision(class string, conf float64, reason string) Decision {
	return Decision{
		SchemaVersion:      1,
		Intent:             "report",
		FailureClass:       class,
		Confidence:         conf,
		ExplanationForUser: "Read-only: " + reason + ".",
		StopReason:         reason,
	}
}

func isProtectedClass(class string) bool {
	return class == classify.ProtectedAudioVLANRouteTrap || class == classify.ProtectedTopologyConstraint
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
