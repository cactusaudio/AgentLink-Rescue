package genome

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCorpusLoadsAllCardsAndStats(t *testing.T) {
	c, err := Load("../../agentlink_rescue_network_genome_v0_2")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Cards) != 300 {
		t.Fatalf("cards=%d", len(c.Cards))
	}
	if _, ok := c.ByID["MAC-PROXY-002"]; !ok {
		t.Fatal("MAC-PROXY-002 missing")
	}
	stats := c.Stats(t.TempDir())
	if stats.Cards != 300 || stats.Layers != 15 || stats.FTSRows != 300 || stats.SQLitePath == "" {
		t.Fatalf("bad stats: %+v", stats)
	}
}

func TestCorpusValidationRejectsDuplicateMissingAndUnknownLayer(t *testing.T) {
	card := validTestCard("TEST-001")
	c := Corpus{Cards: []Card{card, card}, ByID: map[string]Card{}, Layers: map[string]bool{"L05_proxy": true}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate not rejected: %v", err)
	}
	missing := validTestCard("TEST-002")
	missing.Title = ""
	c = Corpus{Cards: []Card{missing}, ByID: map[string]Card{}, Layers: map[string]bool{"L05_proxy": true}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing field not rejected: %v", err)
	}
	unknown := validTestCard("TEST-003")
	unknown.Layer = "L99_unknown"
	c = Corpus{Cards: []Card{unknown}, ByID: map[string]Card{}, Layers: map[string]bool{"L05_proxy": true}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "unknown layer") {
		t.Fatalf("unknown layer not rejected: %v", err)
	}
	unknownRisk := validTestCard("TEST-004")
	unknownRisk.Risk = "surprise"
	c = Corpus{Cards: []Card{unknownRisk}, ByID: map[string]Card{}, Layers: map[string]bool{"L05_proxy": true}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "unknown risk") {
		t.Fatalf("unknown risk not rejected: %v", err)
	}
	malformedRepair := validTestCard("TEST-005")
	malformedRepair.Repair.Mode = ""
	c = Corpus{Cards: []Card{malformedRepair}, ByID: map[string]Card{}, Layers: map[string]bool{"L05_proxy": true}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "malformed repair") {
		t.Fatalf("malformed repair not rejected: %v", err)
	}
	missingVerifier := validTestCard("TEST-006")
	missingVerifier.Verify = nil
	c = Corpus{Cards: []Card{missingVerifier}, ByID: map[string]Card{}, Layers: map[string]bool{"L05_proxy": true}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "verify") {
		t.Fatalf("missing verifier not rejected: %v", err)
	}
}

func TestDiagnoseBrowserCodexReturnsCardBackedTopThree(t *testing.T) {
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", t.TempDir())
	d, err := Diagnose("../../agentlink_rescue_network_genome_v0_2", "browser works but codex fails", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Hypotheses) != 3 {
		t.Fatalf("hypotheses=%d", len(d.Hypotheses))
	}
	foundProxy := false
	for _, h := range d.Hypotheses {
		if h.CardID == "" || h.RecipeGate.Result == "" {
			t.Fatalf("missing card/gate: %+v", h)
		}
		if h.CardID == "MAC-PROXY-002" {
			foundProxy = true
		}
	}
	if !strings.Contains(strings.Join(d.Hypotheses[0].WhyMatched, " "), "SQLite/FTS") {
		t.Fatalf("diagnosis did not report FTS candidate use: %+v", d.Hypotheses[0].WhyMatched)
	}
	if !foundProxy {
		t.Fatalf("MAC-PROXY-002 not in top three: %+v", d.Hypotheses)
	}
}

