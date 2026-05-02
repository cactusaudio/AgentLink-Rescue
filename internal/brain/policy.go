package brain

import (
	"fmt"

	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
)

type AutoPolicy struct {
	DryRun bool
	Yes    bool
	Online bool
}

func ValidateAutoExecution(d planner.Decision, rec recipe.Recipe, policy AutoPolicy) error {
	if d.Intent != "repair" {
		return nil
	}
	switch rec.Risk {
	case recipe.RiskReadOnly, recipe.RiskSafePatch:
		return nil
	case recipe.RiskReversiblePatch:
		if !policy.Yes && !policy.DryRun {
			return fmt.Errorf("reversible_patch requires --yes in brain auto mode")
		}
		return nil
	case recipe.RiskNetworkAction:
		if !policy.Online {
			return fmt.Errorf("network_action requires --online in brain auto mode")
		}
		return nil
	case recipe.RiskPrivilegedAction:
		return fmt.Errorf("privileged_action is refused in v0.3 brain auto mode")
	case recipe.RiskDestructiveAction:
		return fmt.Errorf("destructive_action is refused in v0.3 brain auto mode")
	default:
		return fmt.Errorf("unknown risk: %s", rec.Risk)
	}
}
