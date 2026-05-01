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
	if d.SchemaVersion <= 0 {
		return fmt.Errorf("schemaVersion required")
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("confidence must be 0..1")
	}
	if d.Intent != "repair" && d.Intent != "probe" && d.Intent != "report" {
		return fmt.Errorf("invalid intent")
	}
	if d.Intent == "repair" && d.Confidence < 0.55 {
		return fmt.Errorf("repair confidence below 0.55")
	}
	rec, ok := recipes.Get(d.SelectedRecipe.ID)
	if !ok {
		return fmt.Errorf("selected recipe not found")
	}
	if _, err := recipe.ResolveParams(rec, d.SelectedRecipe.Params); err != nil {
		return err
	}
	if recipe.RiskRank(d.Risk) > recipe.RiskRank(rec.Risk) {
		return fmt.Errorf("decision risk exceeds recipe risk")
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
	if recipe.WritableRisk(rec.Risk) && len(rec.Patches) > 0 && len(rec.Rollback) == 0 {
		return fmt.Errorf("writable recipe lacks rollback")
	}
	if containsSecretShape(d.ExplanationForUser) {
		return fmt.Errorf("decision contains unredacted secret-like text")
	}
	return nil
}

func containsSecretShape(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "authorization: bearer ") || strings.Contains(lower, "sk-") || strings.Contains(lower, "api_key=")
}
