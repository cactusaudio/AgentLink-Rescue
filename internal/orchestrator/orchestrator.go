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
	"cactus-agentlink-rescue/internal/diagnosisgraph"
	"cactus-agentlink-rescue/internal/genome"
	"cactus-agentlink-rescue/internal/genomekernel"
	"cactus-agentlink-rescue/internal/networkverify"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/repair"
	"cactus-agentlink-rescue/internal/restartgate"
	"cactus-agentlink-rescue/internal/system"
	"cactus-agentlink-rescue/internal/ticket"
	"cactus-agentlink-rescue/internal/verifier"
)

type Options struct {
	Target         string
	DryRun         bool
	Yes            bool
	MaxCycles      int
	TimeoutSeconds int
	RulesDir       string
	RecipesDir     string
	GenomeDir      string
	Home           string
	PackageRoot    string
	BrainBackend   brain.BrainBackend
}

func Run(ctx context.Context, runner command.Runner, opts Options) Report {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	target := opts.Target
	if target == "" {
		target = "auto"
	}
	if opts.TimeoutSeconds <= 0 {
		opts.TimeoutSeconds = 600
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutSeconds)*time.Second)
	defer cancel()
	start := time.Now()
	report := Report{
		SchemaVersion:  1,
		ToolVersion:    system.Version,
		StartedAt:      start.Format(time.RFC3339),
		Mode:           mode(opts),
		Target:         target,
		Status:         "failed",
		SupervisorMode: "deterministic",
		PlanSource:     "deterministic",
		Cycles:         []Cycle{},
	}

	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: opts.RulesDir})
	diag := engine.Run(ctx)
	classify.Apply(&diag)
	// diagnose -> classify -> graph: project the heavy report into the compact
	// confidence/edge-weighted diagnosis graph so the kernel routes on a
	// ranked primary hypothesis, not a flat unordered class set.
	graph := diagnosisgraph.Build(diag, true)
	// Knowledge integration: surface the distilled genome cards for the primary
	// class as read-only advisory context. Best-effort — it never blocks a
	// rescue and never influences the decision.
	report.GenomeAdvisory = genomeAdvisory(graph, opts.GenomeDir)
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
			"defaultRoute":         diag.Network.DefaultRoute.Present,
			"recommended":          diag.RecommendedRepairLevel,
			"tunRecommended":       tun.RecommendedRepair,
			"providers":            tun.Providers,
			"protectedConstraints": len(diag.Topology.ProtectedConstraints),
			"repairCorridor":       diag.Topology.RepairCorridor,
			"graphPrimaryClass":    graph.PrimaryClass,
			"graphConfidence":      graph.Confidence,
		},
		Result: "facts_collected",
	})
	if contains(report.FailureClasses, classify.ProtectedAudioVLANRouteTrap) || contains(report.FailureClasses, classify.ProtectedTopologyConstraint) || diagnose.HasProtectedTopologyConstraint(diag) {
		decision := topologyBoundaryDecision(diag)
		report.Status = "manual_action_required"
		if connectivityBroken(diag) {
			// Connectivity is the first imperative: even under a protected
			// topology, offer the nuclear reset to get this Mac back online. It
			// overrides the protected boundary and is reversible via a captured
			// network snapshot.
			report.HumanSummary = "Protected topology detected, but this Mac cannot get online. Restoring internet is the first priority — the nuclear network reset will get it back online (it overrides protected topology and is reversible via a captured snapshot)."
			report.NextAction = "sudo ./bin/agentlink rescue --level nuclear --yes --json"
			report.Cycles = append(report.Cycles, Cycle{Index: 2, State: "TopologyBoundary", SupervisorDecision: decision, Result: "connectivity_first_nuclear_offered"})
		} else {
			report.HumanSummary = decision.ExplanationForUser
			report.NextAction = "agentlink support bundle --json"
			report.Cycles = append(report.Cycles, Cycle{Index: 2, State: "TopologyBoundary", SupervisorDecision: decision, Result: "protected_topology_online_no_action"})
		}
		return finish(report, opts.Home)
	}
	brainBackend := opts.BrainBackend
	if brainBackend == nil {
		brainBackend = brain.NewLlamaCLIBackend(runner, opts.Home)
	}
	brainAvailability := brainBackend.Available(ctx)
	report.BrainAvailable = brainAvailability.BrainPackAvailable
	if healthy(diag, tun) {
		report.Status = "healthy"
		report.HumanSummary = "Network and AgentLink path look healthy. No repair was run."
		return finish(report, opts.Home)
	}

	deterministicPlan := supervise(diag, tun, target)
	decision := deterministicPlan
	if report.BrainAvailable {
		if gemmaDecision, err := gemmaSupervise(ctx, brainBackend, diag, tun, target); err == nil {
			report.GemmaCalled = true
			report.GemmaCallCount = 1
			arbitrated := arbitratePlans(deterministicPlan, gemmaDecision, tun.RecommendedRepair == "tun")
			decision = arbitrated.Decision
			report.SupervisorMode = arbitrated.SupervisorMode
			report.PlanSource = arbitrated.PlanSource
			report.GemmaOverrideAccepted = arbitrated.GemmaOverrideAccepted
			report.GemmaOverrideRejectedReason = arbitrated.GemmaOverrideRejectedReason
			if arbitrated.Warning != "" {
				report.Warnings = append(report.Warnings, arbitrated.Warning)
			}
		} else {
			report.SupervisorMode = "deterministic"
			report.PlanSource = "deterministic"
			report.Warnings = append(report.Warnings, "Gemma supervisor unavailable; deterministic policy selected the rescue plan: "+err.Error())
		}
	}
	validationErr := ValidatePlan(decision, tun.RecommendedRepair == "tun")
	validateCycle := Cycle{Index: 2, State: "SupervisorDecision", FailureClasses: report.FailureClasses, SupervisorDecision: decision, PlanValidation: map[string]any{"ok": validationErr == nil, "planSource": report.PlanSource}, Result: "plan_validated"}
	if report.GemmaCalled {
		validateCycle.PlanValidation["gemmaOverrideAccepted"] = report.GemmaOverrideAccepted
		if report.GemmaOverrideRejectedReason != "" {
			validateCycle.PlanValidation["gemmaOverrideRejectedReason"] = report.GemmaOverrideRejectedReason
		}
	}
	if validationErr != nil {
		validateCycle.PlanValidation = map[string]any{"ok": false, "error": validationErr.Error(), "planSource": report.PlanSource}
		validateCycle.Result = "plan_refused"
		report.Cycles = append(report.Cycles, validateCycle)
		report.Status = "manual_action_required"
		report.HumanSummary = "No validated safe rescue plan is available."
		report.NextAction = "agentlink support bundle"
		return finish(report, opts.Home)
	}
	report.Cycles = append(report.Cycles, validateCycle)
	if decision.Intent != "repair" {
		// recipe -> planner -> orchestrator -> verifier -> safety: the
		// deterministic TUN supervisor produced no executable privileged
		// repair, so consult the registry-driven planner for a bounded repair
		// routed to the diagnosis graph's primary class — either a reversible
		// user-config recipe (recipe.Run) or a privileged network-state tier
		// (repair.Run, reversible via captured network snapshot).
		if reg, regErr := recipe.LoadRegistry(opts.RecipesDir); regErr == nil {
			pd, kind := plannerRepairPlan(graph, reg, verifier.NewRegistry())
			switch kind {
			case "recipe":
				return runReversibleRepair(ctx, runner, reg, pd, opts, report)
			case "level":
				return runLevelRepair(ctx, runner, pd, opts, report)
			}
		}
		report.SelectedRecipe = decision.SelectedRecipe
		report.RequiresAdmin = decision.RequiresAdmin
		switch decision.Intent {
		case "report", "probe":
			report.Status = "manual_action_required"
			report.HumanSummary = decision.ExplanationForUser
			if report.HumanSummary == "" {
				report.HumanSummary = "AgentLink did not select a writable repair."
			}
			report.NextAction = "agentlink support bundle"
		case "restart_gate":
			report.Status = "restart_required"
			report.HumanSummary = decision.ExplanationForUser
			if report.HumanSummary == "" {
				report.HumanSummary = "Restart gate requested by validated supervisor decision."
			}
			report.NextAction = "agentlink restart-gate verify --json"
		case "last_resort_offer":
			report.Status = "manual_action_required"
			report.HumanSummary = decision.ExplanationForUser
			if report.HumanSummary == "" {
				report.HumanSummary = "Last-resort clean network baseline reset requires explicit user consent."
			}
			report.NextAction = "sudo ./bin/agentlink rescue --level clean-baseline --yes --json"
		default:
			report.Status = "manual_action_required"
			report.HumanSummary = decision.StopReason
			if report.HumanSummary == "" {
				report.HumanSummary = "No high-confidence targeted rescue plan is available."
			}
			report.NextAction = "agentlink support bundle"
		}
		// Connectivity-first: if no bounded repair restored the network, the
		// nuclear reset is the decisive get-online action.
		if report.Status == "manual_action_required" && connectivityBroken(diag) {
			report.HumanSummary = "No bounded repair restored connectivity. The nuclear network reset is the decisive get-online action — it resets this Mac to a clean, online-capable baseline (reversible via a captured snapshot)."
			report.NextAction = "sudo ./bin/agentlink rescue --level nuclear --yes --json"
		}
		return finish(report, opts.Home)
	}
	if decision.SelectedRecipe != "macos-clash-tun-force-repair" {
		report.Status = "manual_action_required"
		report.HumanSummary = "Validated repair intent selected an unknown or unsupported recipe."
		report.NextAction = "agentlink support bundle"
		report.Cycles = append(report.Cycles, Cycle{Index: 3, State: "ValidatePlan", Result: "unsupported_recipe", PlanValidation: map[string]any{"ok": false, "recipe": decision.SelectedRecipe}})
		return finish(report, opts.Home)
	}
	report.SelectedRecipe = decision.SelectedRecipe
	report.RequiresAdmin = decision.RequiresAdmin
	dry := opts.DryRun || !opts.Yes
	if !dry && decision.RequiresAdmin && !system.IsRoot() {
		ticketReport := ticket.Create(ctx, runner, ticket.Options{Home: opts.Home, Type: "clash-tun-fix", Version: system.Version, PackageRoot: opts.PackageRoot})
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
		gate := restartgate.Prepare(ctx, runner, opts.RulesDir, restartgate.Context{TunRepairAttempted: true, AfterRestorePoint: res.RestorePointID})
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

const rescueSupervisorPrompt = `You are Cactus AgentLink Rescue Supervisor.
You choose bounded rescue intent only. You cannot execute commands. You cannot write shell.
Return exactly one RescuePlanDecision JSON object. No markdown. No prose. No code fences.
Allowed intents: repair, probe, report, restart_gate, manual_action, last_resort_offer.
Allowed repair recipe in this release: macos-clash-tun-force-repair.
Allowed last-resort recipe: macos-clean-network-baseline-reset.
If Clash/Mihomo TUN evidence is high-confidence, prefer macos-clash-tun-force-repair before standard/deep reset.
Privileged repair must set requiresAdmin=true, requiresTerminalTicket=true, requiresUserConsent=true.
If confidence is below 0.55, use probe, report, or manual_action.`

func gemmaSupervise(ctx context.Context, backend brain.BrainBackend, diag diagnose.DiagnosticReport, tun diagnose.TunReport, target string) (RescuePlanDecision, error) {
	payload := map[string]any{
		"target": target,
		"policy": map[string]any{
			"gemmaCannotExecuteShell":       true,
			"guiCannotRunSudo":              true,
			"runnerExecutesOnlyRecipes":     true,
			"verifierOwnsTruth":             true,
			"rollbackRequiredForMutation":   true,
			"cleanBaselineIsLastResortOnly": true,
		},
		"allowedRecipes": []string{
			"macos-clash-tun-force-repair",
			"macos-clean-network-baseline-reset",
		},
		"facts": map[string]any{
			"classifications":     diag.Classifications,
			"recommendedRepair":   diag.RecommendedRepairLevel,
			"defaultRoutePresent": diag.Network.DefaultRoute.Present,
			"defaultRouteGateway": diag.Network.DefaultRoute.Gateway,
			"defaultRouteIface":   diag.Network.DefaultRoute.Interface,
			"proxyDirty":          diag.Network.ProxySummary.Dirty,
			"tunRecommended":      tun.RecommendedRepair,
			"tunSignatures":       tun.SuspiciousSignatures,
			"providers":           tun.Providers,
			"topology":            diag.Topology,
			"repairCorridor":      diag.Topology.RepairCorridor,
		},
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	resp, err := backend.Generate(ctx, brain.BrainRequest{
		SystemPrompt: rescueSupervisorPrompt,
		UserPrompt:   string(data),
		MaxTokens:    1024,
		Temperature:  0,
		ContextSize:  8192,
		Timeout:      2 * time.Minute,
		ExpectJSON:   true,
	})
	if err != nil {
		return RescuePlanDecision{}, err
	}
	jsonText := resp.ExtractedJSON
	if jsonText == "" {
		jsonText, err = brain.ExtractPlannerJSONObject(resp.RawText)
		if err != nil {
			return RescuePlanDecision{}, err
		}
	}
	var decision RescuePlanDecision
	if err := json.Unmarshal([]byte(jsonText), &decision); err != nil {
		return RescuePlanDecision{}, err
	}
	return decision, nil
}

type arbitrationResult struct {
	Decision                    RescuePlanDecision
	PlanSource                  string
	SupervisorMode              string
	GemmaOverrideAccepted       bool
	GemmaOverrideRejectedReason string
	Warning                     string
}

func arbitratePlans(deterministicPlan, gemmaPlan RescuePlanDecision, tunHighConfidence bool) arbitrationResult {
	result := arbitrationResult{
		Decision:       deterministicPlan,
		PlanSource:     "deterministic",
		SupervisorMode: "deterministic_with_gemma_commentary",
	}
	if err := ValidatePlan(gemmaPlan, tunHighConfidence); err != nil {
		result.GemmaOverrideRejectedReason = "Gemma plan rejected by policy validator: " + err.Error()
		result.Warning = result.GemmaOverrideRejectedReason
		return result
	}
	if isProtectedBoundaryPlan(deterministicPlan) {
		result.GemmaOverrideRejectedReason = "protected topology repair corridor retained; model cannot relax deterministic boundary"
		result.Warning = result.GemmaOverrideRejectedReason
		return result
	}
	if isHighConfidenceSafetyCritical(deterministicPlan, tunHighConfidence) {
		if gemmaPlan.Intent == "repair" && gemmaPlan.SelectedRecipe == deterministicPlan.SelectedRecipe {
			result.Decision = mergeGemmaCommentary(deterministicPlan, gemmaPlan)
			return result
		}
		if hasValidatedContradictoryFacts(gemmaPlan) {
			result.Decision = gemmaPlan
			result.PlanSource = "gemma"
			result.SupervisorMode = "gemma"
			result.GemmaOverrideAccepted = true
			return result
		}
		result.GemmaOverrideRejectedReason = "high-confidence deterministic TUN repair retained; Gemma did not provide validated contradictory facts"
		result.Warning = result.GemmaOverrideRejectedReason
		return result
	}
	if deterministicPlan.Intent != "repair" || deterministicPlan.Confidence < 0.55 {
		if gemmaPlan.Intent == "repair" {
			result.Decision = gemmaPlan
			result.PlanSource = "gemma"
			result.SupervisorMode = "gemma"
			result.GemmaOverrideAccepted = true
			return result
		}
	}
	if gemmaPlan.Intent == "repair" && deterministicPlan.Intent == "repair" && gemmaPlan.SelectedRecipe == deterministicPlan.SelectedRecipe && gemmaPlan.Confidence >= deterministicPlan.Confidence {
		result.Decision = mergeGemmaCommentary(deterministicPlan, gemmaPlan)
	}
	return result
}

func isHighConfidenceSafetyCritical(plan RescuePlanDecision, tunHighConfidence bool) bool {
	if plan.Intent != "repair" || plan.Confidence < 0.85 {
		return false
	}
	if plan.FailureClass == classify.ClashTunActiveOrStale || plan.SelectedRecipe == "macos-clash-tun-force-repair" {
		return true
	}
	return tunHighConfidence
}

func mergeGemmaCommentary(deterministicPlan, gemmaPlan RescuePlanDecision) RescuePlanDecision {
	out := deterministicPlan
	if gemmaPlan.ExplanationForUser != "" {
		out.ExplanationForUser = gemmaPlan.ExplanationForUser
	}
	out.Evidence = appendUnique(out.Evidence, gemmaPlan.Evidence...)
	if len(gemmaPlan.ExpectedVerifiers) > 0 {
		out.ExpectedVerifiers = appendUnique(out.ExpectedVerifiers, gemmaPlan.ExpectedVerifiers...)
	}
	return out
}

func appendUnique(base []string, values ...string) []string {
	seen := map[string]bool{}
	for _, value := range base {
		seen[value] = true
	}
	out := append([]string(nil), base...)
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func hasValidatedContradictoryFacts(_ RescuePlanDecision) bool {
	// v0.5.0 does not let Gemma introduce new facts. Contradiction must come
	// from AgentLink probes, not from model commentary, so no downgrade is
	// accepted here.
	return false
}

func supervise(diag diagnose.DiagnosticReport, tun diagnose.TunReport, target string) RescuePlanDecision {
	if contains(diag.Classifications, classify.ProtectedAudioVLANRouteTrap) || contains(diag.Classifications, classify.ProtectedTopologyConstraint) || diagnose.HasProtectedTopologyConstraint(diag) {
		return topologyBoundaryDecision(diag)
	}
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
				RequiresUserConsent:    true,
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

func topologyBoundaryDecision(diag diagnose.DiagnosticReport) RescuePlanDecision {
	class := classify.ProtectedTopologyConstraint
	conf := 0.9
	if contains(diag.Classifications, classify.ProtectedAudioVLANRouteTrap) || diagnose.ProtectedAudioRouteTrap(diag) {
		class = classify.ProtectedAudioVLANRouteTrap
		conf = 0.95
	}
	return RescuePlanDecision{
		SchemaVersion:      1,
		Intent:             "manual_action",
		FailureClass:       class,
		Confidence:         conf,
		ExplanationForUser: "Protected topology detected. AgentLink will only gather route/network snapshots, support bundles, and incident reports; it will not run TUN cleanup, DHCP renew, route flush, proxy reset, clean-baseline, or VLAN mutation across protected surfaces.",
		StopReason:         "protected topology repair corridor",
	}
}

func isProtectedBoundaryPlan(plan RescuePlanDecision) bool {
	return plan.Intent != "repair" && (plan.FailureClass == classify.ProtectedAudioVLANRouteTrap || plan.FailureClass == classify.ProtectedTopologyConstraint)
}

func healthy(diag diagnose.DiagnosticReport, tun diagnose.TunReport) bool {
	return len(diag.Classifications) == 1 && diag.Classifications[0] == classify.OK && tun.RecommendedRepair == ""
}

// connectivityBroken reports the core "NIC up but no internet" condition: no
// default route, or every raw-IP / DNS / HTTPS probe failing. It is the trigger
// for offering the nuclear connectivity-first reset.
func connectivityBroken(diag diagnose.DiagnosticReport) bool {
	if !diag.Network.DefaultRoute.Present {
		return true
	}
	allFail := func(m map[string]diagnose.ProbeResult) bool {
		if len(m) == 0 {
			return false
		}
		for _, p := range m {
			if p.OK {
				return false
			}
		}
		return true
	}
	return allFail(diag.Reachability.RawIPs) || allFail(diag.Reachability.DNSNames) || allFail(diag.Reachability.HTTPSTargets)
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
	if err := os.MkdirAll(dir, 0700); err != nil {
		return ""
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "incident.json"), data, 0600); err != nil {
		return ""
	}
	human := "Cactus AgentLink Rescue incident\n\nStatus: " + report.Status + "\nSummary: " + report.HumanSummary + "\nNext: " + report.NextAction + "\n"
	_ = os.WriteFile(filepath.Join(dir, "human-report.txt"), []byte(human), 0600)
	_ = os.WriteFile(filepath.Join(dir, "agent-dispatch.md"), []byte(human), 0600)
	for _, cycle := range report.Cycles {
		writeArtifact := func(name string, value any) {
			if value == nil {
				return
			}
			data, _ := json.MarshalIndent(value, "", "  ")
			_ = os.WriteFile(filepath.Join(dir, name), data, 0600)
		}
		if cycle.SupervisorDecision != nil {
			writeArtifact("supervisor-decision.json", cycle.SupervisorDecision)
		}
		if cycle.DryRun != nil {
			writeArtifact("repair-result.json", cycle.DryRun)
		}
		if cycle.Execution != nil {
			switch cycle.State {
			case "PrivilegeGate":
				writeArtifact("terminal-ticket.json", cycle.Execution)
			default:
				writeArtifact("repair-result.json", cycle.Execution)
			}
		}
		if cycle.Verification != nil {
			writeArtifact("network-verify.json", cycle.Verification)
		}
	}
	if report.RestartGate != nil {
		writeArtifact := func(name string, value any) {
			data, _ := json.MarshalIndent(value, "", "  ")
			_ = os.WriteFile(filepath.Join(dir, name), data, 0600)
		}
		writeArtifact("restart-gate.json", report.RestartGate)
	}
	return dir
}

// plannerRepairPlan consults the registry-driven planner (the distilled routing
// judgment) for a bounded repair when the deterministic supervisor produced no
// executable privileged repair. It returns the validated decision and a kind:
//   - "recipe": an executable reversible/safe user-config recipe (recipe.Run,
//     no root) is the correct executor;
//   - "level":  a privileged network-state repair tier (repair.Run) is the
//     correct executor — reversible via the engine's captured network snapshot;
//   - "":       not an executable repair (report/probe/invalid) — fall through.
func plannerRepairPlan(graph diagnosisgraph.Graph, reg recipe.Registry, vreg verifier.Registry) (planner.Decision, string) {
	pd := planner.Plan(planner.Inputs{
		PrimaryClass:      graph.PrimaryClass,
		Classes:           graph.FailureClasses,
		Confidence:        graph.Confidence,
		ProtectedTopology: len(graph.ProtectedConstraints) > 0,
	}, reg)
	if planner.ValidateDecision(pd, reg, vreg) != nil || pd.Intent != "repair" {
		return pd, ""
	}
	if pd.RepairLevel != "" {
		return pd, "level"
	}
	rec, ok := reg.Get(pd.SelectedRecipe.ID)
	if !ok || rec.RequiresRoot || rec.Risk == recipe.RiskPrivilegedAction || !recipe.WritableRisk(rec.Risk) {
		return pd, ""
	}
	return pd, "recipe"
}

// runReversibleRepair executes a planner-selected reversible recipe through
// the recipe runner — snapshot, verifier, and rollback are all enforced inside
// recipe.Run — and maps the runner result onto the orchestrator report.
func runReversibleRepair(ctx context.Context, runner command.Runner, reg recipe.Registry, pd planner.Decision, opts Options, report Report) Report {
	report.SupervisorMode = "deterministic_planner"
	report.PlanSource = "planner"
	report.SelectedRecipe = pd.SelectedRecipe.ID
	report.Cycles = append(report.Cycles, Cycle{
		Index:              len(report.Cycles) + 1,
		State:              "PlannerDecision",
		FailureClasses:     report.FailureClasses,
		SupervisorDecision: pd,
		PlanValidation:     map[string]any{"ok": true, "planSource": "planner", "recipe": pd.SelectedRecipe.ID, "risk": pd.Risk},
		Result:             "plan_validated",
	})
	dry := opts.DryRun || !opts.Yes
	rr := recipe.Run(ctx, runner, reg, pd.SelectedRecipe.ID, recipe.RunOptions{
		Home:   opts.Home,
		DryRun: dry,
		Yes:    opts.Yes,
		JSON:   true,
		Params: pd.SelectedRecipe.Params,
	})
	execCycle := Cycle{Index: len(report.Cycles) + 1, State: "DryRunOrExecute", Execution: rr, Result: rr.Status}
	if dry {
		execCycle.DryRun = rr
	}
	report.Cycles = append(report.Cycles, execCycle)
	if rr.SnapshotID != "" {
		report.SnapshotID = rr.SnapshotID
		report.RollbackAvailable = true
		report.RollbackCommand = "agentlink rollback --id " + rr.SnapshotID
	}
	report.Warnings = append(report.Warnings, rr.Warnings...)
	if dry {
		report.Status = "planned"
		report.HumanSummary = "Reversible repair dry-run for " + pd.SelectedRecipe.ID + " is complete. No changes were made."
		report.NextAction = "agentlink recipe run " + pd.SelectedRecipe.ID + " --yes --json"
		return finish(report, opts.Home)
	}
	switch rr.Status {
	case recipe.StatusSuccess, recipe.StatusSuccessWithWarnings:
		report.Status = "repaired"
		report.HumanSummary = "Reversible repair " + pd.SelectedRecipe.ID + " completed and verifiers passed."
	case recipe.StatusRolledBack:
		report.Status = "rolled_back_after_worsening"
		report.HumanSummary = "Reversible repair verifiers worsened, so AgentLink rolled back automatically."
		report.WorsenedSignals = []string{"recipe verifier rollback"}
	case recipe.StatusVerifierFailed:
		report.Status = "manual_action_required"
		report.HumanSummary = "Reversible repair ran but verifiers did not confirm success."
		report.NextAction = "agentlink support bundle"
	default:
		report.Status = "manual_action_required"
		report.HumanSummary = pd.ExplanationForUser
		if report.HumanSummary == "" {
			report.HumanSummary = "Reversible repair did not complete; see recipe result."
		}
		report.NextAction = "agentlink support bundle"
	}
	return finish(report, opts.Home)
}

// runLevelRepair executes a planner-selected privileged network-state repair
// tier through the repair engine. The engine captures a network snapshot and
// auto-rolls-back on postflight worsening; the orchestrator never runs sudo in
// the GUI, so without root it produces the dry-run plan and a privileged
// NextAction. Yes=true is passed only to unlock the plan/level gate — DryRun
// guarantees no mutation unless the caller is genuinely root + consented.
func runLevelRepair(ctx context.Context, runner command.Runner, pd planner.Decision, opts Options, report Report) Report {
	report.SupervisorMode = "deterministic_planner"
	report.PlanSource = "planner"
	report.SelectedRecipe = "rescue:" + pd.RepairLevel
	report.RequiresAdmin = true
	report.Cycles = append(report.Cycles, Cycle{
		Index:              len(report.Cycles) + 1,
		State:              "PlannerDecision",
		FailureClasses:     report.FailureClasses,
		SupervisorDecision: pd,
		PlanValidation:     map[string]any{"ok": true, "planSource": "planner", "repairLevel": pd.RepairLevel, "risk": pd.Risk},
		Result:             "plan_validated",
	})
	dry := opts.DryRun || !opts.Yes || !system.IsRoot()
	res := repair.Run(ctx, runner, repair.Options{Level: pd.RepairLevel, Yes: true, DryRun: dry, RulesDir: opts.RulesDir})
	execCycle := Cycle{Index: len(report.Cycles) + 1, State: "DryRunOrExecute", Execution: res, Result: res.Status}
	if dry {
		execCycle.DryRun = res
	}
	report.Cycles = append(report.Cycles, execCycle)
	report.Warnings = append(report.Warnings, res.Warnings...)
	if res.RestorePointID != "" {
		report.SnapshotID = res.RestorePointID
		report.RollbackAvailable = true
		report.RollbackCommand = res.RollbackCommand
	}
	if dry {
		report.Status = "planned"
		report.HumanSummary = "Privileged " + pd.RepairLevel + "-tier network repair for " + pd.FailureClass + " is planned (dry-run). It is reversible via a captured network snapshot; apply it with administrator approval."
		report.NextAction = "sudo ./bin/agentlink rescue --level " + pd.RepairLevel + " --yes --json"
		return finish(report, opts.Home)
	}
	switch {
	case res.Status == "rolled_back_after_worsening":
		report.Status = "rolled_back_after_worsening"
		report.HumanSummary = "Postflight was worse than preflight, so AgentLink automatically rolled back."
		report.WorsenedSignals = []string{"critical postflight comparator triggered"}
	case res.Status == "partial_rollback_failed" || res.Status == "rollback_failed_after_worsening":
		report.Status = "manual_action_required"
		report.HumanSummary = "Network repair worsened connectivity and rollback did not fully complete; export a support bundle."
		report.NextAction = "agentlink support bundle"
	case res.ExitCode == 0:
		report.Status = "repaired"
		report.HumanSummary = "Privileged " + pd.RepairLevel + "-tier network repair completed and verification improved."
	default:
		report.Status = "manual_action_required"
		report.HumanSummary = "Network repair did not prove success; export a support bundle before trying a broader reset."
		report.NextAction = "agentlink support bundle"
	}
	return finish(report, opts.Home)
}

// genomeAdvisory best-effort projects the distilled genome cards for the
// graph's primary class into read-only report context. A missing/unreadable
// corpus returns nil and must never block a rescue.
func genomeAdvisory(graph diagnosisgraph.Graph, genomeDir string) *genomekernel.Advisory {
	if graph.PrimaryClass == "" || graph.PrimaryClass == classify.OK {
		return nil
	}
	corpus, err := genome.Load(genomeDir)
	if err != nil {
		return nil
	}
	adv, ok := genomekernel.AdvisoryForClass(corpus, graph.PrimaryClass, 3)
	if !ok {
		return nil
	}
	return &adv
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
