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
	// Opt-in override: Gemma 4 E4B is a reasoning model ("[Start
	// thinking]…"); the 1024 default can truncate it mid-reasoning
	// before it emits the answer. Zero default-behavior change — only
	// honored when explicitly set (e.g. the Tier-A recommender harness).
	if v := os.Getenv("AGENTLINK_BRAIN_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 8192 {
			req.MaxTokens = n
		}
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
		RawText: raw,
		// answer-isolated: strip the llama-cli banner/preamble + trailing
		// chrome (raw is already RedactSensitive'd above).
		Answer:       stripLlamaTrailer(stripLlamaBanner(llamaAssistantOutput(raw))),
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

// stripLlamaTrailer removes llama-cli's trailing chrome (perf line,
// Exiting..., llama_perf, [end of text], Gemma <end_of_turn>) so an
// answer-isolated consumer sees ONLY the model generation. Cuts at the
// earliest trailing marker. Used for BrainResponse.Answer.
func stripLlamaTrailer(s string) string {
	for _, m := range []string{
		"<end_of_turn>", "[ Prompt:", "\n[ Prompt", "[end of text]",
		"\nExiting...", "Exiting...", "\nllama_perf", "llama_perf_",
	} {
		if i := strings.Index(s, m); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}

// stripLlamaBanner is the FALLBACK for when the Gemma turn marker is
// absent (large prompts / truncated format) and llamaAssistantOutput
// returned raw-with-banner. It drops leading llama-cli banner/system
// lines until the first line of real model content. Banner lines are
// deterministic chrome; a rescue answer (agentlink.* / 中文 / "recommend")
// never matches these patterns, so real text is preserved.
func stripLlamaBanner(s string) string {
	bannerish := func(ln string) bool {
		t := strings.TrimSpace(ln)
		if t == "" {
			return true
		}
		// pure box-drawing / ascii-art line
		nonArt := strings.TrimLeft(t, "▄▀█ \t")
		if nonArt == "" {
			return true
		}
		for _, p := range []string{
			"Loading model", "build      :", "build :", "model      :",
			"model :", "modalities :", "main:", "system info", "system_info",
			"available commands", "/exit", "/regen", "/help", "/clear",
			"llama_", "ggml_", "register_backend", "load:", "print_info",
			"common_init", "sampler", "generate:", "srv ", "warming up",
			"build:", "n_ctx", "init:", "<start_of_turn>", "<bos>",
		} {
			if strings.HasPrefix(t, p) || strings.Contains(t, p) {
				return true
			}
		}
		return false
	}
	lines := strings.Split(s, "\n")
	i := 0
	for i < len(lines) && bannerish(lines[i]) {
		i++
	}
	return strings.TrimSpace(strings.Join(lines[i:], "\n"))
}

func LlamaCLIArgs(modelPath, prompt string, maxTokens, contextSize int, temperature float64) []string {
	return []string{
		"-m", modelPath,
		"-p", prompt,
		"-n", strconv.Itoa(maxTokens),
		"-c", strconv.Itoa(contextSize),
		"--temp", strconv.FormatFloat(temperature, 'f', -1, 64),
		// -no-cnv: one-shot completion. Without it llama-cli runs in
		// interactive CONVERSATION mode — prints the REPL banner, echoes
		// the prompt, and emits no stable <start_of_turn>model marker, so
		// answer-isolation leaks the banner + echoed prompt (incl. any
		// sentinels) and truncates the model's real answer. This is the
		// true source of the v0.3.x Gemma answer-isolation fragility:
		// fix it at the backend so the Tier-A recommender surface is
		// clean for the product, not just for harnesses.
		"-no-cnv",
		"-st",
		"--no-display-prompt",
		"--simple-io",
		"--no-warmup",
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
