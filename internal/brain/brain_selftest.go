package brain

import (
	"context"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
)

type SelftestResult struct {
	OK       bool              `json:"ok"`
	Doctor   BrainAvailability `json:"doctor"`
	Decision planner.Decision  `json:"decision,omitempty"`
	Error    string            `json:"error,omitempty"`
}

func Selftest(ctx context.Context, runner command.Runner, home string, reg recipe.Registry) SelftestResult {
	backend := NewLlamaCLIBackend(runner, home)
	doc := backend.Available(ctx)
	res := SelftestResult{Doctor: doc}
	if !doc.BrainPackAvailable {
		res.Error = "brain assets missing"
		return res
	}
	f := facts.Facts{
		SchemaVersion: 1,
		OS:            "darwin",
		Arch:          "arm64",
		Home:          home,
		APIKeys:       map[string]facts.KeyFact{},
	}
	p := NewPlanner(backend, reg)
	plan, err := p.Plan(ctx, PlanInput{Target: "keys", Home: home, Facts: f})
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Decision = plan.Decision
	res.OK = true
	return res
}
