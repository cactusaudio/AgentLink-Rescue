package orchestrator

import (
	"context"
	"testing"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
)

type fakeBrainBackend struct {
	available bool
	json      string
}

func (f fakeBrainBackend) Name() string { return "fake" }

func (f fakeBrainBackend) Available(context.Context) brain.BrainAvailability {
	return brain.BrainAvailability{BrainPackAvailable: f.available, ModelExists: f.available, ModelSHA256OK: f.available, RuntimeExecutable: f.available}
}

func (f fakeBrainBackend) Generate(context.Context, brain.BrainRequest) (brain.BrainResponse, error) {
	return brain.BrainResponse{RawText: f.json, ExtractedJSON: f.json, Backend: "fake"}, nil
}

func TestValidatePlanRejectsStandardWhenTunHighConfidence(t *testing.T) {
	err := ValidatePlan(RescuePlanDecision{SchemaVersion: 1, Intent: "repair", Confidence: 0.9, SelectedRecipe: "standard", RequiresAdmin: true, RequiresTerminalTicket: true}, true)
	if err == nil {
		t.Fatal("standard plan accepted despite high-confidence TUN")
	}
}

func TestValidatePlanRequiresTerminalTicketForAdmin(t *testing.T) {
	err := ValidatePlan(RescuePlanDecision{SchemaVersion: 1, Intent: "repair", Confidence: 0.9, SelectedRecipe: "macos-clash-tun-force-repair", RequiresAdmin: true}, true)
	if err == nil {
		t.Fatal("admin plan without terminal ticket accepted")
	}
}

func TestValidatePlanRejectsLowConfidenceRepair(t *testing.T) {
	err := ValidatePlan(RescuePlanDecision{SchemaVersion: 1, Intent: "repair", Confidence: 0.4, SelectedRecipe: "macos-clash-tun-force-repair"}, false)
	if err == nil {
		t.Fatal("low-confidence repair accepted")
	}
}

func TestValidatePlanRejectsUnknownRecipe(t *testing.T) {
	err := ValidatePlan(RescuePlanDecision{SchemaVersion: 1, Intent: "repair", Confidence: 0.9, SelectedRecipe: "unknown-recipe"}, false)
	if err == nil {
		t.Fatal("unknown rescue recipe accepted")
	}
}

func TestValidatePlanAcceptsLastResortOffer(t *testing.T) {
	err := ValidatePlan(RescuePlanDecision{SchemaVersion: 1, Intent: "last_resort_offer", Confidence: 0.8, SelectedRecipe: "macos-clean-network-baseline-reset", RequiresUserConsent: true}, false)
	if err != nil {
		t.Fatalf("last-resort offer rejected: %v", err)
	}
}

func TestSuperviseHealthyReportsNoRepair(t *testing.T) {
	decision := RescuePlanDecision{SchemaVersion: 1, Intent: "report", Confidence: 0.9}
	if decision.Intent == "repair" {
		t.Fatal("report decision should not be treated as repair")
	}
}

func TestDeterministicTunDecisionRequiresConsent(t *testing.T) {
	decision := supervise(diagnose.DiagnosticReport{}, diagnose.TunReport{RecommendedRepair: "tun", SuspiciousSignatures: []string{"utun0 198.18.0.1"}}, "clash-tun")
	if decision.Intent != "repair" || !decision.RequiresAdmin || !decision.RequiresTerminalTicket || !decision.RequiresUserConsent {
		t.Fatalf("bad deterministic TUN decision: %+v", decision)
	}
}

func TestDeterministicProtectedAudioVLANTrapRefusesRepair(t *testing.T) {
	decision := supervise(diagnose.DiagnosticReport{Classifications: []string{classify.ProtectedAudioVLANRouteTrap}}, diagnose.TunReport{RecommendedRepair: "tun"}, "network")
	if decision.Intent == "repair" || decision.SelectedRecipe != "" {
		t.Fatalf("protected route trap selected repair: %+v", decision)
	}
	if decision.FailureClass != classify.ProtectedAudioVLANRouteTrap || decision.StopReason == "" {
		t.Fatalf("protected route trap decision missing class/stop reason: %+v", decision)
	}
}

func TestDeterministicProtectedTopologyConstraintRefusesRepair(t *testing.T) {
	diag := diagnose.DiagnosticReport{
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true, Gateway: "192.168.0.1", Interface: "en0"},
			Interfaces:   []diagnose.NetworkInterface{{Name: "en0", Status: "active", IPv4: []string{"192.168.0.103"}}, {Name: "en1", Status: "active", IPv4: []string{"192.168.0.104"}}},
		},
		Reachability: diagnose.ReachabilityInfo{Gateway: diagnose.ProbeResult{Target: "192.168.0.1", OK: false}},
		Topology: diagnose.TopologyInfo{Interfaces: []diagnose.TopologyInterface{{
			Name: "en0", Protected: true, Roles: []string{"protected_media"}, RoleEvidence: []string{"AES67 audio VLAN"},
		}}},
	}
	decision := supervise(diag, diagnose.TunReport{RecommendedRepair: "tun"}, "network")
	if decision.Intent == "repair" || decision.FailureClass != classify.ProtectedAudioVLANRouteTrap {
		t.Fatalf("protected topology selected repair: %+v", decision)
	}
}

