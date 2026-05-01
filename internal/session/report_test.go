package session

import (
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/snapshot"
)

func TestAgentDispatchRedactsSecret(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-secret-value")
	s := Session{
		ToolVersion: "test",
		SessionID:   "s",
		FinalState:  "fail",
		CommandsRun: []snapshot.ActionLog{{
			Command:        `/usr/bin/git config --global http.proxy http://user:secret@127.0.0.1:7897`,
			RedactedStdout: "token=sk-test-secret-value",
		}},
	}
	out := AgentDispatch(s)
	if strings.Contains(out, "sk-test-secret-value") {
		t.Fatal("agent dispatch leaked secret")
	}
	if strings.Contains(out, "user:secret@") {
		t.Fatal("agent dispatch leaked proxy credentials")
	}
}
