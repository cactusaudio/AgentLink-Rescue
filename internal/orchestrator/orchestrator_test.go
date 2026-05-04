package orchestrator

import "testing"

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