func TestArbitrationRejectsGemmaRepairAcrossProtectedBoundary(t *testing.T) {
	deterministic := RescuePlanDecision{SchemaVersion: 1, Intent: "manual_action", FailureClass: classify.ProtectedTopologyConstraint, Confidence: 0.9}
	gemma := RescuePlanDecision{SchemaVersion: 1, Intent: "repair", FailureClass: classify.ClashTunActiveOrStale, Confidence: 0.9, SelectedRecipe: "macos-clash-tun-force-repair", RequiresAdmin: true, RequiresTerminalTicket: true, RequiresUserConsent: true}
	got := arbitratePlans(deterministic, gemma, true)
	if got.Decision.Intent == "repair" || got.GemmaOverrideAccepted {
		t.Fatalf("Gemma relaxed protected boundary: %+v", got)
	}
	if got.GemmaOverrideRejectedReason == "" {
		t.Fatalf("missing protected-boundary rejection: %+v", got)
	}
}

func TestGemmaSupervisorDecisionParses(t *testing.T) {
	decision, err := gemmaSupervise(context.Background(), fakeBrainBackend{available: true, json: `{"schemaVersion":1,"intent":"repair","failureClass":"CLASH_TUN_ACTIVE_OR_STALE","confidence":0.91,"selectedRecipe":"macos-clash-tun-force-repair","evidence":["utun0 198.18.0.1"],"requiresAdmin":true,"requiresTerminalTicket":true,"requiresUserConsent":true,"expectedVerifiers":["raw_ip_ping_ok"],"explanationForUser":"Targeted TUN repair is appropriate."}`}, diagnose.DiagnosticReport{Classifications: []string{classify.ClashTunActiveOrStale}}, diagnose.TunReport{RecommendedRepair: "tun"}, "clash-tun")
	if err != nil {
		t.Fatalf("gemma supervisor parse failed: %v", err)
	}
	if decision.Intent != "repair" || decision.SelectedRecipe != "macos-clash-tun-force-repair" {
		t.Fatalf("unexpected gemma decision: %+v", decision)
	}
}

func TestArbitrationRetainsHighConfidenceTunWhenGemmaManualAction(t *testing.T) {
	deterministic := RescuePlanDecision{SchemaVersion: 1, Intent: "repair", FailureClass: classify.ClashTunActiveOrStale, Confidence: 0.92, SelectedRecipe: "macos-clash-tun-force-repair", RequiresAdmin: true, RequiresTerminalTicket: true, RequiresUserConsent: true}
	gemma := RescuePlanDecision{SchemaVersion: 1, Intent: "manual_action", FailureClass: classify.ClashTunActiveOrStale, Confidence: 0.8, ExplanationForUser: "ask user"}
	got := arbitratePlans(deterministic, gemma, true)
	if got.Decision.Intent != "repair" || got.Decision.SelectedRecipe != "macos-clash-tun-force-repair" || got.PlanSource != "deterministic" || got.GemmaOverrideAccepted {
		t.Fatalf("high-confidence TUN deterministic plan was not retained: %+v", got)
	}
	if got.GemmaOverrideRejectedReason == "" || got.SupervisorMode != "deterministic_with_gemma_commentary" {
		t.Fatalf("missing rejected override metadata: %+v", got)
	}
}

func TestArbitrationRetainsHighConfidenceTunWhenGemmaReports(t *testing.T) {
	deterministic := RescuePlanDecision{SchemaVersion: 1, Intent: "repair", FailureClass: classify.ClashTunActiveOrStale, Confidence: 0.92, SelectedRecipe: "macos-clash-tun-force-repair", RequiresAdmin: true, RequiresTerminalTicket: true, RequiresUserConsent: true}
	gemma := RescuePlanDecision{SchemaVersion: 1, Intent: "report", FailureClass: classify.ClashTunActiveOrStale, Confidence: 0.8, ExplanationForUser: "no repair"}
	got := arbitratePlans(deterministic, gemma, true)
	if got.Decision.Intent != "repair" || got.Decision.SelectedRecipe != "macos-clash-tun-force-repair" || got.PlanSource != "deterministic" || got.GemmaOverrideAccepted {
		t.Fatalf("Gemma report incorrectly downgraded TUN repair: %+v", got)
	}
}

func TestArbitrationAcceptsGemmaRepairWhenDeterministicLowConfidence(t *testing.T) {
	deterministic := RescuePlanDecision{SchemaVersion: 1, Intent: "manual_action", FailureClass: "UNKNOWN", Confidence: 0.5, StopReason: "ambiguous"}
	gemma := RescuePlanDecision{SchemaVersion: 1, Intent: "repair", FailureClass: classify.ClashTunActiveOrStale, Confidence: 0.87, SelectedRecipe: "macos-clash-tun-force-repair", RequiresAdmin: true, RequiresTerminalTicket: true, RequiresUserConsent: true}
	got := arbitratePlans(deterministic, gemma, false)
	if got.Decision.SelectedRecipe != "macos-clash-tun-force-repair" || got.PlanSource != "gemma" || !got.GemmaOverrideAccepted || got.SupervisorMode != "gemma" {
		t.Fatalf("valid Gemma repair was not accepted for low-confidence deterministic plan: %+v", got)
	}
}

func TestArbitrationRejectsInvalidGemmaRecipe(t *testing.T) {
	deterministic := RescuePlanDecision{SchemaVersion: 1, Intent: "manual_action", FailureClass: "UNKNOWN", Confidence: 0.5, StopReason: "ambiguous"}
	gemma := RescuePlanDecision{SchemaVersion: 1, Intent: "repair", FailureClass: "UNKNOWN", Confidence: 0.9, SelectedRecipe: "unknown-recipe"}
	got := arbitratePlans(deterministic, gemma, false)
	if got.PlanSource != "deterministic" || got.GemmaOverrideAccepted || got.GemmaOverrideRejectedReason == "" {
		t.Fatalf("invalid Gemma recipe was not rejected: %+v", got)
	}
}