func TestRedactionStandardAndStrict(t *testing.T) {
	t.Setenv("USER", "bowei")
	text := "Authorization: Bearer secret-token\nhttps://user:pass@127.0.0.1:7890\n/Users/bowei/project\n192.168.1.10\n10.1.2.3\n172.16.1.10\n169.254.30.98\n198.18.0.1\nhostname: studio-host\n{\"auth\":\"dXNlcjpzdXBlcnNlY3JldA==\",\"identitytoken\":\"dockertoken-SECRET-1234567890\"}\n"
	standard := Redact(text, "standard")
	if strings.Contains(standard, "secret-token") || strings.Contains(standard, "user:pass") {
		t.Fatalf("standard redaction leaked: %s", standard)
	}
	strict := Redact(text, "strict")
	for _, leak := range []string{"/Users/bowei", "192.168.1.10", "10.1.2.3", "172.16.1.10", "169.254.30.98", "198.18.0.1", "studio-host", "dXNlcjpzdXBlcnNlY3JldA==", "dockertoken-SECRET"} {
		if strings.Contains(strict, leak) {
			t.Fatalf("strict redaction leaked %q: %s", leak, strict)
		}
	}
	if strings.Contains(strict, "<private-ip>.3") {
		t.Fatalf("strict redaction leaked: %s", strict)
	}
	jsonPath := `{"path":"/Users/bowei/project/report.json","other":"ok"}`
	redactedJSON := Redact(jsonPath, "strict")
	var parsed map[string]string
	if err := json.Unmarshal([]byte(redactedJSON), &parsed); err != nil {
		t.Fatalf("strict path redaction broke JSON: %v: %s", err, redactedJSON)
	}
	if strings.Contains(redactedJSON, "/Users/bowei") {
		t.Fatalf("strict JSON path redaction leaked user path: %s", redactedJSON)
	}
}

