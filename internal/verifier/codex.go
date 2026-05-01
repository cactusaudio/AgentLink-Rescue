package verifier

import (
	"context"
	"time"

	"cactus-agentlink-rescue/internal/command"
)

func codexVersionRuns(ctx Context, args map[string]string) Result {
	runner := ctx.Runner
	if runner == nil {
		runner = command.NewExecRunner()
	}
	callCtx, cancel := context.WithTimeout(ctx.Context, 5*time.Second)
	defer cancel()
	res := runner.Run(callCtx, "codex", "--version")
	if res.ExitCode == 0 {
		return pass("codex version runs")
	}
	return warn(commandString(res))
}

func providerSmokeTestRequired(ctx Context, args map[string]string) Result {
	_ = ctx
	_ = args
	return warn("provider_smoke_test_required: static config template is not proof of online provider compatibility")
}
