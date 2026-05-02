package brain

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/session"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

type LoopOptions struct {
	Home        string
	Target      string
	DryRun      bool
	Yes         bool
	Online      bool
	JSON        bool
	CommandLine []string
}

type LoopResult struct {
	ToolVersion       string           `json:"toolVersion"`
	BrainEnabled      bool             `json:"brainEnabled"`
	Status            string           `json:"status"`
	Target            string           `json:"target"`
	DryRun            bool             `json:"dryRun"`
	Decision          any              `json:"plannerDecision,omitempty"`
	ValidationErrors  []string         `json:"validationErrors,omitempty"`
	PlannerCalls      []PlannerCall    `json:"plannerCalls,omitempty"`
	RecipeResult      recipe.RunResult `json:"recipeResult,omitempty"`
	SessionID         string           `json:"sessionId,omitempty"`
	HumanReportPath   string           `json:"humanReportPath,omitempty"`
	AgentDispatchPath string           `json:"agentDispatchPath,omitempty"`
	StopReason        string           `json:"stopReason,omitempty"`
	Error             string           `json:"error,omitempty"`
}

func RunLoop(ctx context.Context, runner command.Runner, reg recipe.Registry, opts LoopOptions) LoopResult {
	return runLoopWithBackend(ctx, runner, reg, opts, NewLlamaCLIBackend(runner, opts.Home))
}

func RunLoopWithBackendForTest(ctx context.Context, runner command.Runner, reg recipe.Registry, backend BrainBackend, opts LoopOptions) LoopResult {
	return runLoopWithBackend(ctx, runner, reg, opts, backend)
}

