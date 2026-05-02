package guided

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/session"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

const (
	StatusHealthy              = "healthy"
	StatusDryRunComplete       = "dry_run_complete"
	StatusRepaired             = "repaired"
	StatusNoSafeAction         = "no_safe_action"
	StatusManualActionRequired = "manual_action_required"
	StatusFailed               = "failed"
	StatusRolledBack           = "rolled_back"
)

type Options struct {
	Home           string
	Target         string
	DryRun         bool
	Yes            bool
	MaxCycles      int
	TimeoutSeconds int
	CommandLine    []string
}

type Report struct {
	SchemaVersion     int      `json:"schemaVersion"`
	ToolVersion       string   `json:"toolVersion"`
	StartedAt         string   `json:"startedAt"`
	EndedAt           string   `json:"endedAt"`
	Target            string   `json:"target"`
	Mode              string   `json:"mode"`
	Status            string   `json:"status"`
	Cycles            []Cycle  `json:"cycles"`
	FinalSummary      string   `json:"finalSummary"`
	SelectedRecipe    string   `json:"selectedRecipe,omitempty"`
	PlannerUsed       bool     `json:"plannerUsed"`
	BrainModel        string   `json:"brainModel,omitempty"`
	SnapshotID        string   `json:"snapshotID,omitempty"`
	RollbackAvailable bool     `json:"rollbackAvailable"`
	RollbackCommand   string   `json:"rollbackCommand,omitempty"`
	HumanReportPath   string   `json:"humanReportPath,omitempty"`
	CodexDispatchPath string   `json:"codexDispatchPath,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	NextSafeCommand   string   `json:"nextSafeCommand,omitempty"`
}

type Cycle struct {
	Index            int               `json:"index"`
	StateTransitions []string          `json:"stateTransitions"`
	FactsSummary     map[string]any    `json:"factsSummary,omitempty"`
	FailureClasses   []string          `json:"failureClasses,omitempty"`
	CandidateRecipes []string          `json:"candidateRecipes,omitempty"`
	PlannerDecision  any               `json:"plannerDecision,omitempty"`
	PlanValidation   map[string]any    `json:"planValidation,omitempty"`
	DryRun           *recipe.RunResult `json:"dryRun,omitempty"`
	Execution        *recipe.RunResult `json:"execution,omitempty"`
	Verifiers        []any             `json:"verifiers,omitempty"`
	Result           string            `json:"result"`
}

type backendFactory func(command.Runner, string) brain.BrainBackend

func Run(ctx context.Context, runner command.Runner, reg recipe.Registry, opts Options) Report {
	return run(ctx, runner, reg, opts, func(r command.Runner, home string) brain.BrainBackend {
		return brain.NewLlamaCLIBackend(r, home)
	}, nil)
}

func RunForTest(ctx context.Context, runner command.Runner, reg recipe.Registry, opts Options, f facts.Facts, backend brain.BrainBackend) Report {
	return run(ctx, runner, reg, opts, func(command.Runner, string) brain.BrainBackend { return backend }, &f)
}

func run(ctx context.Context, runner command.Runner, reg recipe.Registry, opts Options, backendFor backendFactory, factsOverride *facts.Facts) Report {
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	home := opts.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	target := opts.Target
	if target == "" {
		target = "auto"
	}
	maxCycles := opts.MaxCycles
	if maxCycles <= 0 {
		maxCycles = 3
	}
	timeout := opts.TimeoutSeconds
	if timeout <= 0 {
		timeout = 600
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()
	rep := Report{
		SchemaVersion: 1,
		ToolVersion:   system.Version,
		StartedAt:     start.Format(time.RFC3339),
		Target:        target,
		Mode:          mode(opts),
		Status:        StatusFailed,
		BrainModel:    brain.DefaultModelID,
		Cycles:        []Cycle{},
		Warnings:      []string{},
	}
	sess := session.New(system.Version, opts.CommandLine)
	sess.SelectedTarget = target
	sess.DryRun = opts.DryRun || !opts.Yes
	sess.BrainModelID = brain.DefaultModelID
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "Start")

	addCycle := func(c Cycle) {
		rep.Cycles = append(rep.Cycles, c)
		sess.LoopStateTransitions = append(sess.LoopStateTransitions, c.StateTransitions...)
	}

	f := facts.Facts{}
	if factsOverride != nil {
		f = *factsOverride
	} else {
		f = facts.Collect(ctx, runner, home, true)
	}
	sess.InitialFacts = f
	sess.RealUser = f.RealUser
	sess.HostSummary = f.OS + "/" + f.Arch
	classes := failureClasses(f)
	cycle := Cycle{
		Index:            1,
		StateTransitions: []string{"CollectFacts", "Classify"},
		FactsSummary:     factsSummary(f),
		FailureClasses:   classes,
		Result:           "facts_collected",
	}

	if target == "auto" && healthy(f) {
		cycle.Result = StatusHealthy
		addCycle(cycle)
		rep.Status = StatusHealthy
		rep.FinalSummary = "AgentLink appears healthy. No repair was run."
		rep.NextSafeCommand = "agentlink guided rescue --target auto --dry-run --json"
		return finish(rep, sess, home)
	}
	if target == "network" {
		cycle.StateTransitions = append(cycle.StateTransitions, "FinalReport")
		cycle.Result = StatusManualActionRequired
		addCycle(cycle)
		rep.Status = StatusManualActionRequired
		rep.FinalSummary = "Guided rescue does not auto-run network safe/standard/deep. Use the copyable Terminal command if network repair is needed."
		rep.NextSafeCommand = "sudo ./bin/agentlink rescue --level safe"
		rep.Warnings = append(rep.Warnings, "network rescue remains explicit CLI/Terminal only")
		return finish(rep, sess, home)
	}

	attempted := map[string]bool{}
	verifierFailures := 0
	for i := 1; i <= maxCycles; i++ {
		c := Cycle{
			Index:            i,
			StateTransitions: []string{"DeterministicCandidateSelection"},
			FactsSummary:     factsSummary(f),
			FailureClasses:   classes,
		}
		candidates := candidateRecipes(target, f)
		c.CandidateRecipes = candidates
		for _, id := range candidates {
			if attempted[id] {
				continue
			}
			attempted[id] = true
			if !safeRecipe(reg, id) {
				rep.Warnings = append(rep.Warnings, "refused unsafe recipe: "+id)
				continue
			}
			out := tryRecipe(ctx, runner, reg, home, id, opts, nil)
			c.StateTransitions = append(c.StateTransitions, "DryRun")
			c.DryRun = &out.DryRun
			rep.SelectedRecipe = id
			sess.SelectedRecipe = id
			if rec, ok := reg.Get(id); ok {
				sess.RiskLevel = rec.Risk
			}
			if out.DryRun.Error != "" {
				c.Result = "dry_run_failed"
				rep.Warnings = append(rep.Warnings, out.DryRun.Error)
				continue
			}
			if opts.DryRun || !opts.Yes {
				c.Result = StatusDryRunComplete
				addCycle(c)
				rep.Status = StatusDryRunComplete
				rep.FinalSummary = "Dry-run complete. No files were changed."
				rep.NextSafeCommand = "agentlink guided rescue --target " + target + " --yes --json"
				return finish(rep, sess, home)
			}
			c.StateTransitions = append(c.StateTransitions, "Snapshot", "ExecuteReversibleRecipe", "Verify")
			c.Execution = &out.Execution
			sess.SnapshotID = out.Execution.SnapshotID
			sess.SnapshotPath = out.Execution.SnapshotPath
			sess.ChangedFiles = out.Execution.ChangedFiles
			sess.RollbackAvailable = out.Execution.SnapshotID != ""
			sess.VerifierResults = out.Execution.VerifierResults
			rep.SnapshotID = out.Execution.SnapshotID
			rep.RollbackAvailable = out.Execution.SnapshotID != ""
			if out.Execution.SnapshotID != "" {
				rep.RollbackCommand = "agentlink restore last"
			}
			if out.Execution.Status == recipe.StatusVerifierFailed {
				verifierFailures++
				c.Result = recipe.StatusVerifierFailed
				if verifierFailures >= 2 || out.Execution.SnapshotID != "" {
					rollbackStatus := rollbackLatest(ctx, runner, home, out.Execution.SnapshotID)
					c.StateTransitions = append(c.StateTransitions, "RetryOrRollback")
					rep.Status = rollbackStatus
					rep.FinalSummary = "Verification failed. AgentLink rolled back the latest snapshot where available."
					addCycle(c)
					return finish(rep, sess, home)
				}
				continue
			}
			if out.Execution.Error != "" || out.Execution.Status == recipe.StatusFail {
				c.Result = StatusFailed
				rep.Warnings = append(rep.Warnings, out.Execution.Error)
				continue
			}
			c.Result = StatusRepaired
			addCycle(c)
			rep.Status = StatusRepaired
			rep.FinalSummary = "A reversible recipe completed successfully."
			return finish(rep, sess, home)
		}
		c.Result = "no_deterministic_candidate"
		addCycle(c)
		break
	}

	backend := backendFor(runner, home)
	avail := backend.Available(ctx)
	if !avail.BrainPackAvailable {
		rep.Status = StatusNoSafeAction
		rep.FinalSummary = "No deterministic safe action was available and Gemma Brain assets are missing."
		rep.Warnings = append(rep.Warnings, "Gemma Brain unavailable")
		return finish(rep, sess, home)
	}
	rep.PlannerUsed = true
	sess.BrainEnabled = true
	sess.BrainBackend = backend.Name()
	pl := brain.NewPlanner(backend, reg)
	plan, err := pl.Plan(ctx, brain.PlanInput{Target: nonAutoTarget(target, f), Home: home, DryRun: opts.DryRun || !opts.Yes, Yes: opts.Yes, Facts: f})
	plannerCycle := Cycle{
		Index:            len(rep.Cycles) + 1,
		StateTransitions: []string{"BrainPlanIfNeeded", "ValidatePlan"},
		FactsSummary:     factsSummary(f),
		FailureClasses:   classes,
		PlannerDecision:  plan.Decision,
		PlanValidation: map[string]any{
			"errors": plan.ValidationErrors,
			"ok":     err == nil,
		},
	}
	sess.PlannerCalls = toSessionPlannerCalls(plan.Calls)
	sess.PlannerDecision = plan.Decision
	sess.PlannerIntent = plan.Decision.Intent
	sess.PlannerConfidence = plan.Decision.Confidence
	sess.ValidationErrors = append(sess.ValidationErrors, plan.ValidationErrors...)
	if err != nil {
		plannerCycle.Result = "planner_invalid"
		addCycle(plannerCycle)
		rep.Status = StatusManualActionRequired
		rep.FinalSummary = "Gemma planner did not produce a valid safe repair decision."
		rep.Warnings = append(rep.Warnings, err.Error())
		return finish(rep, sess, home)
	}
	id := plan.Decision.SelectedRecipe.ID
	rec, ok := reg.Get(id)
	if !ok {
		plannerCycle.Result = "unknown_recipe"
		addCycle(plannerCycle)
		rep.Status = StatusManualActionRequired
		rep.FinalSummary = "Gemma selected an unknown recipe; decision refused."
		rep.Warnings = append(rep.Warnings, "unknown recipe: "+id)
		return finish(rep, sess, home)
	}
	if err := brain.ValidateAutoExecution(plan.Decision, rec, brain.AutoPolicy{DryRun: opts.DryRun || !opts.Yes, Yes: opts.Yes}); err != nil {
		plannerCycle.Result = "policy_refused"
		addCycle(plannerCycle)
		rep.Status = StatusManualActionRequired
		rep.FinalSummary = "Gemma selected a recipe that is not allowed in guided mode."
		rep.Warnings = append(rep.Warnings, err.Error())
		return finish(rep, sess, home)
	}
	out := tryRecipe(ctx, runner, reg, home, id, opts, plan.Decision.SelectedRecipe.Params)
	plannerCycle.CandidateRecipes = []string{id}
	plannerCycle.DryRun = &out.DryRun
	rep.SelectedRecipe = id
	sess.SelectedRecipe = id
	sess.RiskLevel = rec.Risk
	if out.DryRun.Error != "" {
		plannerCycle.StateTransitions = append(plannerCycle.StateTransitions, "DryRun")
		plannerCycle.Result = "dry_run_failed"
		addCycle(plannerCycle)
		rep.Status = StatusFailed
		rep.FinalSummary = out.DryRun.Error
		return finish(rep, sess, home)
	}
	if opts.DryRun || !opts.Yes {
		plannerCycle.StateTransitions = append(plannerCycle.StateTransitions, "DryRun", "FinalReport")
		plannerCycle.Result = StatusDryRunComplete
		addCycle(plannerCycle)
		rep.Status = StatusDryRunComplete
		rep.FinalSummary = "Gemma-guided dry-run complete. No files were changed."
		rep.NextSafeCommand = "agentlink guided rescue --target " + target + " --yes --json"
		return finish(rep, sess, home)
	}
	plannerCycle.StateTransitions = append(plannerCycle.StateTransitions, "DryRun", "Snapshot", "ExecuteReversibleRecipe", "Verify")
	plannerCycle.Execution = &out.Execution
	sess.SnapshotID = out.Execution.SnapshotID
	sess.SnapshotPath = out.Execution.SnapshotPath
	sess.ChangedFiles = out.Execution.ChangedFiles
	sess.RollbackAvailable = out.Execution.SnapshotID != ""
	sess.VerifierResults = out.Execution.VerifierResults
	rep.SnapshotID = out.Execution.SnapshotID
	rep.RollbackAvailable = out.Execution.SnapshotID != ""
	if out.Execution.SnapshotID != "" {
		rep.RollbackCommand = "agentlink restore last"
	}
	if out.Execution.Status == recipe.StatusVerifierFailed {
		plannerCycle.StateTransitions = append(plannerCycle.StateTransitions, "RetryOrRollback")
		plannerCycle.Result = recipe.StatusVerifierFailed
		addCycle(plannerCycle)
		rep.Status = rollbackLatest(ctx, runner, home, out.Execution.SnapshotID)
		rep.FinalSummary = "Gemma-guided repair failed verification. Rolled back where possible."
		return finish(rep, sess, home)
	}
	if out.Execution.Error != "" {
		plannerCycle.Result = StatusFailed
		addCycle(plannerCycle)
		rep.Status = StatusFailed
		rep.FinalSummary = out.Execution.Error
		return finish(rep, sess, home)
	}
	plannerCycle.Result = StatusRepaired
	addCycle(plannerCycle)
	rep.Status = StatusRepaired
	rep.FinalSummary = "Gemma-guided reversible repair completed successfully."
	return finish(rep, sess, home)
}

type recipeAttempt struct {
	DryRun    recipe.RunResult
	Execution recipe.RunResult
}

func tryRecipe(ctx context.Context, runner command.Runner, reg recipe.Registry, home, id string, opts Options, params map[string]string) recipeAttempt {
	dry := recipe.Run(ctx, runner, reg, id, recipe.RunOptions{Home: home, DryRun: true, Params: params, CommandLine: append(opts.CommandLine, "--internal-guided-dry-run")})
	if opts.DryRun || !opts.Yes || dry.Error != "" {
		return recipeAttempt{DryRun: dry}
	}
	exec := recipe.Run(ctx, runner, reg, id, recipe.RunOptions{Home: home, Yes: true, Params: params, CommandLine: opts.CommandLine})
	return recipeAttempt{DryRun: dry, Execution: exec}
}

func finish(rep Report, sess session.Session, home string) Report {
	rep.EndedAt = time.Now().Format(time.RFC3339)
	sess.Finish(rep.Status)
	sess.SelectedRecipe = rep.SelectedRecipe
	sess.StopReason = rep.FinalSummary
	sess.Warnings = append(sess.Warnings, rep.Warnings...)
	if rep.SnapshotID != "" {
		sess.SnapshotID = rep.SnapshotID
		sess.RollbackAvailable = rep.RollbackAvailable
	}
	_ = session.NewStore(home).Save(&sess)
	rep.HumanReportPath = sess.HumanReportPath
	rep.CodexDispatchPath = sess.AgentDispatchPath
	if rep.Status == "" {
		rep.Status = StatusFailed
	}
	return rep
}

func MarshalReport(rep Report) string {
	data, _ := json.MarshalIndent(rep, "", "  ")
	return safety.RedactSensitive(string(data))
}

func HumanReport(rep Report) string {
	var b strings.Builder
	b.WriteString("Cactus AgentLink Rescue " + rep.ToolVersion + " guided rescue\n\n")
	b.WriteString("Target: " + rep.Target + "\n")
	b.WriteString("Mode: " + rep.Mode + "\n")
	b.WriteString("Status: " + rep.Status + "\n")
	if rep.SelectedRecipe != "" {
		b.WriteString("Recipe: " + rep.SelectedRecipe + "\n")
	}
	b.WriteString("Changes made: " + yesNo(rep.Mode == "execute" && rep.Status == StatusRepaired) + "\n")
	if rep.FinalSummary != "" {
		b.WriteString("\n" + rep.FinalSummary + "\n")
	}
	if rep.RollbackCommand != "" {
		b.WriteString("\nRollback:\n" + rep.RollbackCommand + "\n")
	}
	if rep.HumanReportPath != "" {
		b.WriteString("\nReport path: " + rep.HumanReportPath + "\n")
	}
	return safety.RedactSensitive(b.String())
}

func mode(opts Options) string {
	if opts.Yes && !opts.DryRun {
		return "execute"
	}
	return "dry-run"
}

func failureClasses(f facts.Facts) []string {
	seen := map[string]bool{}
	var out []string
	if len(f.Network.Classifications) > 0 {
		for _, item := range f.Network.Classifications {
			if item != "" && !seen[item] {
				out = append(out, item)
				seen[item] = true
			}
		}
	}
	if len(f.LikelyFailures) > 0 {
		for _, item := range f.LikelyFailures {
			if item != "" && !seen[item] {
				out = append(out, item)
				seen[item] = true
			}
		}
	}
	return out
}

func healthy(f facts.Facts) bool {
	classes := failureClasses(f)
	if len(classes) == 1 && classes[0] == classify.OK {
		return f.Network.RecommendedRepairLevel == "" || f.Network.RecommendedRepairLevel == "none"
	}
	return false
}

func factsSummary(f facts.Facts) map[string]any {
	return map[string]any{
		"os":             f.OS,
		"arch":           f.Arch,
		"proxyEnvCount":  len(f.ProxyEnv),
		"recommended":    f.Network.RecommendedRepairLevel,
		"defaultRouteOK": f.Network.Network.DefaultRoute.Present,
		"codexConfig":    f.ConfigPaths["codex"].Exists,
	}
}

func candidateRecipes(target string, f facts.Facts) []string {
	switch target {
	case "path":
		return []string{"macos-zsh-path-repair"}
	case "proxy":
		out := []string{}
		if len(f.ProxyEnv) > 0 || hasClass(f, classify.UserProxyDirty) {
			out = append(out, "proxy-clean-stale-env")
		}
		if hasClass(f, classify.GitProxyDirty) || hasClass(f, classify.NpmProxyDirty) {
			out = append(out, "npm-git-proxy-conflict-repair")
		}
		if len(out) == 0 {
			out = append(out, "proxy-clean-stale-env")
		}
		return out
	case "codex":
		return []string{"codex-config-parse-repair"}
	case "keys":
		return []string{"api-key-detection-redaction"}
	case "auto":
		if hasClass(f, classify.UserProxyDirty) || hasClass(f, classify.GitProxyDirty) || hasClass(f, classify.NpmProxyDirty) {
			return candidateRecipes("proxy", f)
		}
		return nil
	default:
		return nil
	}
}

func hasClass(f facts.Facts, c string) bool {
	for _, item := range failureClasses(f) {
		if item == c {
			return true
		}
	}
	return false
}

func nonAutoTarget(target string, f facts.Facts) string {
	if target != "auto" {
		return target
	}
	if hasClass(f, classify.UserProxyDirty) || hasClass(f, classify.GitProxyDirty) || hasClass(f, classify.NpmProxyDirty) {
		return "proxy"
	}
	return "path"
}

func safeRecipe(reg recipe.Registry, id string) bool {
	r, ok := reg.Get(id)
	if !ok {
		return false
	}
	switch r.Risk {
	case recipe.RiskReadOnly, recipe.RiskSafePatch, recipe.RiskReversiblePatch:
		return true
	default:
		return false
	}
}

func rollbackLatest(ctx context.Context, runner command.Runner, home, snapshotID string) string {
	if snapshotID == "" {
		return StatusFailed
	}
	rp, err := snapshot.Load(filepath.Join(system.UserRestorePointsDir(home), filepath.Base(snapshotID)))
	if err != nil {
		return StatusFailed
	}
	rp.SetMutationPolicy(system.MutationOptions{RealUserHome: home, ExtraAllowedPaths: manifestPaths(rp.Manifest)})
	if err := rp.RestoreAllWithRunner(ctx, runner); err != nil {
		return StatusFailed
	}
	return StatusRolledBack
}

func manifestPaths(m snapshot.Manifest) []string {
	var out []string
	for _, entry := range m.Entries {
		if entry.OriginalPath != "" {
			out = append(out, entry.OriginalPath)
		}
	}
	return out
}

func toSessionPlannerCalls(calls []brain.PlannerCall) []session.PlannerCall {
	out := make([]session.PlannerCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, session.PlannerCall{Index: c.Index, Backend: c.Backend, DurationMs: c.DurationMs, RawOutput: safety.RedactSensitive(c.RawOutput), Validation: c.Validation, Correction: c.Correction})
	}
	return out
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func ValidateReport(rep Report) error {
	if rep.SchemaVersion != 1 {
		return fmt.Errorf("invalid schema version")
	}
	if rep.ToolVersion == "" || rep.Target == "" || rep.Status == "" {
		return fmt.Errorf("missing required guided report fields")
	}
	return nil
}

var _ = planner.Decision{}
