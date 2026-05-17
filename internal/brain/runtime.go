package brain

import (
	"context"
	"time"
)

type BrainBackend interface {
	Name() string
	Available(ctx context.Context) BrainAvailability
	Generate(ctx context.Context, req BrainRequest) (BrainResponse, error)
}

type BrainAvailability struct {
	Backend            string   `json:"backend"`
	BrainPackAvailable bool     `json:"brainPackAvailable"`
	PackageRoot        string   `json:"packageRoot,omitempty"`
	ModelFamily        string   `json:"modelFamily,omitempty"`
	ModelID            string   `json:"modelID,omitempty"`
	ModelName          string   `json:"modelName,omitempty"`
	ModelPath          string   `json:"modelPath,omitempty"`
	ModelExists        bool     `json:"modelExists"`
	ModelSHA256OK      bool     `json:"modelSha256OK"`
	ModelSizeBytes     int64    `json:"modelSizeBytes,omitempty"`
	RuntimePath        string   `json:"runtimePath,omitempty"`
	ServerPath         string   `json:"serverPath,omitempty"`
	RuntimeExists      bool     `json:"runtimeExists"`
	RuntimeExecutable  bool     `json:"runtimeExecutable"`
	ServerExists       bool     `json:"serverExists"`
	ServerExecutable   bool     `json:"serverExecutable"`
	RuntimeArch        string   `json:"runtimeArch,omitempty"`
	EstimatedModelSize int64    `json:"estimatedModelSize"`
	PackageLocalAssets bool     `json:"packageLocalAssets"`
	UserCacheAssets    bool     `json:"userCacheAssets"`
	MissingAssets      []string `json:"missingAssets,omitempty"`
	FetchCommands      []string `json:"fetchCommands,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

type BrainRequest struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
	ContextSize  int
	Timeout      time.Duration
	ExpectJSON   bool
}

type BrainResponse struct {
	RawText string `json:"rawText"`
	// Answer is the model generation ONLY (llama-cli banner/preamble +
	// trailing perf/exit chrome stripped via the same llamaAssistantOutput
	// extractor `brain chat` uses, then redacted). Answer-isolated
	// consumers MUST use this, not RawText (RawText carries the banner —
	// the v0.3.0 transcript-level caveat root cause).
	Answer        string `json:"answer"`
	ExtractedJSON string `json:"extractedJSON,omitempty"`
	DurationMs    int64  `json:"durationMs"`
	Backend       string `json:"backend"`
	ModelPath     string `json:"modelPath"`
	TokensApprox  int    `json:"tokensApprox"`
}
