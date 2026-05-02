package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/verifier"
)

type PlanInput struct {
	Target string
	Home   string
	DryRun bool
	Yes    bool
	Online bool
	Facts  facts.Facts
}

type PlannerCall struct {
	Index      int              `json:"index"`
	Backend    string           `json:"backend"`
	DurationMs int64            `json:"durationMs"`
	RawOutput  string           `json:"rawOutput,omitempty"`
	Decision   planner.Decision `json:"decision,omitempty"`
	Validation string           `json:"validation,omitempty"`
	Correction bool             `json:"correction"`
}

type PlanResult struct {
	Decision         planner.Decision `json:"decision"`
	Calls            []PlannerCall    `json:"calls"`
	ValidationErrors []string         `json:"validationErrors,omitempty"`
	RawOutput        string           `json:"rawOutput,omitempty"`
}

type Planner struct {
	Backend   BrainBackend
	Recipes   recipe.Registry
	Verifiers verifier.Registry
}

func NewPlanner(backend BrainBackend, recipes recipe.Registry) Planner {
	return Planner{Backend: backend, Recipes: recipes, Verifiers: verifier.NewRegistry()}
}

func (p Planner) Plan(ctx context.Context, in PlanInput) (PlanResult, error) {
	if p.Backend == nil {
		return PlanResult{}, fmt.Errorf("brain backend missing")
	}
	if p.Verifiers.IDs() == nil {
		p.Verifiers = verifier.NewRegistry()
	}
	userPrompt := BuildPlannerUserPrompt(PlanPromptInput{
		Target:        in.Target,
		DryRun:        in.DryRun,
		Yes:           in.Yes,
		Online:        in.Online,
		Facts:         in.Facts,
		Recipes:       targetRecipes(p.Recipes, in.Target),
		FailureBudget: 2,
	})
	result, err := p.generateAndValidate(ctx, userPrompt, false, 1)
	if err == nil {
		return result, nil
	}
	correctionPrompt := userPrompt + "\n\nYour previous output was invalid. Return only valid PlannerDecision JSON. Validation error: " + err.Error()
	corrected, err2 := p.generateAndValidate(ctx, correctionPrompt, true, 2)
	corrected.Calls = append(result.Calls, corrected.Calls...)
	corrected.ValidationErrors = append(result.ValidationErrors, err.Error())
	if err2 != nil {
		corrected.ValidationErrors = append(corrected.ValidationErrors, err2.Error())
		return corrected, err2
	}
	return corrected, nil
}

func (p Planner) generateAndValidate(ctx context.Context, userPrompt string, correction bool, index int) (PlanResult, error) {
	resp, err := p.Backend.Generate(ctx, BrainRequest{
		SystemPrompt: PlannerSystemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    DefaultModelManifest().DefaultMaxTokens,
		Temperature:  0,
		ContextSize:  DefaultModelManifest().DefaultContext,
		Timeout:      3 * time.Minute,
		ExpectJSON:   true,
	})
	call := PlannerCall{Index: index, Backend: resp.Backend, DurationMs: resp.DurationMs, RawOutput: safety.RedactSensitive(resp.RawText), Correction: correction}
	out := PlanResult{RawOutput: call.RawOutput, Calls: []PlannerCall{call}}
	if err != nil {
		out.ValidationErrors = append(out.ValidationErrors, err.Error())
		return out, err
	}
	var d planner.Decision
	if err := json.Unmarshal([]byte(resp.ExtractedJSON), &d); err != nil {
		out.ValidationErrors = append(out.ValidationErrors, err.Error())
		return out, err
	}
	call.Decision = d
	if err := planner.ValidateDecision(d, p.Recipes, p.Verifiers); err != nil {
		call.Validation = err.Error()
		out.Calls[0] = call
		out.ValidationErrors = append(out.ValidationErrors, err.Error())
		return out, err
	}
	call.Validation = "ok"
	out.Calls[0] = call
	out.Decision = d
	return out, nil
}