func TestFeatureExtractionFromSyntheticSnapshot(t *testing.T) {
	dir := t.TempDir()
	redacted := filepath.Join(dir, "redacted")
	if err := os.MkdirAll(redacted, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(redacted, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("proxy_scutil.txt", "HTTPEnable : 1\nHTTPProxy : 127.0.0.1\n")
	write("shell_proxy_env.txt", "HTTP_PROXY=http://127.0.0.1:7890\nNO_PROXY=localhost\n")
	write("dns_scutil.txt", "resolver #1\nscoped queries\n")
	write("route_default.txt", "interface: utun4\n")
	write("network_ifconfig.txt", "utun4: flags\n")
	write("git_proxy.txt", "http://127.0.0.1:7890\n")
	write("npm_proxy.txt", "null\n")
	write("npm_registry.txt", "https://registry.npmjs.org/\n")
	write("docker_config_hint.txt", "{}\n")
	sf, err := ExtractFeatures(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"system_proxy_enabled", "system_proxy_localhost", "shell_proxy_env_present", "default_route_through_utun", "utun_present", "tun_dns_route_risk_hint", "git_proxy_override_present"} {
		if sf.Features[key] != "true" {
			t.Fatalf("%s=%q features=%+v", key, sf.Features[key], sf.Features)
		}
	}
}

func TestNoProxyDoesNotCreateLocalhostProxyFalsePositive(t *testing.T) {
	dir := t.TempDir()
	redacted := filepath.Join(dir, "redacted")
	if err := os.MkdirAll(redacted, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(redacted, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("proxy_scutil.txt", "HTTPEnable : 0\n")
	write("shell_proxy_env.txt", "NO_PROXY=localhost,127.0.0.1\n")
	write("dns_scutil.txt", "")
	write("route_default.txt", "")
	write("network_ifconfig.txt", "")
	write("git_proxy.txt", "")
	write("npm_proxy.txt", "")
	write("npm_registry.txt", "")
	write("docker_config_hint.txt", "{}\n")
	sf, err := ExtractFeatures(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sf.Features["system_proxy_localhost"] == "true" || sf.Features["shell_proxy_localhost"] == "true" {
		t.Fatalf("NO_PROXY caused localhost proxy false positive: %+v", sf.Features)
	}
	if sf.Features["no_proxy_may_bypass_target"] != "true" {
		t.Fatalf("NO_PROXY signal missing: %+v", sf.Features)
	}
}

func TestCollectWritesManifestFeaturesAndRedactedRaw(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dockerDir := filepath.Join(home, ".docker")
	if err := os.MkdirAll(dockerDir, 0755); err != nil {
		t.Fatal(err)
	}
	dockerSecret := `{"auths":{"registry.example.com":{"auth":"dXNlcjpzdXBlcnNlY3JldA==","identitytoken":"dockertoken-SECRET-1234567890"}},"credsStore":"osxkeychain","credHelpers":{"registry.example.com":"desktop"}}`
	if err := os.WriteFile(filepath.Join(dockerDir, "config.json"), []byte(dockerSecret), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HTTP_PROXY", "http://user:pass@127.0.0.1:7890")
	t.Setenv("AGENTLINK_ALLOW_UNREDACTED_RAW", "")
	dir := t.TempDir()
	manifest, err := Collect(context.Background(), dir, "strict")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(dir, "snapshot_manifest.json"),
		filepath.Join(dir, "raw", "shell_proxy_env.txt"),
		filepath.Join(dir, "redacted", "shell_proxy_env.txt"),
		filepath.Join(dir, "features", "snapshot_features.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, "raw", "shell_proxy_env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "user:pass") {
		t.Fatalf("default raw directory leaked proxy credentials: %s", raw)
	}
	for _, rel := range []string{filepath.Join("raw", "docker_config_hint.txt"), filepath.Join("redacted", "docker_config_hint.txt")} {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, leak := range []string{"dXNlcjpzdXBlcnNlY3JldA==", "dockertoken-SECRET-1234567890", "osxkeychain", "desktop", "registry.example.com"} {
			if strings.Contains(text, leak) {
				t.Fatalf("%s leaked docker secret/value %q: %s", rel, leak, text)
			}
		}
		if !strings.Contains(text, `"docker_auths_present": true`) || !strings.Contains(text, `"docker_registry_count": 1`) {
			t.Fatalf("%s missing safe docker summary: %s", rel, text)
		}
	}
	featureData, err := os.ReadFile(filepath.Join(dir, "features", "snapshot_features.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(featureData), `"privacyMode": "strict"`) {
		t.Fatalf("features privacyMode not strict: %s", featureData)
	}
	if len(manifest.Warnings) == 0 || !strings.Contains(manifest.Warnings[0], "raw directory is redacted") {
		t.Fatalf("missing raw-redaction warning: %+v", manifest.Warnings)
	}
}

func TestGateOutcomes(t *testing.T) {
	c, err := Load("../../agentlink_rescue_network_genome_v0_2")
	if err != nil {
		t.Fatal(err)
	}
	if got := GateCard(c.ByID["MAC-PROXY-002"]); got.Result != "human_confirmed_allowed" {
		t.Fatalf("MAC-PROXY-002 gate=%+v", got)
	}
	if got := GateCard(c.ByID["DANTE-CLOCK-001"]); got.Result != "manual_only" {
		t.Fatalf("DANTE-CLOCK-001 gate=%+v", got)
	}
	if got := GateCard(c.ByID["CLASH-TUN-001"]); got.Result != "manual_only" {
		t.Fatalf("CLASH-TUN-001 gate=%+v", got)
	}
	blocked := validTestCard("TLS-BYPASS-TEST")
	blocked.Title = "TLS verification bypass"
	blocked.Repair.Commands = []string{"curl -k"}
	if got := GateCard(blocked); got.Result != "blocked" {
		t.Fatalf("TLS bypass not blocked: %+v", got)
	}
	noVerifier := validTestCard("NO-VERIFY")
	noVerifier.Verify = nil
	if got := GateCard(noVerifier); got.Result != "blocked" {
		t.Fatalf("no verifier not blocked: %+v", got)
	}
	noRollback := validTestCard("NO-ROLLBACK")
	noRollback.Rollback = nil
	if got := GateCard(noRollback); got.Result != "blocked" {
		t.Fatalf("no rollback not blocked: %+v", got)
	}
}

func TestReportsGeneratedAndRedacted(t *testing.T) {
	dir := t.TempDir()
	redacted := filepath.Join(dir, "redacted")
	if err := os.MkdirAll(redacted, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(redacted, "shell_proxy_env.txt"), []byte("HTTP_PROXY=http://user:pass@127.0.0.1:7890\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "features"), 0755); err != nil {
		t.Fatal(err)
	}
	sf, err := ExtractFeatures(dir)
	if err != nil {
		t.Fatal(err)
	}
	data := `{"schemaVersion":1,"privacyMode":"strict","features":{"shell_proxy_env_present":"true"}}`
	if len(sf.Features) > 0 {
		data = `{"schemaVersion":1,"privacyMode":"strict","features":{"shell_proxy_env_present":"true","shell_proxy_localhost":"true"}}`
	}
	if err := os.WriteFile(filepath.Join(dir, "features", "snapshot_features.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	md, err := WriteReport("../../agentlink_rescue_network_genome_v0_2", "markdown", dir, "browser works but codex fails", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "AgentLink Rescue Report") || !strings.Contains(md, "MAC-PROXY-002") {
		t.Fatalf("markdown report missing expected content:\n%s", md)
	}
	if strings.Contains(md, "user:pass") || strings.Contains(md, "/Users/") {
		t.Fatalf("report leaked credentials:\n%s", md)
	}
	jsonReport, err := WriteReport("../../agentlink_rescue_network_genome_v0_2", "json", dir, "browser works but codex fails", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonReport, `"hypotheses"`) || strings.Contains(jsonReport, "user:pass") || strings.Contains(jsonReport, "/Users/") {
		t.Fatalf("bad json report:\n%s", jsonReport)
	}
}

func TestSymptomRoutesAreLoadedFromCorpusYAML(t *testing.T) {
	root := t.TempDir()
	writeMiniCorpus(t, root, []Card{validTestCard("ROUTE-TEST-001")})
	routeDir := filepath.Join(root, "ontology")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatal(err)
	}
	routeYAML := `version: test
routes:
- id: SYM-CUSTOM-FROM-YAML
  user_phrase:
  - custom yaml route phrase
  priority_layers:
  - L05_proxy
  first_verifiers:
  - custom verifier
`
	if err := os.WriteFile(filepath.Join(routeDir, "symptom_routes.yaml"), []byte(routeYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", t.TempDir())
	d, err := Diagnose(root, "custom yaml route phrase", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Routes) == 0 || d.Routes[0].ID != "SYM-CUSTOM-FROM-YAML" || !strings.Contains(d.Routes[0].Reason, "symptom_routes.yaml") {
		t.Fatalf("route did not come from YAML: %+v", d.Routes)
	}
}

func TestRouteRankingBoostsAreLoadedFromCorpusYAML(t *testing.T) {
	root := t.TempDir()
	boosted := validTestCard("BOOSTED-001")
	boosted.Title = "Boosted card"
	plain := validTestCard("PLAIN-001")
	plain.Title = "Plain card"
	writeMiniCorpus(t, root, []Card{plain, boosted})
	routeDir := filepath.Join(root, "ontology")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatal(err)
	}
	routeYAML := `version: test
routes:
- id: SYM-BOOST-FROM-YAML
  user_phrase:
  - custom route ranking phrase
  priority_layers:
  - L05_proxy
  ranking_boosts:
  - card_id: BOOSTED-001
    score: 25
    reason: corpus-defined boost
`
	if err := os.WriteFile(filepath.Join(routeDir, "symptom_routes.yaml"), []byte(routeYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", t.TempDir())
	d, err := Diagnose(root, "custom route ranking phrase", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Hypotheses) == 0 || d.Hypotheses[0].CardID != "BOOSTED-001" {
		t.Fatalf("corpus ranking boost did not win: %+v", d.Hypotheses)
	}
	if !strings.Contains(strings.Join(d.Hypotheses[0].WhyMatched, " "), "corpus ranking boost") {
		t.Fatalf("boost provenance missing: %+v", d.Hypotheses[0].WhyMatched)
	}
}

func TestGenomeRuntimeDoesNotUsePythonOrScenarioBoost(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "genome.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, `exec.Command("python3"`) {
		t.Fatal("genome runtime still shells out to python3")
	}
	if strings.Contains(text, "func scenarioBoost") {
		t.Fatal("scenarioBoost hard-coded ranking function is still present")
	}
}

func TestCorruptFTSIndexRebuildsFromCorpusJSON(t *testing.T) {
	indexDir := t.TempDir()
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", indexDir)
	if err := os.WriteFile(filepath.Join(indexDir, "network_genome_v0_2.sqlite"), []byte("not sqlite"), 0644); err != nil {
		t.Fatal(err)
	}
	d, err := Diagnose("../../agentlink_rescue_network_genome_v0_2", "browser works but codex fails", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Hypotheses) == 0 {
		t.Fatal("no hypotheses after corrupt index rebuild")
	}
	stats, err := RebuildIndex("../../agentlink_rescue_network_genome_v0_2", indexDir)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FTSRows != 300 {
		t.Fatalf("bad rebuilt stats: %+v", stats)
	}
	if !stats.FTSIDsUsable || stats.FTSNullIDs != 0 {
		t.Fatalf("rebuilt FTS IDs are not usable: %+v", stats)
	}
}

func TestRebuiltFTSReturnsRealCardIDs(t *testing.T) {
	indexDir := t.TempDir()
	root := "../../agentlink_rescue_network_genome_v0_2"
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", indexDir)
	stats, err := RebuildIndex(root, indexDir)
	if err != nil {
		t.Fatal(err)
	}
	if !stats.FTSIDsUsable || stats.FTSNullIDs != 0 {
		t.Fatalf("rebuilt FTS IDs are not usable: %+v", stats)
	}
	ids, err := queryFTS(stats.SQLitePath, []string{"browser", "codex", "fails"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("FTS query returned no IDs")
	}
	for _, id := range ids {
		if id == "" {
			t.Fatalf("FTS returned empty ID: %+v", ids)
		}
	}
	if !containsString(ids, "CODEX-API-001") && !containsString(ids, "MAC-PROXY-002") {
		t.Fatalf("FTS query did not return expected card IDs: %+v", ids)
	}
	d, err := Diagnose(root, "browser works but codex fails", "", 5)
	if err != nil {
		t.Fatal(err)
	}
	foundFTSBacked := false
	for _, h := range d.Hypotheses {
		if strings.Contains(strings.Join(h.WhyMatched, " "), "SQLite/FTS") {
			foundFTSBacked = true
			if !containsString(ids, h.CardID) {
				t.Fatalf("hypothesis %s claimed FTS but was not returned by FTS IDs %+v", h.CardID, ids)
			}
		}
	}
	if !foundFTSBacked {
		t.Fatalf("no FTS-backed hypothesis found: %+v", d.Hypotheses)
	}
}

func TestRouteExpandedCandidatesDoNotClaimFTS(t *testing.T) {
	root := t.TempDir()
	card := validTestCard("ROUTE-ONLY-001")
	card.Title = "Layer-selected proxy card"
	card.Symptoms = []string{"unrelated managed proxy condition"}
	card.Observations = []string{"unrelated observation"}
	card.Discriminators = []string{"unrelated discriminator"}
	writeMiniCorpus(t, root, []Card{card})
	routeDir := filepath.Join(root, "ontology")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		t.Fatal(err)
	}
	routeYAML := `version: test
routes:
- id: SYM-ROUTE-ONLY
  user_phrase:
  - custom yaml route phrase
  priority_layers:
  - L05_proxy
`
	if err := os.WriteFile(filepath.Join(routeDir, "symptom_routes.yaml"), []byte(routeYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", t.TempDir())
	d, err := Diagnose(root, "custom yaml route phrase", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Hypotheses) != 1 {
		t.Fatalf("hypotheses=%d: %+v", len(d.Hypotheses), d.Hypotheses)
	}
	if strings.Contains(strings.Join(d.Hypotheses[0].WhyMatched, " "), "SQLite/FTS") {
		t.Fatalf("route-expanded candidate falsely claimed FTS: %+v", d.Hypotheses[0].WhyMatched)
	}
}

func TestSchemaFileCanDriveRiskValidation(t *testing.T) {
	root := t.TempDir()
	card := validTestCard("SCHEMA-TEST-001")
	writeMiniCorpus(t, root, []Card{card})
	schemaPath := filepath.Join(root, "schema", "failure_card.schema.yaml")
	if err := os.WriteFile(schemaPath, []byte(`version: test
required_fields:
- id
- title
- layer
- domain
- tags
- symptoms
- observations
- discriminators
- likely_causes
- safe_checks
- repair
- verify
- rollback
- risk
- false_positives
- source_anchors
- related
risk_classes:
- read_only
repair_contract:
  no_recipe_without_verifier: true
  no_mutating_recipe_without_rollback: true
`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unknown risk low") {
		t.Fatalf("schema risk_classes did not drive validation: %v", err)
	}
}

func writeMiniCorpus(t *testing.T, root string, cards []Card) {
	t.Helper()
	for _, dir := range []string{"cards", "schema", "indexes"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	payload := map[string]any{"version": "test", "generated_at": "test", "card_count": len(cards), "cards": cards}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cards", "failure_cards.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	manifestData, err := json.MarshalIndent(map[string]any{
		"version":      "test",
		"card_count":   len(cards),
		"layer_counts": map[string]int{"L05_proxy": len(cards)},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestData, 0644); err != nil {
		t.Fatal(err)
	}
	schema := `version: test
required_fields:
- id
- title
- layer
- domain
- tags
- symptoms
- observations
- discriminators
- likely_causes
- safe_checks
- repair
- verify
- rollback
- risk
- false_positives
- source_anchors
- related
risk_classes:
- read_only
- low
- medium
- high
- never_auto
repair_contract:
  no_recipe_without_verifier: true
  no_mutating_recipe_without_rollback: true
  never_auto_layers:
  - L10_multicast_discovery
  - L11_ptp_clock
  - L12_aoip_media
`
	if err := os.WriteFile(filepath.Join(root, "schema", "failure_card.schema.yaml"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}
}

func validTestCard(id string) Card {
	return Card{
		ID:             id,
		Title:          "Test proxy card",
		Layer:          "L05_proxy",
		Domain:         "proxy_control_plane",
		Tags:           []string{"proxy"},
		Symptoms:       []string{"browser works terminal fails"},
		Observations:   []string{"env proxy vars"},
		Discriminators: []string{"HTTP_PROXY set"},
		LikelyCauses:   []string{"stale proxy"},
		SafeChecks:     []string{"env | grep -i proxy"},
		Repair:         Repair{Mode: "low", Commands: []string{"unset HTTP_PROXY"}, AutomationClass: "human_confirmed"},
		Verify:         []string{"proxy vars absent"},
		Rollback:       []string{"restore env"},
		Risk:           "low",
		FalsePositives: []string{"system proxy"},
		Related:        []string{"MAC-PROXY-002"},
		SourceAnchors:  []string{"test"},
	}
}

func containsString(list []string, want string) bool {
	for _, got := range list {
		if got == want {
			return true
		}
	}
	return false
}
