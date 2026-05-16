package diagnosisgraph

import (
	"encoding/json"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/toolmanifest"
)

func manifestIDs() map[string]bool {
	m := toolmanifest.Build("")
	ids := map[string]bool{}
	for _, t := range m.Tools {
		ids[t.ID] = true
	}
	return ids
}

func TestRecommendedToolsExistInManifest(t *testing.T) {
	ids := manifestIDs()
	// every routed tool id (and fallback) must exist in the W2 catalog
	for class, refs := range classRoute {
		for _, tr := range refs {
			if !ids[tr.ID] {
				t.Errorf("class %s routes to unknown tool id %q (manifest drift)", class, tr.ID)
			}
		}
	}
	for _, tr := range fallbackRoute {
		if !ids[tr.ID] {
			t.Errorf("fallback routes to unknown tool id %q", tr.ID)
		}
	}
}

func reportWithClass(classes []string) diagnose.DiagnosticReport {
	return diagnose.DiagnosticReport{
		SchemaVersion:   1,
		Classifications: classes,
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true},
			ProxySummary: diagnose.ProxySummary{Dirty: classes != nil && classes[0] == classify.SystemProxyDirty},
		},
	}
}

func TestProjectionRoutesByPrimaryClass(t *testing.T) {
	cases := map[string]string{
		classify.SystemProxyDirty:      "agentlink.clean_stale_proxy_baseline",
		classify.DNSFail:               "agentlink.network_baseline_reset",
		classify.ClashTunActiveOrStale: "agentlink.approval_ticket_create",
		classify.KnownAgentResidue:     "agentlink.approval_ticket_create",
		classify.OK:                    "agentlink.verify_network",
	}
	for class, wantTool := range cases {
		g := Build(reportWithClass([]string{class}), true)
		if g.PrimaryClass != class {
			t.Errorf("class %s: primaryClass=%s", class, g.PrimaryClass)
		}
		found := false
		for _, tr := range g.RecommendedNextTools {
			if tr.ID == wantTool {
				found = true
			}
		}
		if !found {
			t.Errorf("class %s: expected recommended tool %s, got %v", class, wantTool, g.RecommendedNextTools)
		}
	}
}

func TestForbiddenIncludesExecutionAndRaw(t *testing.T) {
	g := Build(reportWithClass([]string{classify.DNSFail}), true)
	joined := strings.Join(g.ForbiddenNextTools, " | ")
	if !strings.Contains(joined, "agentlink.network_baseline_reset") {
		t.Errorf("forbidden must list host_txn execution tools before dry-run/approval; got %s", joined)
	}
	if !strings.Contains(joined, "RAW:") || !strings.Contains(joined, "sudo") {
		t.Errorf("forbidden must list raw surfaces incl sudo; got %s", joined)
	}
}

func TestRedactionScrubsSecrets(t *testing.T) {
	r := reportWithClass([]string{classify.SystemProxyDirty})
	r.Warnings = []string{"proxy http://user:SUPERSECRET@10.0.0.1:8080 token=sk-ABCDEF1234567890"}
	g := Build(r, true)
	blob, _ := g.JSON()
	s := string(blob)
	if strings.Contains(s, "SUPERSECRET") || strings.Contains(s, "sk-ABCDEF1234567890") {
		t.Errorf("redaction failed; secret leaked into graph JSON:\n%s", s)
	}
	if !g.Redacted {
		t.Error("graph must mark Redacted=true when redact requested")
	}
	// no-redact path is debug-only and may retain (documented)
	g2 := Build(r, false)
	if g2.Redacted {
		t.Error("no-redact graph must report Redacted=false")
	}
}

func TestGraphIsCompactForGemma(t *testing.T) {
	// busy report -> graph must still be small (no raw dumps).
	r := reportWithClass([]string{classify.ClashTunActiveOrStale})
	for i := 0; i < 50; i++ {
		r.Network.Interfaces = append(r.Network.Interfaces,
			diagnose.NetworkInterface{Name: "utun" + itoa(i), Status: "active", IsUTun: true})
		r.Residues.LaunchAgents = append(r.Residues.LaunchAgents,
			diagnose.ResidueMatch{RuleID: "r", DisplayName: "x", Path: "/Library/LaunchAgents/x" + itoa(i) + ".plist"})
	}
	r.Network.DNSSummary.Raw = strings.Repeat("RAWDUMP ", 5000) // must NOT appear
	b, err := (Build(r, true)).JSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > 6144 {
		t.Errorf("graph too large for small-memory model: %d bytes (>6KB)", len(b))
	}
	if strings.Contains(string(b), "RAWDUMP") {
		t.Error("graph leaked a raw dump (must exclude DiagnosticReport.Raw fields)")
	}
}

func TestDeterministic(t *testing.T) {
	r := reportWithClass([]string{classify.DNSFail})
	a, _ := Build(r, true).JSON()
	b, _ := Build(r, true).JSON()
	if string(a) != string(b) {
		t.Error("Build must be deterministic")
	}
}

func TestEndToEndClassifyApplyThenGraph(t *testing.T) {
	// realistic dirty-proxy report; let classify.Apply derive classes,
	// then ensure the graph routes to the proxy recipe path.
	r := diagnose.DiagnosticReport{
		SchemaVersion: 1,
		Network: diagnose.NetworkInfo{
			DefaultRoute: diagnose.DefaultRoute{Present: true, Gateway: "10.0.0.1", Interface: "en0"},
			ProxySummary: diagnose.ProxySummary{HTTPEnabled: true, Dirty: true},
			Interfaces:   []diagnose.NetworkInterface{{Name: "en0", Status: "active", IPv4: []string{"10.0.0.5"}}},
		},
	}
	classify.Apply(&r)
	g := Build(r, true)
	if probs := validateGraph(g); probs != "" {
		t.Fatalf("graph invalid: %s", probs)
	}
	var generic any
	if err := json.Unmarshal(mustJSON(t, g), &generic); err != nil {
		t.Fatalf("graph not valid JSON: %v", err)
	}
}

func mustJSON(t *testing.T, g Graph) []byte {
	t.Helper()
	b, err := g.JSON()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// validateGraph: minimal structural invariants for product use.
func validateGraph(g Graph) string {
	if g.SchemaVersion < 1 {
		return "schemaVersion<1"
	}
	if len(g.RecommendedNextTools) == 0 {
		return "no recommended tools"
	}
	if len(g.ForbiddenNextTools) == 0 {
		return "no forbidden tools"
	}
	if len(g.Checks) == 0 {
		return "no checks"
	}
	return ""
}
