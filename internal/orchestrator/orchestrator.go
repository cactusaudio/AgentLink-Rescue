package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/networkverify"
	"cactus-agentlink-rescue/internal/repair"
	"cactus-agentlink-rescue/internal/restartgate"
	"cactus-agentlink-rescue/internal/system"
	"cactus-agentlink-rescue/internal/ticket"
)

type Options struct {
	Target         string
	DryRun         bool
	Yes            bool
	MaxCycles      int
	TimeoutSeconds int
	RulesDir       string
	Home           string
}

func Run(ctx context.Context, runner command.Runner, opts Options) Report {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	target := opts.Target
	if target == "" {
		target = "auto"
	}
	if target == "clash-tun" {
		target = "clash-tun"
	}
	if opts.TimeoutSeconds <= 0 {
		opts.TimeoutSeconds = 600
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()
	start := time.Now()
	report := Report{
		SchemaVersion: 1,
		ToolVersion:   system.Version,
		StartedAt:     start.Format(time.RFC3339),
		Mode:          mode(opts),
		Target:        target,
		Status:        "failed",
		Cycles:        []Cycle{},
	}

	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: opts.RulesDir})
	diag := engine.Run(ctx)
	classify.Apply(&diag)
	tun := diagnose.DiagnoseTun(diag)
	report.FailureClasses = append([]string(nil), diag.Classifications...)
	if tun.RecommendedRepair == "tun" && !contains(report.FailureClasses, classify.ClashTunActiveOrStale) {
		report.FailureClasses = append(report.FailureClasses, classify.ClashTunActiveOrStale)
	}
	report.Cycles = append(report.Cycles, Cycle{
		Index:          1,
		State:          "CollectFacts",
		FailureClasses: report.FailureClasses,
		FactsSummary: map[string]any{
			"defaultRoute":   diag.Network.DefaultRoute.Present,
			"recommended":    diag.RecommendedRepairLevel,
			"tunRecommended": tun.RecommendedRepair,
			"providers":      tun.Providers,
		},
		Result: "facts_collected",
	})
	if target == "auto" && healthy(diag, tun) {
		report.Status = "healthy"
		report.HumanSummary = "Network and AgentLink path look healthy. No repair was run."
		return finish(report, opts.Home)
	}

	decision := supervise(diag, tun, target)
	report.GemmaUsed = brain.NewLlamaCLIBackend(runner, opts.Home).Available(ctx).BrainPackAvailable
	validationErr := ValidatePlan(decision, tun.RecommendedRepair == "tun")
	validateCycle := Cycle{Index: 2, State: "GemmaSupervise", FailureClasses: report.FailureClasses, SupervisorDecision: decision, PlanValidation: map[string]any{"ok": validationErr == nil}, Result: "plan_validated"}
	if validationErr != nil {
		validateCycle.PlanValidation = map[string]any{"ok": false, "error": validationErr.Error()}
		validateCycle.Result = "plan_refused"
		report.Cycles = append(report.Cycles, validateCycle)
		report.Status = "manual_action_required"
		report.HumanSummary = "No validated safe rescue plan is available."
		report.NextAction = "agentlink support bundle"
		return finish(report, opts.Home)
	}
	report.Cycles = append(report.Cycles, validateCycle)
	report.SelectedRecipe = decision.SelectedRecipe
	report.RequiresAdmin = decision.RequiresAdmin
	dry := opts.DryRun || !opts.Yes
	if !dry && decision.RequiresAdmin && !system.IsRoot() {
		ticketReport := ticket.Create(ctx, runner, ticket.Options{Home: opts.Home, Type: "clash-tun-fix", Version: system.Version})
		report.Status = "ticket_created"
		report.TerminalTicketPath = ticketReport.Directory
		report.HumanSummary = "This repair needs administrator permission. AgentLink created a Terminal repair ticket instead of running sudo in the GUI."
		report.NextAction = filepath.Join(ticketReport.Directory, "Run-Clash-TUN-Fix.command")
		report.Cycles = append(report.Cycles, Cycle{Index: 3, State: "PrivilegeGate", Result: "terminal_ticket_created", Execution: ticketReport})
		return finish(report, opts.Home)
	}
	res := repair.Run(ctx, runner, repair.Options{Level: repair.LevelTun, Yes: opts.Yes, DryRun: dry, JSON: true, RulesDir: opts.RulesDir})
	report.Cycles = append(report.Cycles, Cycle{Index: 3, State: "DryRunOrExecute", DryRun: res, Execution: res, Result: res.Status})
	if dry {
		report.Status = "planned"
		report.HumanSummary = "Targeted Clash/TUN repair dry-run is complete. No changes were made."
		report.NextAction = "agentlink ticket create --type clash-tun-fix --json"
		return finish(report, opts.Home)
	}
	report.SnapshotID = res.RestorePointID
	report.RollbackAvailable = res.RestorePointID != ""
	report.RollbackCommand = res.RollbackCommand
	switch res.Status {
	case "critical connectivity improved", "agent link restored":
		report.Status = "repaired"
		report.HumanSummary = "Targeted Clash/TUN repair completed and verification improved."
	case "rolled_back_after_worsening":
		report.Status = "rolled_back_after_worsening"
		report.HumanSummary = "Postflight was worse than preflight, so AgentLink automatically rolled back."
		report.WorsenedSignals = []string{"critical postflight comparator triggered"}
	default:
		gate := restartgate.Prepare(ctx, runner, opts.RulesDir)
		report.RestartGate = map[string]any{"restartRequired": gate.RestartRequired, "reason": gate.Reason, "postRestartCommand": gate.PostRestartCommand}
		if gate.RestartRequired {
			report.Status = "restart_required"
			report.HumanSummary = "Targeted repair did not fully release macOS NetworkExtension/TUN state. Restart gate is required."
			report.NextAction = gate.PostRestartCommand
		} else {
			report.Status = "manual_action_required"
			report.HumanSummary = "Targeted repair did not prove success. Export a support bundle before trying broader system reset."
			report.NextAction = "agentlink support bundle"
		}
	}
	verify := networkverify.Run(ctx, runner, opts.RulesDir, false)
	report.Cycles = append(report.Cycles, Cycle{Index: 4, State: "Verify", Verification: verify, Result: boolResult(verify.OK)})
	return finish(report, opts.Home)
}

