package brain

import (
	"context"

	"cactus-agentlink-rescue/internal/command"
)

func Doctor(ctx context.Context, runner command.Runner, home string) BrainAvailability {
	backend := NewLlamaCLIBackend(runner, home)
	return backend.Available(ctx)
}
