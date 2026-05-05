package restartgate

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestPrepareWithoutTunRepairAttemptDoesNotRequireRestart(t *testing.T) {
	rep := Prepare(context.Background(), &command.MockRunner{}, "", Context{})
	if rep.RestartRequired {
		t.Fatalf("restart gate required without targeted TUN repair context: %+v", rep)
	}
	if rep.NextAction == "" {
		t.Fatalf("expected next action when restart gate is refused: %+v", rep)
	}
}

func TestPrepareAfterTunRepairMayRequireRestart(t *testing.T) {
	rep := Prepare(context.Background(), &command.MockRunner{}, "", Context{TunRepairAttempted: true, AfterRestorePoint: "rp"})
	if !rep.RestartRequired {
		t.Fatalf("expected restart gate to be allowed after targeted TUN repair context on failing verify: %+v", rep)
	}
	if rep.PostRestartCommand == "" {
		t.Fatalf("missing post-restart verify command: %+v", rep)
	}
}