func RescuePlan(ctx context.Context, runner command.Runner, opts Options) RescuePlanDecision {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	target := opts.Target
	if target == "" {
		target = "network"
	}
	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: opts.RulesDir})
	diag := engine.Run(ctx)
	classify.Apply(&diag)
	tun := diagnose.DiagnoseTun(diag)
	return supervise(diag, tun, target)
}

func supervise(diag diagnose.DiagnosticReport, tun diagnose.TunReport, target string) RescuePlanDecision {
	if target == "network" || target == "clash-tun" || target == "auto" {
		if tun.RecommendedRepair == "tun" {
			return RescuePlanDecision{
				SchemaVersion:          1,
				Intent:                 "repair",
				FailureClass:           classify.ClashTunActiveOrStale,
				Confidence:             0.92,
				SelectedRecipe:         "macos-clash-tun-force-repair",
				Evidence:               tun.SuspiciousSignatures,
				RequiresAdmin:          true,
				RequiresTerminalTicket: true,
				ExpectedVerifiers:      []string{"default_route_present", "raw_ip_ping_ok", "dns_lookup_ok", "https_baidu_ok", "stale_tun_absent_or_down", "awdl0_up"},
				ExplanationForUser:     "Clash/Mihomo TUN signatures are present, so targeted TUN runtime repair is safer than broad standard network reset.",
			}
		}
	}
	if len(diag.Classifications) == 1 && diag.Classifications[0] == classify.OK {
		return RescuePlanDecision{SchemaVersion: 1, Intent: "report", FailureClass: classify.OK, Confidence: 0.9, ExplanationForUser: "network looks healthy"}
	}
	return RescuePlanDecision{SchemaVersion: 1, Intent: "manual_action", FailureClass: "UNKNOWN", Confidence: 0.6, StopReason: "no high-confidence targeted rescue plan"}
}

func healthy(diag diagnose.DiagnosticReport, tun diagnose.TunReport) bool {
	return len(diag.Classifications) == 1 && diag.Classifications[0] == classify.OK && tun.RecommendedRepair == ""
}

func mode(opts Options) string {
	if opts.DryRun || !opts.Yes {
		return "dry-run"
	}
	if !system.IsRoot() {
		return "terminal-assisted"
	}
	return "execute"
}

func finish(report Report, home string) Report {
	report.EndedAt = time.Now().Format(time.RFC3339)
	if shouldWriteIncident(report.Status) {
		if path := writeIncident(report, home); path != "" {
			report.IncidentReportPath = path
		}
	}
	return report
}

func shouldWriteIncident(status string) bool {
	switch status {
	case "failed", "rolled_back_after_worsening", "restart_required", "manual_action_required", "ticket_created":
		return true
	default:
		return false
	}
}

func writeIncident(report Report, home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if home == "" {
		return ""
	}
	dir := filepath.Join(home, "Library", "Application Support", system.AppName, "incidents", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "incident.json"), data, 0644); err != nil {
		return ""
	}
	human := "Cactus AgentLink Rescue incident\n\nStatus: " + report.Status + "\nSummary: " + report.HumanSummary + "\nNext: " + report.NextAction + "\n"
	_ = os.WriteFile(filepath.Join(dir, "human-report.txt"), []byte(human), 0644)
	_ = os.WriteFile(filepath.Join(dir, "agent-dispatch.md"), []byte(human), 0644)
	return dir
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func boolResult(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}
