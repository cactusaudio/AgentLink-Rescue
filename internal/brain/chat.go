package brain

import (
	"context"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

const ChatSandboxSystemPrompt = `You are Cactus Brain Sandbox, a local offline assistant inside AgentLink Rescue.
Cactus AgentLink Rescue is an offline macOS rescue tool that diagnoses and repairs the local path between a Mac and AI agents through deterministic recipes, snapshots, verifiers, and rollback.
You cannot execute commands.
You cannot read files unless the user explicitly pasted text.
You cannot change system settings.
You should explain, summarize, and suggest safe next steps.
Never claim you performed an action.
Never reveal or request secrets.
If asked to repair, suggest using Guided Rescue or Expert Console.`

type ChatReport struct {
	SchemaVersion int      `json:"schemaVersion"`
	ToolVersion   string   `json:"toolVersion"`
	OK            bool     `json:"ok"`
	Prompt        string   `json:"prompt,omitempty"`
	Response      string   `json:"response,omitempty"`
	Backend       string   `json:"backend,omitempty"`
	ModelID       string   `json:"modelID,omitempty"`
	ModelPath     string   `json:"modelPath,omitempty"`
	DurationMs    int64    `json:"durationMs,omitempty"`
	Error         string   `json:"error,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

func Chat(ctx context.Context, runner command.Runner, home, prompt string) ChatReport {
	out := ChatReport{
		SchemaVersion: 1,
		ToolVersion:   system.Version,
		Prompt:        safety.RedactSensitive(prompt),
		ModelID:       DefaultModelID,
	}
	if strings.TrimSpace(prompt) == "" {
		out.Error = "empty prompt"
		return out
	}
	backend := NewLlamaCLIBackend(runner, home)
	doc := backend.Available(ctx)
	out.Backend = doc.Backend
	out.ModelPath = doc.ModelPath
	if !doc.BrainPackAvailable {
		out.Error = "brain assets missing"
		out.Warnings = append(out.Warnings, doc.MissingAssets...)
		out.Warnings = append(out.Warnings, doc.FetchCommands...)
		return out
	}
	req := BrainRequest{
		SystemPrompt: ChatSandboxSystemPrompt,
		UserPrompt:   safety.RedactSensitive(prompt),
		MaxTokens:    384,
		Temperature:  0,
		ContextSize:  4096,
		Timeout:      3 * time.Minute,
		ExpectJSON:   false,
	}
	resp, err := backend.Generate(ctx, req)
	out.DurationMs = resp.DurationMs
	if err != nil {
		out.Error = safety.RedactSensitive(err.Error())
		return out
	}
	out.OK = true
	out.Response = safety.RedactSensitive(strings.TrimSpace(llamaAssistantOutput(resp.RawText)))
	if out.Response == "" {
		out.Response = safety.RedactSensitive(strings.TrimSpace(resp.RawText))
	}
	return out
}