func runLoopWithBackend(ctx context.Context, runner command.Runner, reg recipe.Registry, opts LoopOptions, backend BrainBackend) LoopResult {
	start := time.Now()
	home := opts.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	res := LoopResult{ToolVersion: system.Version, BrainEnabled: true, Target: opts.Target, DryRun: opts.DryRun, Status: "started"}
	sess := session.New(system.Version, opts.CommandLine)
	sess.BrainEnabled = true
	sess.BrainModelID = DefaultModelID
	sess.SelectedTarget = opts.Target
	sess.DryRun = opts.DryRun
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "CollectFacts")
	f := facts.Collect(ctx, runner, home, opts.Target == "network")
	sess.InitialFacts = f
	sess.RealUser = f.RealUser
	sess.HostSummary = f.OS + "/" + f.Arch

	sess.BrainBackend = backend.Name()
	avail := backend.Available(ctx)
	if !avail.BrainPackAvailable {
		res.Status = "brain_unavailable"
		res.Error = "brain assets missing"
		res.StopReason = "brain assets missing"
		sess.StopReason = res.StopReason
		sess.ValidationErrors = append(sess.ValidationErrors, res.Error)
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "BrainPlan")
	planner := NewPlanner(backend, reg)
	plan, err := planner.Plan(ctx, PlanInput{Target: opts.Target, Home: home, DryRun: opts.DryRun, Yes: opts.Yes, Online: opts.Online, Facts: f})
	res.PlannerCalls = plan.Calls
	res.ValidationErrors = plan.ValidationErrors
	res.Decision = plan.Decision
	sess.PlannerCalls = toSessionPlannerCalls(plan.Calls)
	sess.PlannerDecision = plan.Decision
	sess.PlannerIntent = plan.Decision.Intent
	sess.PlannerConfidence = plan.Decision.Confidence
	sess.ValidationErrors = append(sess.ValidationErrors, plan.ValidationErrors...)
	_ = writePlannerRaw(home, &sess, plan.RawOutput)
	if err != nil {
		res.Status = "planner_invalid"
		res.Error = err.Error()
		res.StopReason = "planner decision invalid"
		sess.StopReason = res.StopReason
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "ValidateDecision")
	decision := plan.Decision
	if decision.Intent != "repair" {
		res.Status = "reported"
		res.StopReason = decision.StopReason
		if res.StopReason == "" {
			res.StopReason = "planner returned " + decision.Intent
		}
		sess.StopReason = res.StopReason
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	rec, ok := reg.Get(decision.SelectedRecipe.ID)
	if !ok {
		res.Status = "planner_invalid"
		res.Error = "selected recipe not found"
		sess.StopReason = res.Error
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	if err := ValidateAutoExecution(decision, rec, AutoPolicy{DryRun: opts.DryRun, Yes: opts.Yes, Online: opts.Online}); err != nil {
		res.Status = "policy_refused"
		res.Error = err.Error()
		res.StopReason = err.Error()
		sess.StopReason = res.StopReason
		sess.SelectedRecipe = rec.ID
		sess.RiskLevel = rec.Risk
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	sess.SelectedRecipe = rec.ID
	sess.RiskLevel = rec.Risk
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "DryRun")
	dry := recipe.Run(ctx, runner, reg, rec.ID, recipe.RunOptions{Home: home, DryRun: true, Params: decision.SelectedRecipe.Params, CommandLine: append(opts.CommandLine, "--internal-dry-run")})
	if dry.Error != "" {
		res.Status = "dry_run_failed"
		res.Error = dry.Error
		sess.StopReason = dry.Error
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	if opts.DryRun {
		res.Status = recipe.StatusDryRun
		res.RecipeResult = dry
		sess.Finish(res.Status)
		_ = session.NewStore(home).Save(&sess)
		return finishLoop(res, sess)
	}
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "Snapshot")
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "ExecuteRecipe")
	run := recipe.Run(ctx, runner, reg, rec.ID, recipe.RunOptions{Home: home, Yes: opts.Yes, Params: decision.SelectedRecipe.Params, CommandLine: opts.CommandLine})
	res.RecipeResult = run
	sess.SnapshotID = run.SnapshotID
	sess.SnapshotPath = run.SnapshotPath
	sess.ChangedFiles = run.ChangedFiles
	sess.VerifierResults = run.VerifierResults
	sess.RollbackAvailable = run.SnapshotID != ""
	sess.Warnings = append(sess.Warnings, run.Warnings...)
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "Verify")
	if run.Status == recipe.StatusVerifierFailed && run.SnapshotID != "" {
		sess.LoopStateTransitions = append(sess.LoopStateTransitions, "Rollback")
		if rp, loadErr := snapshot.Latest(system.UserRestorePointsDir(home)); loadErr == nil {
			rp.SetMutationPolicy(system.MutationOptions{RealUserHome: home, ExtraAllowedPaths: manifestPaths(rp.Manifest)})
			if restoreErr := rp.RestoreAllWithRunner(ctx, runner); restoreErr != nil {
				res.Status = recipe.StatusVerifierFailed
				res.Error = restoreErr.Error()
			} else {
				res.Status = recipe.StatusRolledBack
			}
		}
	} else {
		res.Status = run.Status
	}
	if res.Status == "" {
		res.Status = run.Status
	}
	if run.Error != "" {
		res.Error = run.Error
	}
	sess.StopReason = decision.StopReason
	sess.LoopStateTransitions = append(sess.LoopStateTransitions, "Report")
	_ = start
	sess.Finish(res.Status)
	_ = session.NewStore(home).Save(&sess)
	return finishLoop(res, sess)
}

func finishLoop(res LoopResult, sess session.Session) LoopResult {
	res.SessionID = sess.SessionID
	res.HumanReportPath = sess.HumanReportPath
	res.AgentDispatchPath = sess.AgentDispatchPath
	if res.StopReason == "" {
		res.StopReason = sess.StopReason
	}
	return res
}

func writePlannerRaw(home string, sess *session.Session, raw string) error {
	if raw == "" {
		return nil
	}
	store := session.NewStore(home)
	dir := store.SessionDir(sess.SessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "planner-raw-1.txt")
	if err := os.WriteFile(path, []byte(safety.RedactSensitive(raw)), 0644); err != nil {
		return err
	}
	sess.PlannerRawOutputPath = path
	return nil
}

func toSessionPlannerCalls(calls []PlannerCall) []session.PlannerCall {
	out := make([]session.PlannerCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, session.PlannerCall{
			Index:      c.Index,
			Backend:    c.Backend,
			DurationMs: c.DurationMs,
			RawOutput:  safety.RedactSensitive(c.RawOutput),
			Validation: c.Validation,
			Correction: c.Correction,
		})
	}
	return out
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

func MarshalLoopResult(res LoopResult) string {
	data, _ := json.MarshalIndent(res, "", "  ")
	return safety.RedactSensitive(string(data))
}
