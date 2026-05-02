package planner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/verifier"
)

func LoadDecision(path string) (Decision, error) {
	var d Decision
	data, err := os.ReadFile(path)
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return d, err
	}
	return d, nil
}

func ValidateDecision(d Decision, recipes recipe.Registry, verifiers verifier.Registry) error {
	if d.SchemaVersion != 1 {
		return fmt.Errorf("schemaVersion must be 1")
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("confidence must be 0..1")
	}
	if d.Intent != "repair" && d.Intent != "probe" && d.Intent != "report" && d.Intent != "rollback" {
		return fmt.Errorf("invalid intent")
	}
	if d.Risk != "" && d.Risk != recipe.RiskReadOnly && d.Risk != recipe.RiskSafePatch && d.Risk != recipe.RiskReversiblePatch && d.Risk != recipe.RiskNetworkAction && d.Risk != recipe.RiskPrivilegedAction && d.Risk != recipe.RiskDestructiveAction {
		return fmt.Errorf("invalid risk")
	}
	if d.Intent == "repair" && d.Confidence < 0.55 {
		return fmt.Errorf("repair confidence below 0.55")
	}
	if d.Intent == "repair" && d.Risk == "" {
		return fmt.Errorf("repair risk required")
	}
	var rec recipe.Recipe
	if d.Intent == "repair" || d.SelectedRecipe.ID != "" {
		var ok bool
		rec, ok = recipes.Get(d.SelectedRecipe.ID)
		if !ok {
			return fmt.Errorf("selected recipe not found")
		}
		if _, err := recipe.ResolveParams(rec, d.SelectedRecipe.Params); err != nil {
			return err
		}
		if recipe.RiskRank(d.Risk) > recipe.RiskRank(rec.Risk) {
			return fmt.Errorf("decision risk exceeds recipe risk")
		}
		if recipe.WritableRisk(rec.Risk) && len(rec.Patches) > 0 && len(rec.Rollback) == 0 {
			return fmt.Errorf("writable recipe lacks rollback")
		}
	} else if d.Risk != "" && recipe.RiskRank(d.Risk) > recipe.RiskRank(recipe.RiskNetworkAction) {
		return fmt.Errorf("invalid report/probe risk")
	}
	for _, id := range d.ExpectedVerifiers {
		if !verifiers.Exists(id) {
			return fmt.Errorf("unknown expected verifier: %s", id)
		}
	}
	for _, id := range d.FallbackRecipes {
		if !recipes.Exists(id) {
			return fmt.Errorf("unknown fallback recipe: %s", id)
		}
	}
	if containsSecretShape(d.ExplanationForUser) {
		return fmt.Errorf("decision contains unredacted secret-like text")
	}
	for _, e := range d.Evidence {
		if containsSecretShape(e) {
			return fmt.Errorf("decision evidence contains unredacted secret-like text")
		}
	}
	if containsSecretShape(d.StopReason) {
		return fmt.Errorf("decision stopReason contains unredacted secret-like text")
	}
	return nil
}

func containsSecretShape(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "authorization: bearer ") || strings.Contains(lower, "sk-") || strings.Contains(lower, "api_key=")
}
