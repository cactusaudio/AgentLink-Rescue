package brain

import (
	"strings"
	"testing"
)

func TestChatSandboxPromptHasNoExecutorClaims(t *testing.T) {
	for _, want := range []string{"cannot execute commands", "cannot change system settings", "Never claim you performed an action"} {
		if !strings.Contains(ChatSandboxSystemPrompt, want) {
			t.Fatalf("sandbox prompt missing %q", want)
		}
	}
}

func TestFormatChatPromptDoesNotForcePlannerDecision(t *testing.T) {
	prompt := FormatChatPrompt(ChatSandboxSystemPrompt, "hello")
	if strings.Contains(prompt, "PlannerDecision") {
		t.Fatalf("chat prompt should not force PlannerDecision: %s", prompt)
	}
	if !strings.Contains(prompt, "cannot execute commands") {
		t.Fatalf("chat prompt missing sandbox policy")
	}
}
