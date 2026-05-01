package repair

import (
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/diagnose"
)

func TestBuildActionsSkipsDisabledServices(t *testing.T) {
	report := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			Services: []diagnose.NetworkService{
				{Name: "Wi-Fi"},
				{Name: "Thunderbolt Bridge", Disabled: true},
			},
		},
	}
	actions := buildActions(LevelSafe, report)
	for _, action := range actions {
		if strings.Contains(action.Description, "Thunderbolt Bridge") {
			t.Fatalf("disabled service produced action: %+v", action)
		}
		for _, arg := range action.Args {
			if arg == "Thunderbolt Bridge" {
				t.Fatalf("disabled service produced command args: %+v", action)
			}
		}
	}
}

func TestFormatRollbackCommandUsesAbsoluteQuotedExecutable(t *testing.T) {
	got := FormatRollbackCommand("/tmp/Cactus AgentLink Rescue/bin/agentlink", "20260501-021413")
	want := `sudo "/tmp/Cactus AgentLink Rescue/bin/agentlink" rollback --id "20260501-021413"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
