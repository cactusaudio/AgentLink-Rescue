package brain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type LlamaCLIBackend struct {
	Runner   command.Runner
	Home     string
	Manifest ModelManifest
}

func NewLlamaCLIBackend(runner command.Runner, home string) LlamaCLIBackend {
	return LlamaCLIBackend{Runner: runner, Home: home, Manifest: DefaultModelManifest()}
}

func (b LlamaCLIBackend) Name() string {
	return "llama-cli"
}

func (b LlamaCLIBackend) Available(ctx context.Context) BrainAvailability {
	_ = ctx
	manifest := b.Manifest
	if manifest.ID == "" {
		manifest = DefaultModelManifest()
	}
	manifest = ResolveModelManifest(b.Home, manifest)
	loc := LocateAssets(b.Home, manifest)
	out := BrainAvailability{
		Backend:            b.Name(),
		PackageRoot:        loc.PackageRoot,
		ModelFamily:        manifest.Family,
		ModelID:            manifest.ID,
		ModelName:          manifest.Model,
		ModelPath:          loc.ModelPath,
		ModelExists:        loc.ModelExists,
		ModelSHA256OK:      loc.ModelSHA256OK,
		ModelSizeBytes:     fileSize(loc.ModelPath),
		RuntimePath:        loc.RuntimePath,
		ServerPath:         loc.ServerPath,
		RuntimeExists:      loc.RuntimeExists,
		RuntimeExecutable:  loc.RuntimeExecutable,
		ServerExists:       loc.ServerExists,
		ServerExecutable:   loc.ServerExecutable,
		RuntimeArch:        loc.RuntimeArch,
		EstimatedModelSize: manifest.SizeBytesApprox,
		PackageLocalAssets: loc.PackageLocalAssets,
		UserCacheAssets:    loc.UserCacheAssets,
	}
	if profile := os.Getenv("AGENTLINK_BRAIN_PROFILE"); profile != "" && profile != DefaultModelID {
		out.Warnings = append(out.Warnings, "unsupported Brain profile "+profile+"; Gemma 4 E4B is the active Brain model")
	}
	if !out.ModelExists {
		out.MissingAssets = append(out.MissingAssets, "gemma model")
	}
	if out.ModelExists && manifest.SHA256 == "" {
		out.Warnings = append(out.Warnings, "model checksum unavailable; run scripts/fetch_brain_assets.sh to refresh manifest.lock.json")
	} else if out.ModelExists && !out.ModelSHA256OK {
		out.Warnings = append(out.Warnings, "model checksum mismatch")
	}
	if !out.RuntimeExecutable {
		out.MissingAssets = append(out.MissingAssets, "llama-cli runtime")
	}
	out.BrainPackAvailable = out.ModelExists && out.ModelSHA256OK && out.RuntimeExecutable
	if !out.BrainPackAvailable {
		out.FetchCommands = []string{"./bin/agentlink brain fetch", "./scripts/fetch_brain_assets.sh", "download Cactus-AgentLink-Rescue-v0.5.0-brain-gemma4-e4b-q4km.zip for offline Brain use"}
	}
	return out
}

func (b LlamaCLIBackend) Generate(ctx context.Context, req BrainRequest) (BrainResponse, error) {
	manifest := b.Manifest
	if manifest.ID == "" {
		manifest = DefaultModelManifest()
	}
	manifest = ResolveModelManifest(b.Home, manifest)
	avail := b.Available(ctx)
	if !avail.ModelExists {
		return BrainResponse{}, fmt.Errorf("brain model missing")
	}
	if !avail.ModelSHA256OK {
		return BrainResponse{}, fmt.Errorf("brain model checksum mismatch")
	}
	if !avail.RuntimeExecutable {
		return BrainResponse{}, fmt.Errorf("llama-cli runtime missing or not executable")
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = manifest.DefaultMaxTokens
	}
	if req.ContextSize <= 0 {
		req.ContextSize = manifest.DefaultContext
	}
	if req.Timeout <= 0 {
		req.Timeout = 2 * time.Minute
	}
	prompt := FormatChatPrompt(req.SystemPrompt, req.UserPrompt)
	if req.ExpectJSON {
		prompt = FormatPlannerPrompt(req.SystemPrompt, req.UserPrompt)
	}
	bin, args := b.commandForGeneration(avail.RuntimePath, avail.ModelPath, prompt, req.MaxTokens, req.ContextSize, req.Temperature)
	runner := b.Runner
	if runner == nil {
		runner = command.NewExecRunner()
	}
	callCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()
	start := time.Now()
	res := runner.Run(callCtx, bin, args...)
	raw := safety.RedactSensitive(res.Stdout)
	if res.Stderr != "" && (res.ExitCode != 0 || res.TimedOut || res.Missing) {
		raw += "\n" + safety.RedactSensitive(res.Stderr)
	}
	out := BrainResponse{
		RawText:      raw,
		DurationMs:   time.Since(start).Milliseconds(),
		Backend:      b.Name(),
		ModelPath:    avail.ModelPath,
		TokensApprox: len(raw) / 4,
	}
	if req.ExpectJSON {
		extracted, err := ExtractPlannerJSONObject(llamaAssistantOutput(raw))
		if err != nil {
			return out, err
		}
		out.ExtractedJSON = extracted
	}
	if res.ExitCode != 0 || res.TimedOut || res.Missing {
		return out, fmt.Errorf("llama-cli failed: %s", safety.RedactSensitive(res.Error+res.Stderr))
	}
	return out, nil
}

func llamaAssistantOutput(raw string) string {
	gemmaMarker := "<start_of_turn>model"
	if idx := strings.LastIndex(raw, gemmaMarker); idx >= 0 {
		return raw[idx+len(gemmaMarker):]
	}
	return raw
}

func LlamaCLIArgs(modelPath, prompt string, maxTokens, contextSize int, temperature float64) []string {
	return []string{
		"-m", modelPath,
		"-p", prompt,
		"-n", strconv.Itoa(maxTokens),
		"-c", strconv.Itoa(contextSize),
		"--temp", strconv.FormatFloat(temperature, 'f', -1, 64),
		"-st",
		"--no-display-prompt",
		"--simple-io",
	}
}

func LlamaCompletionArgs(modelPath, prompt string, maxTokens, contextSize int, temperature float64) []string {
	return []string{
		"-m", modelPath,
		"-p", prompt,
		"-n", strconv.Itoa(maxTokens),
		"-c", strconv.Itoa(contextSize),
		"--temp", strconv.FormatFloat(temperature, 'f', -1, 64),
		"-no-cnv",
		"--no-display-prompt",
		"--simple-io",
		"--no-warmup",
	}
}

func (b LlamaCLIBackend) commandForGeneration(runtimePath, modelPath, prompt string, maxTokens, contextSize int, temperature float64) (string, []string) {
	completion := filepath.Join(filepath.Dir(runtimePath), "llama-completion")
	if system.CommandExists(completion) {
		return completion, LlamaCompletionArgs(modelPath, prompt, maxTokens, contextSize, temperature)
	}
	return runtimePath, LlamaCLIArgs(modelPath, prompt, maxTokens, contextSize, temperature)
}

func fileSize(path string) int64 {
	if path == "" {
		return 0
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}
