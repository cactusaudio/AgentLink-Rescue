package genome

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	DefaultRootName = "agentlink_rescue_network_genome_v0_2"
	ToolVersion     = "agentlink-rescue-v0.5"
)

type Corpus struct {
	Root     string
	Manifest Manifest
	Cards    []Card
	ByID     map[string]Card
	Layers   map[string]bool
	Schema   SchemaConstraints
}

type SchemaConstraints struct {
	RequiredFields                  []string
	RiskClasses                     map[string]bool
	NeverAutoLayers                 map[string]bool
	NoRecipeWithoutVerifier         bool
	NoMutatingRecipeWithoutRollback bool
	ReadOnlyCollectorsOnlyByDefault bool
}

type Manifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	GeneratedAt string         `json:"generated_at"`
	CardCount   int            `json:"card_count"`
	LayerCounts map[string]int `json:"layer_counts"`
	RiskCounts  map[string]int `json:"risk_counts"`
}

type cardFile struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
	CardCount   int    `json:"card_count"`
	Cards       []Card `json:"cards"`
}

type Card struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Layer          string   `json:"layer"`
	Domain         string   `json:"domain"`
	Tags           []string `json:"tags"`
	Symptoms       []string `json:"symptoms"`
	Observations   []string `json:"observations"`
	Discriminators []string `json:"discriminators"`
	LikelyCauses   []string `json:"likely_causes"`
	SafeChecks     []string `json:"safe_checks"`
	Repair         Repair   `json:"repair"`
	Verify         []string `json:"verify"`
	Rollback       []string `json:"rollback"`
	Risk           string   `json:"risk"`
	FalsePositives []string `json:"false_positives"`
	Related        []string `json:"related"`
	SourceAnchors  []string `json:"source_anchors"`
	VersionAdded   string   `json:"version_added"`
}

type Repair struct {
	Mode            string   `json:"mode"`
	Commands        []string `json:"commands"`
	ManualSteps     []string `json:"manual_steps"`
	Preconditions   []string `json:"preconditions"`
	AutomationClass string   `json:"automation_class"`
}

type Stats struct {
	CorpusVersion string `json:"corpusVersion"`
	Cards         int    `json:"cards"`
	Layers        int    `json:"layers"`
	FTSRows       int    `json:"ftsRows"`
	FTSIDsUsable  bool   `json:"ftsIdsUsable"`
	FTSNullIDs    int    `json:"ftsNullIds"`
	GraphEdges    int    `json:"graphEdges"`
	SQLitePath    string `json:"sqlitePath"`
	IndexDir      string `json:"indexDir,omitempty"`
}

type RouteCandidate struct {
	ID              string       `json:"id"`
	CandidateLayers []string     `json:"candidateLayers"`
	CandidateDomain []string     `json:"candidateDomains,omitempty"`
	Reason          string       `json:"reason"`
	FirstVerifiers  []string     `json:"firstVerifiers,omitempty"`
	RankingBoosts   []RouteBoost `json:"rankingBoosts,omitempty"`
}

type RouteBoost struct {
	CardID   string `json:"cardId,omitempty"`
	IDPrefix string `json:"idPrefix,omitempty"`
	Layer    string `json:"layer,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Score    int    `json:"score"`
	Reason   string `json:"reason,omitempty"`
}

type Diagnosis struct {
	Timestamp     string            `json:"timestamp"`
	ToolVersion   string            `json:"toolVersion"`
	CorpusVersion string            `json:"corpusVersion"`
	Symptom       string            `json:"symptom"`
	SnapshotPath  string            `json:"snapshotPath,omitempty"`
	PrivacyMode   string            `json:"privacyMode,omitempty"`
	Routes        []RouteCandidate  `json:"routes"`
	Features      map[string]string `json:"features,omitempty"`
	Hypotheses    []Hypothesis      `json:"hypotheses"`
}

type Hypothesis struct {
	Rank                   int        `json:"rank"`
	CardID                 string     `json:"cardId"`
	Title                  string     `json:"title"`
	Layer                  string     `json:"layer"`
	Score                  int        `json:"score"`
	WhyMatched             []string   `json:"whyMatched"`
	RelevantSymptoms       []string   `json:"relevantSymptoms,omitempty"`
	RelevantDiscriminators []string   `json:"relevantDiscriminators,omitempty"`
	SnapshotFeaturesUsed   []string   `json:"snapshotFeaturesUsed,omitempty"`
	ConfirmingObservations []string   `json:"confirmingObservations,omitempty"`
	FalsifyingObservations []string   `json:"falsifyingObservations,omitempty"`
	NextReadOnlyChecks     []string   `json:"nextReadOnlyChecks,omitempty"`
	RiskClass              string     `json:"riskClass"`
	Verifier               []string   `json:"verifier,omitempty"`
	Rollback               []string   `json:"rollback,omitempty"`
	RecipeGate             GateResult `json:"recipeGate"`
}

type GateResult struct {
	CardID          string `json:"cardId"`
	Title           string `json:"title"`
	Layer           string `json:"layer"`
	RiskClass       string `json:"riskClass"`
	RecipeType      string `json:"recipeType"`
	VerifierPresent bool   `json:"verifierPresent"`
	RollbackPresent bool   `json:"rollbackPresent"`
	Result          string `json:"result"`
	Reason          string `json:"reason"`
	AllowedNextStep string `json:"allowedNextStep,omitempty"`
}

type SnapshotManifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	CreatedAt     string   `json:"createdAt"`
	ToolVersion   string   `json:"toolVersion"`
	PrivacyMode   string   `json:"privacyMode"`
	RawDir        string   `json:"rawDir"`
	RedactedDir   string   `json:"redactedDir"`
	FeaturesPath  string   `json:"featuresPath"`
	Redacted      bool     `json:"redacted"`
	HostOS        string   `json:"hostOS,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

type SnapshotFeatures struct {
	SchemaVersion int               `json:"schemaVersion"`
	CreatedAt     string            `json:"createdAt"`
	PrivacyMode   string            `json:"privacyMode"`
	Features      map[string]string `json:"features"`
	Evidence      map[string]string `json:"evidence,omitempty"`
}

type Report struct {
	Timestamp            string            `json:"timestamp"`
	ToolVersion          string            `json:"tool_version"`
	CorpusVersion        string            `json:"corpus_version"`
	Snapshot             map[string]any    `json:"snapshot"`
	Redaction            map[string]any    `json:"redaction"`
	Symptom              string            `json:"symptom"`
	Features             map[string]string `json:"features"`
	Routes               []RouteCandidate  `json:"routes"`
	Hypotheses           []Hypothesis      `json:"hypotheses"`
	RemainingUncertainty []string          `json:"remaining_uncertainty"`
}

var requiredFields = []string{
	"id", "title", "layer", "domain", "tags", "symptoms", "observations",
	"discriminators", "likely_causes", "safe_checks", "repair", "verify",
	"rollback", "risk", "false_positives", "source_anchors", "related",
}

var allowedRisks = map[string]bool{
	"read_only":  true,
	"low":        true,
	"medium":     true,
	"high":       true,
	"never_auto": true,
}

func FindRoot() (string, error) {
	if env := os.Getenv("AGENTLINK_GENOME_DIR"); env != "" {
		return env, nil
	}
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, DefaultRootName),
		filepath.Join(cwd, "..", DefaultRootName),
		filepath.Join(cwd, "..", "..", DefaultRootName),
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, DefaultRootName),
			filepath.Join(dir, "..", DefaultRootName),
			filepath.Join(dir, "..", "..", DefaultRootName),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(c, "cards", "failure_cards.json")); err == nil && !st.IsDir() {
			return filepath.Clean(c), nil
		}
	}
	return "", fmt.Errorf("genome corpus not found; set AGENTLINK_GENOME_DIR or place %s in the repo root", DefaultRootName)
}

func Load(root string) (Corpus, error) {
	if root == "" {
		var err error
		root, err = FindRoot()
		if err != nil {
			return Corpus{}, err
		}
	}
	var m Manifest
	if data, err := os.ReadFile(filepath.Join(root, "manifest.json")); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	data, err := os.ReadFile(filepath.Join(root, "cards", "failure_cards.json"))
	if err != nil {
		return Corpus{}, err
	}
	var cf cardFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return Corpus{}, err
	}
	if len(cf.Cards) == 0 {
		return Corpus{}, errors.New("failure card corpus contains zero cards")
	}
	if m.Version == "" {
		m.Version = cf.Version
	}
	if m.GeneratedAt == "" {
		m.GeneratedAt = cf.GeneratedAt
	}
	if m.CardCount == 0 {
		m.CardCount = len(cf.Cards)
	}
	schema := loadSchemaConstraints(filepath.Join(root, "schema", "failure_card.schema.yaml"))
	c := Corpus{Root: root, Manifest: m, Cards: cf.Cards, ByID: map[string]Card{}, Layers: map[string]bool{}, Schema: schema}
	for layer := range m.LayerCounts {
		c.Layers[layer] = true
	}
	if err := c.Validate(); err != nil {
		return Corpus{}, err
	}
	return c, nil
}

func (c Corpus) Validate() error {
	seen := map[string]bool{}
	schema := c.Schema
	if len(schema.RequiredFields) == 0 || len(schema.RiskClasses) == 0 {
		schema = defaultSchemaConstraints()
	}
	if c.Manifest.CardCount != 0 && c.Manifest.CardCount != len(c.Cards) {
		return fmt.Errorf("manifest card count %d does not match loaded cards %d", c.Manifest.CardCount, len(c.Cards))
	}
	for i, card := range c.Cards {
		if err := validateCard(card, c.Layers, schema); err != nil {
			return fmt.Errorf("card[%d] %s: %w", i, card.ID, err)
		}
		if seen[card.ID] {
			return fmt.Errorf("duplicate card id %s", card.ID)
		}
		seen[card.ID] = true
		c.ByID[card.ID] = card
	}
	return nil
}

func validateCard(card Card, layers map[string]bool, schema SchemaConstraints) error {
	if card.ID == "" {
		return errors.New("missing id")
	}
	for _, field := range schema.RequiredFields {
		if missingRequiredField(card, field) {
			return fmt.Errorf("missing required field %s", field)
		}
	}
	if len(layers) > 0 && !layers[card.Layer] {
		return fmt.Errorf("unknown layer %s", card.Layer)
	}
	if !schema.RiskClasses[card.Risk] {
		return fmt.Errorf("unknown risk %s", card.Risk)
	}
	if card.Repair.Mode == "" || card.Repair.AutomationClass == "" {
		return errors.New("malformed repair shape: repair.mode and repair.automation_class are required")
	}
	if schema.NoRecipeWithoutVerifier && len(card.Verify) == 0 {
		return errors.New("schema repair contract violation: recipe has no verifier")
	}
	if schema.NoMutatingRecipeWithoutRollback && hasMutatingCommand(card) && len(card.Rollback) == 0 {
		return errors.New("schema repair contract violation: mutating recipe has no rollback")
	}
	return nil
}

func missingRequiredField(card Card, field string) bool {
	switch field {
	case "id":
		return card.ID == ""
	case "title":
		return card.Title == ""
	case "layer":
		return card.Layer == ""
	case "domain":
		return card.Domain == ""
	case "tags":
		return len(card.Tags) == 0
	case "symptoms":
		return len(card.Symptoms) == 0
	case "observations":
		return len(card.Observations) == 0
	case "discriminators":
		return len(card.Discriminators) == 0
	case "likely_causes":
		return len(card.LikelyCauses) == 0
	case "safe_checks":
		return len(card.SafeChecks) == 0
	case "repair":
		return card.Repair.Mode == "" && card.Repair.AutomationClass == "" && len(card.Repair.Commands) == 0 && len(card.Repair.ManualSteps) == 0
	case "verify":
		return len(card.Verify) == 0
	case "rollback":
		return len(card.Rollback) == 0
	case "risk":
		return card.Risk == ""
	case "false_positives":
		return len(card.FalsePositives) == 0
	case "source_anchors":
		return len(card.SourceAnchors) == 0
	case "related":
		return len(card.Related) == 0
	default:
		return false
	}
}

func defaultSchemaConstraints() SchemaConstraints {
	risk := map[string]bool{}
	for k, v := range allowedRisks {
		risk[k] = v
	}
	return SchemaConstraints{
		RequiredFields:                  append([]string(nil), requiredFields...),
		RiskClasses:                     risk,
		NeverAutoLayers:                 map[string]bool{"L10_multicast_discovery": true, "L11_ptp_clock": true, "L12_aoip_media": true},
		NoRecipeWithoutVerifier:         true,
		NoMutatingRecipeWithoutRollback: true,
		ReadOnlyCollectorsOnlyByDefault: true,
	}
}

func loadSchemaConstraints(path string) SchemaConstraints {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultSchemaConstraints()
	}
	s := defaultSchemaConstraints()
	s.RequiredFields = nil
	s.RiskClasses = map[string]bool{}
	s.NeverAutoLayers = map[string]bool{}
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		if strings.HasPrefix(line, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			switch section {
			case "required_fields":
				s.RequiredFields = append(s.RequiredFields, val)
			case "risk_classes":
				s.RiskClasses[val] = true
			case "never_auto_layers":
				s.NeverAutoLayers[val] = true
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "no_recipe_without_verifier:"):
			s.NoRecipeWithoutVerifier = strings.Contains(line, "true")
		case strings.HasPrefix(line, "no_mutating_recipe_without_rollback:"):
			s.NoMutatingRecipeWithoutRollback = strings.Contains(line, "true")
		case strings.HasPrefix(line, "read_only_collectors_only_by_default:"):
			s.ReadOnlyCollectorsOnlyByDefault = strings.Contains(line, "true")
		}
	}
	if len(s.RequiredFields) == 0 {
		s.RequiredFields = append([]string(nil), requiredFields...)
	}
	if len(s.RiskClasses) == 0 {
		for k, v := range allowedRisks {
			s.RiskClasses[k] = v
		}
	}
	return s
}

func (c Corpus) Stats(indexDir string) Stats {
	layerSet := map[string]bool{}
	for _, card := range c.Cards {
		layerSet[card.Layer] = true
	}
	return Stats{
		CorpusVersion: c.Manifest.Version,
		Cards:         len(c.Cards),
		Layers:        len(layerSet),
		FTSRows:       len(c.Cards),
		FTSIDsUsable:  false,
		GraphEdges:    countGraphEdges(filepath.Join(c.Root, "indexes", "topology_edges.json")),
		SQLitePath:    filepath.Join(c.Root, "database", "network_genome_v0_2.sqlite"),
		IndexDir:      indexDir,
	}
}

func IndexStats(root, indexDir string) (Stats, error) {
	c, err := Load(root)
	if err != nil {
		return Stats{}, err
	}
	if indexDir == "" {
		indexDir = DefaultIndexDir()
	}
	stats := c.Stats(indexDir)
	indexPath := filepath.Join(indexDir, "network_genome_v0_2.sqlite")
	if sqliteUsable(indexPath) {
		stats.SQLitePath = indexPath
	} else {
		stats.SQLitePath = filepath.Join(c.Root, "database", "network_genome_v0_2.sqlite")
	}
	sqliteStats, err := inspectSQLiteStats(stats.SQLitePath)
	if err != nil {
		return stats, err
	}
	if sqliteStats.Cards > 0 {
		stats.Cards = sqliteStats.Cards
	}
	stats.FTSRows = sqliteStats.FTSRows
	stats.FTSIDsUsable = sqliteStats.FTSIDsUsable
	stats.FTSNullIDs = sqliteStats.FTSNullIDs
	return stats, nil
}

func countGraphEdges(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var list []any
	if json.Unmarshal(data, &list) == nil {
		return len(list)
	}
	var obj map[string]any
	if json.Unmarshal(data, &obj) == nil {
		total := 0
		if edges, ok := obj["edges"].([]any); ok {
			total += len(edges)
		}
		if edges, ok := obj["layer_edges"].([]any); ok {
			total += len(edges)
		}
		if edges, ok := obj["card_edges"].([]any); ok {
			total += len(edges)
		}
		if total > 0 {
			return total
		}
	}
	return 0
}

func DefaultIndexDir() string {
	if env := os.Getenv("AGENTLINK_GENOME_INDEX_DIR"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return filepath.Join(os.TempDir(), "agentlink-network-genome-index")
	}
	return filepath.Join(home, "Library", "Application Support", "Cactus AgentLink Rescue", "network-genome-index")
}

func RebuildIndex(root, indexDir string) (Stats, error) {
	c, err := Load(root)
	if err != nil {
		return Stats{}, err
	}
	if indexDir == "" {
		indexDir = DefaultIndexDir()
	}
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return Stats{}, err
	}
	src := filepath.Join(c.Root, "database", "network_genome_v0_2.sqlite")
	dst := filepath.Join(indexDir, "network_genome_v0_2.sqlite")
	if err := rebuildSQLiteFromCards(filepath.Join(c.Root, "cards", "failure_cards.json"), dst); err != nil {
		if err := copyFile(src, dst); err != nil {
			return Stats{}, err
		}
	}
	stats, err := IndexStats(c.Root, indexDir)
	if err != nil {
		return Stats{}, err
	}
	stats.SQLitePath = dst
	data, _ := json.MarshalIndent(stats, "", "  ")
	if err := os.WriteFile(filepath.Join(indexDir, "index_stats.json"), data, 0644); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func rebuildSQLiteFromCards(cardsPath, dst string) error {
	data, err := os.ReadFile(cardsPath)
	if err != nil {
		return err
	}
	var payload cardFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if len(payload.Cards) == 0 {
		return errors.New("cannot rebuild SQLite index from empty card corpus")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	db, err := sql.Open("sqlite", dst)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA journal_mode=DELETE; PRAGMA synchronous=NORMAL;`); err != nil {
		return err
	}
	schema := []string{
		`CREATE TABLE cards (id TEXT PRIMARY KEY, title TEXT NOT NULL, layer TEXT NOT NULL, domain TEXT NOT NULL, risk TEXT NOT NULL, json TEXT NOT NULL);`,
		`CREATE TABLE card_tags (card_id TEXT NOT NULL, tag TEXT NOT NULL);`,
		`CREATE TABLE card_layers (card_id TEXT NOT NULL, layer TEXT NOT NULL);`,
		`CREATE TABLE observations (card_id TEXT NOT NULL, value TEXT NOT NULL);`,
		`CREATE TABLE discriminators (card_id TEXT NOT NULL, value TEXT NOT NULL);`,
		`CREATE TABLE verifiers (card_id TEXT NOT NULL, value TEXT NOT NULL);`,
		`CREATE TABLE recipes (card_id TEXT NOT NULL, mode TEXT, automation_class TEXT, commands_json TEXT, manual_steps_json TEXT);`,
		`CREATE TABLE rollbacks (card_id TEXT NOT NULL, value TEXT NOT NULL);`,
		`CREATE TABLE edges (src TEXT NOT NULL, dst TEXT NOT NULL, type TEXT NOT NULL);`,
		`CREATE TABLE source_anchors (card_id TEXT NOT NULL, anchor TEXT NOT NULL);`,
		`CREATE VIRTUAL TABLE cards_fts USING fts5(id UNINDEXED, title, symptoms, observations, discriminators, likely_causes, tags);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, card := range payload.Cards {
		cardJSON, err := json.Marshal(card)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO cards VALUES (?,?,?,?,?,?)`, card.ID, card.Title, card.Layer, card.Domain, card.Risk, string(cardJSON)); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO card_layers VALUES (?,?)`, card.ID, card.Layer); err != nil {
			return err
		}
		if err := insertTextList(tx, `INSERT INTO card_tags VALUES (?,?)`, card.ID, card.Tags); err != nil {
			return err
		}
		if err := insertTextList(tx, `INSERT INTO observations VALUES (?,?)`, card.ID, card.Observations); err != nil {
			return err
		}
		if err := insertTextList(tx, `INSERT INTO discriminators VALUES (?,?)`, card.ID, card.Discriminators); err != nil {
			return err
		}
		if err := insertTextList(tx, `INSERT INTO verifiers VALUES (?,?)`, card.ID, card.Verify); err != nil {
			return err
		}
		commandsJSON, err := json.Marshal(card.Repair.Commands)
		if err != nil {
			return err
		}
		manualJSON, err := json.Marshal(card.Repair.ManualSteps)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO recipes VALUES (?,?,?,?,?)`, card.ID, card.Repair.Mode, card.Repair.AutomationClass, string(commandsJSON), string(manualJSON)); err != nil {
			return err
		}
		if err := insertTextList(tx, `INSERT INTO rollbacks VALUES (?,?)`, card.ID, card.Rollback); err != nil {
			return err
		}
		if err := insertTextList(tx, `INSERT INTO source_anchors VALUES (?,?)`, card.ID, card.SourceAnchors); err != nil {
			return err
		}
		for _, related := range card.Related {
			if _, err := tx.Exec(`INSERT INTO edges VALUES (?,?,?)`, card.ID, related, "related"); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`INSERT INTO cards_fts (id,title,symptoms,observations,discriminators,likely_causes,tags) VALUES (?,?,?,?,?,?,?)`,
			card.ID,
			card.Title,
			strings.Join(card.Symptoms, "\n"),
			strings.Join(card.Observations, "\n"),
			strings.Join(card.Discriminators, "\n"),
			strings.Join(card.LikelyCauses, "\n"),
			strings.Join(card.Tags, " "),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func insertTextList(tx *sql.Tx, stmt, cardID string, values []string) error {
	for _, value := range values {
		if _, err := tx.Exec(stmt, cardID, value); err != nil {
			return err
		}
	}
	return nil
}

func Explain(root, id string) (Card, error) {
	c, err := Load(root)
	if err != nil {
		return Card{}, err
	}
	card, ok := c.ByID[id]
	if !ok {
		return Card{}, fmt.Errorf("unknown card %s", id)
	}
	return card, nil
}

func Diagnose(root, symptom, snapshotPath string, top int) (Diagnosis, error) {
	if top <= 0 {
		top = 3
	}
	c, err := Load(root)
	if err != nil {
		return Diagnosis{}, err
	}
	features := map[string]string{}
	privacy := ""
	if snapshotPath != "" {
		sf, err := LoadFeatures(snapshotPath)
		if err != nil {
			return Diagnosis{}, err
		}
		features = sf.Features
		privacy = sf.PrivacyMode
	}
	routes := RouteSymptomForRoot(c.Root, symptom)
	type scored struct {
		card           Card
		score          int
		why            []string
		symptoms       []string
		discriminators []string
		features       []string
	}
	candidates := candidateCards(c, symptom, routes)
	var scoredCards []scored
	for _, card := range candidates.Cards {
		score, why, matchedSymptoms, matchedDisc, usedFeatures := scoreCard(card, symptom, routes, features)
		if score <= 0 {
			continue
		}
		if candidates.FTSIDs[card.ID] {
			why = append(why, "SQLite/FTS candidate retrieval selected this card")
		}
		scoredCards = append(scoredCards, scored{card: card, score: score, why: why, symptoms: matchedSymptoms, discriminators: matchedDisc, features: usedFeatures})
	}
	sort.SliceStable(scoredCards, func(i, j int) bool {
		if scoredCards[i].score == scoredCards[j].score {
			return scoredCards[i].card.ID < scoredCards[j].card.ID
		}
		return scoredCards[i].score > scoredCards[j].score
	})
	if len(scoredCards) > top {
		scoredCards = scoredCards[:top]
	}
	d := Diagnosis{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		ToolVersion:   ToolVersion,
		CorpusVersion: c.Manifest.Version,
		Symptom:       symptom,
		SnapshotPath:  snapshotPath,
		PrivacyMode:   privacy,
		Routes:        routes,
		Features:      features,
	}
	for i, s := range scoredCards {
		d.Hypotheses = append(d.Hypotheses, Hypothesis{
			Rank:                   i + 1,
			CardID:                 s.card.ID,
			Title:                  s.card.Title,
			Layer:                  s.card.Layer,
			Score:                  s.score,
			WhyMatched:             s.why,
			RelevantSymptoms:       s.symptoms,
			RelevantDiscriminators: s.discriminators,
			SnapshotFeaturesUsed:   s.features,
			ConfirmingObservations: firstN(s.card.Observations, 3),
			FalsifyingObservations: firstN(s.card.FalsePositives, 3),
			NextReadOnlyChecks:     firstN(s.card.SafeChecks, 3),
			RiskClass:              s.card.Risk,
			Verifier:               s.card.Verify,
			Rollback:               s.card.Rollback,
			RecipeGate:             GateCard(s.card),
		})
	}
	return d, nil
}

func RouteSymptom(symptom string) []RouteCandidate {
	return fallbackRouteSymptom(symptom)
}

func RouteSymptomForRoot(root, symptom string) []RouteCandidate {
	if routes, err := routeSymptomFromYAML(filepath.Join(root, "ontology", "symptom_routes.yaml"), symptom); err == nil && len(routes) > 0 {
		return routes
	}
	return fallbackRouteSymptom(symptom)
}

func fallbackRouteSymptom(symptom string) []RouteCandidate {
	s := strings.ToLower(symptom)
	routes := []struct {
		id        string
		needles   []string
		layers    []string
		domains   []string
		reason    string
		verifiers []string
	}{
		{"SYM-NO-INTERNET", []string{"no internet", "all network down", "cannot browse", "offline"}, []string{"L00_physical_link", "L01_interface_service", "L02_ip_dhcp", "L03_routing", "L04_dns", "L05_proxy", "L06_vpn_ne_tun", "L07_firewall_pf", "L13_external_provider"}, nil, "general internet outage symptom", []string{"ping gateway", "curl direct IP", "dig public domain"}},
		{"SYM-BROWSER-OK-CLI-FAIL", []string{"browser works", "terminal fails", "curl fails browser", "cli fail"}, []string{"L05_proxy", "L08_app_connectivity", "L09_tls_identity", "L04_dns", "L06_vpn_ne_tun"}, []string{"proxy_control_plane", "developer_connectivity"}, "browser/terminal split usually points at proxy, app, TLS, DNS, or VPN scope", []string{"scutil --proxy", "env proxy diff", "curl -Iv target"}},
		{"SYM-CODEX-API-FAIL", []string{"codex", "openai api", "local agent", "api unreachable"}, []string{"L08_app_connectivity", "L05_proxy", "L04_dns", "L09_tls_identity", "L06_vpn_ne_tun", "L13_external_provider"}, []string{"developer_connectivity"}, "agent/API symptom", []string{"curl target from same shell", "Codex config diff", "TLS issuer check"}},
		{"SYM-CLASH-TUN-BROKEN", []string{"clash", "verge", "mihomo", "tun", "fake-ip", "fake ip"}, []string{"L06_vpn_ne_tun", "L04_dns", "L03_routing", "L05_proxy"}, []string{"vpn_network_extension_tun", "proxy_control_plane"}, "Clash/Verge/mihomo TUN/DNS symptom", []string{"Clash core/listener status", "scutil --dns", "route -n get default"}},
		{"SYM-VPN-LAN-LOSS", []string{"vpn", "lan", "local network", "split tunnel"}, []string{"L06_vpn_ne_tun", "L03_routing", "L04_dns", "L10_multicast_discovery"}, []string{"vpn_network_extension_tun"}, "VPN plus LAN reachability symptom", []string{"route -n get LAN target", "scutil --dns", "ifconfig utun"}},
		{"SYM-DNS-IP-SPLIT", []string{"dns fails", "ip works", "resolution fails", "dig fail"}, []string{"L04_dns", "L05_proxy", "L06_vpn_ne_tun"}, []string{"dns_resolver"}, "DNS fails while IP path may work", []string{"dig public domain", "curl direct IP", "scutil --dns"}},
		{"SYM-LOCALHOST-PROXY-DEAD", []string{"localhost proxy", "127.0.0.1", "dead proxy", "proxy port"}, []string{"L05_proxy", "L08_app_connectivity"}, []string{"proxy_control_plane"}, "local proxy listener symptom", []string{"scutil --proxy", "lsof local proxy port"}},
		{"SYM-DANTE-INVISIBLE", []string{"dante", "devices invisible", "controller empty", "audio devices disappeared"}, []string{"L00_physical_link", "L01_interface_service", "L02_ip_dhcp", "L10_multicast_discovery", "L07_firewall_pf", "L11_ptp_clock"}, []string{"pro_audio_networking"}, "Dante discovery symptom; manual-only domain", []string{"expected NIC active", "same subnet/static IP", "controller interface selection"}},
		{"SYM-DANTE-NO-AUDIO", []string{"dante visible", "no audio", "muted", "dropouts"}, []string{"L11_ptp_clock", "L12_aoip_media", "L10_multicast_discovery"}, []string{"pro_audio_networking"}, "Dante visible but media/clock issue", []string{"Dante Controller Clock Status", "sample rate", "flow status"}},
		{"SYM-AES67-CLOCK-MEDIA", []string{"aes67", "ravenna", "ptp", "subscription but no sound", "stream silent"}, []string{"L10_multicast_discovery", "L11_ptp_clock", "L12_aoip_media", "L02_ip_dhcp", "L07_firewall_pf"}, []string{"pro_audio_networking"}, "AES67/RAVENNA clock/media symptom; manual-only domain", []string{"PTP lock/status", "SAP/SDP stream params", "IGMP/QoS switch state"}},
		{"SYM-DEV-TOOLS-CONNECTIVITY", []string{"git", "npm", "brew", "docker", "homebrew"}, []string{"L08_app_connectivity", "L05_proxy", "L04_dns", "L09_tls_identity"}, []string{"developer_connectivity"}, "developer tool connectivity symptom", []string{"tool proxy config", "same-shell curl", "registry URL check"}},
	}
	var out []RouteCandidate
	for _, r := range routes {
		for _, n := range r.needles {
			if strings.Contains(s, n) {
				out = append(out, RouteCandidate{ID: r.id, CandidateLayers: r.layers, CandidateDomain: r.domains, Reason: r.reason, FirstVerifiers: r.verifiers})
				break
			}
		}
	}
	if len(out) == 0 {
		out = append(out, RouteCandidate{ID: "SYM-GENERIC", CandidateLayers: []string{"L03_routing", "L04_dns", "L05_proxy", "L08_app_connectivity"}, Reason: "generic symptom fallback"})
	}
	return out
}

func routeSymptomFromYAML(path, symptom string) ([]RouteCandidate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	type yamlRoute struct {
		id        string
		phrases   []string
		layers    []string
		domains   []string
		verifiers []string
		boosts    []RouteBoost
	}
	var routes []yamlRoute
	current := -1
	currentBoost := -1
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- id:") {
			routes = append(routes, yamlRoute{id: strings.TrimSpace(strings.TrimPrefix(line, "- id:"))})
			current = len(routes) - 1
			currentBoost = -1
			section = ""
			continue
		}
		if current < 0 {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			currentBoost = -1
			continue
		}
		if section == "ranking_boosts" {
			if strings.HasPrefix(line, "- ") {
				boost := RouteBoost{}
				parseRouteBoostField(&boost, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
				routes[current].boosts = append(routes[current].boosts, boost)
				currentBoost = len(routes[current].boosts) - 1
				continue
			}
			if currentBoost >= 0 {
				parseRouteBoostField(&routes[current].boosts[currentBoost], line)
			}
			continue
		}
		if strings.HasPrefix(line, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			switch section {
			case "user_phrase":
				routes[current].phrases = append(routes[current].phrases, val)
			case "priority_layers":
				routes[current].layers = append(routes[current].layers, val)
			case "priority_domains":
				routes[current].domains = append(routes[current].domains, val)
			case "first_verifiers":
				routes[current].verifiers = append(routes[current].verifiers, val)
			}
		}
	}
	var out []RouteCandidate
	for _, r := range routes {
		for _, phrase := range r.phrases {
			if phraseMatches(symptom, phrase) {
				out = append(out, RouteCandidate{
					ID:              r.id,
					CandidateLayers: append([]string(nil), r.layers...),
					CandidateDomain: append([]string(nil), r.domains...),
					Reason:          "symptom_routes.yaml matched phrase: " + phrase,
					FirstVerifiers:  append([]string(nil), r.verifiers...),
					RankingBoosts:   append([]RouteBoost(nil), r.boosts...),
				})
				break
			}
		}
	}
	return out, nil
}

func parseRouteBoostField(boost *RouteBoost, line string) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.Trim(strings.TrimSpace(value), `"`)
	switch key {
	case "card_id", "cardId":
		boost.CardID = value
	case "id_prefix", "idPrefix":
		boost.IDPrefix = value
	case "layer":
		boost.Layer = value
	case "domain":
		boost.Domain = value
	case "score":
		var score int
		if _, err := fmt.Sscanf(value, "%d", &score); err == nil {
			boost.Score = score
		}
	case "reason":
		boost.Reason = value
	}
}

func phraseMatches(symptom, phrase string) bool {
	s := strings.ToLower(symptom)
	p := strings.ToLower(phrase)
	if strings.Contains(s, p) || strings.Contains(p, s) {
		return true
	}
	return overlap(symptom, phrase) >= 2
}

type candidateSelection struct {
	Cards  []Card
	FTSIDs map[string]bool
}

func candidateCards(c Corpus, symptom string, routes []RouteCandidate) candidateSelection {
	ids, err := ftsCandidateIDs(c.Root, symptom, 50)
	seen := map[string]bool{}
	ftsIDs := map[string]bool{}
	var out []Card
	if err == nil {
		for _, id := range ids {
			if card, ok := c.ByID[id]; ok && !seen[id] {
				out = append(out, card)
				seen[id] = true
				ftsIDs[id] = true
			}
		}
	}
	for _, route := range routes {
		for _, card := range c.Cards {
			if seen[card.ID] {
				continue
			}
			if stringIn(card.Layer, route.CandidateLayers) || stringIn(card.Domain, route.CandidateDomain) {
				out = append(out, card)
				seen[card.ID] = true
			}
			if len(out) >= 80 {
				return candidateSelection{Cards: out, FTSIDs: ftsIDs}
			}
		}
	}
	if len(out) == 0 {
		return candidateSelection{Cards: c.Cards, FTSIDs: ftsIDs}
	}
	return candidateSelection{Cards: out, FTSIDs: ftsIDs}
}

func ftsCandidateIDs(root, symptom string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	tokens := tokenize(symptom)
	var queryTerms []string
	for _, t := range tokens {
		if len(t) >= 3 {
			queryTerms = append(queryTerms, t)
		}
	}
	if len(queryTerms) == 0 {
		return nil, nil
	}
	indexDir := DefaultIndexDir()
	sqlitePath := filepath.Join(indexDir, "network_genome_v0_2.sqlite")
	if !sqliteUsable(sqlitePath) {
		if _, err := RebuildIndex(root, indexDir); err != nil {
			return nil, err
		}
	}
	ids, err := queryFTS(sqlitePath, queryTerms, limit)
	if err == nil {
		return ids, nil
	}
	if _, rebuildErr := RebuildIndex(root, indexDir); rebuildErr != nil {
		return nil, err
	}
	return queryFTS(sqlitePath, queryTerms, limit)
}

func sqliteUsable(path string) bool {
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return false
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return false
	}
	defer db.Close()
	var cards, fts int
	if err := db.QueryRow(`select count(*) from cards`).Scan(&cards); err != nil {
		return false
	}
	if err := db.QueryRow(`select count(*) from cards_fts`).Scan(&fts); err != nil {
		return false
	}
	return cards > 0 && fts > 0
}

type sqliteIndexStats struct {
	Cards        int  `json:"cards"`
	FTSRows      int  `json:"ftsRows"`
	FTSIDsUsable bool `json:"ftsIdsUsable"`
	FTSNullIDs   int  `json:"ftsNullIds"`
}

func inspectSQLiteStats(path string) (sqliteIndexStats, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return sqliteIndexStats{}, err
	}
	defer db.Close()
	stats := sqliteIndexStats{}
	if err := db.QueryRow(`select count(*) from cards`).Scan(&stats.Cards); err != nil {
		return sqliteIndexStats{}, err
	}
	if err := db.QueryRow(`select count(*) from cards_fts`).Scan(&stats.FTSRows); err != nil {
		return sqliteIndexStats{}, err
	}
	if err := db.QueryRow(`select count(*) from cards_fts where id is null or id = ''`).Scan(&stats.FTSNullIDs); err != nil {
		return sqliteIndexStats{}, err
	}
	stats.FTSIDsUsable = stats.FTSRows > 0 && stats.FTSNullIDs == 0
	return stats, nil
}

func queryFTS(sqlitePath string, terms []string, limit int) ([]string, error) {
	query := strings.Join(terms, " OR ")
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`select id from cards_fts where cards_fts match ? order by bm25(cards_fts) limit ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func scoreCard(card Card, symptom string, routes []RouteCandidate, features map[string]string) (int, []string, []string, []string, []string) {
	score := 0
	var why, matchedSymptoms, matchedDisc, usedFeatures []string
	textTokens := tokenize(symptom)
	cardText := strings.ToLower(strings.Join(append(append(append([]string{card.ID, card.Title, card.Layer, card.Domain, card.Risk}, card.Tags...), card.Symptoms...), append(card.Observations, card.Discriminators...)...), " "))
	for _, t := range textTokens {
		if len(t) < 3 {
			continue
		}
		if strings.Contains(cardText, t) {
			score += 2
		}
	}
	for _, s := range card.Symptoms {
		if overlap(symptom, s) >= 2 || containsAnyFold(symptom, s) {
			score += 8
			matchedSymptoms = append(matchedSymptoms, s)
		}
	}
	for _, d := range card.Discriminators {
		if overlap(symptom, d) >= 2 || containsAnyFold(symptom, d) {
			score += 10
			matchedDisc = append(matchedDisc, d)
		}
	}
	for _, r := range routes {
		if stringIn(card.Layer, r.CandidateLayers) {
			score += 8
			why = append(why, "route "+r.ID+" prioritizes "+card.Layer)
		}
		if len(r.CandidateDomain) > 0 && stringIn(card.Domain, r.CandidateDomain) {
			score += 6
			why = append(why, "route "+r.ID+" prioritizes "+card.Domain)
		}
	}
	boostScore, boostWhy := routeRankingBoost(card, routes)
	score += boostScore
	why = append(why, boostWhy...)
	for key, val := range features {
		if val != "true" {
			continue
		}
		terms := strings.Split(strings.ReplaceAll(key, "_", " "), " ")
		hits := 0
		for _, term := range terms {
			if len(term) >= 3 && strings.Contains(cardText, term) {
				hits++
			}
		}
		if hits > 0 {
			score += hits * 3
			usedFeatures = append(usedFeatures, key+"="+val)
		}
	}
	if card.Risk == "never_auto" {
		score -= 2
	}
	if card.Risk == "high" {
		score -= 1
	}
	if len(matchedSymptoms) > 0 {
		why = append(why, "symptom text matched card symptoms")
	}
	if len(matchedDisc) > 0 {
		why = append(why, "symptom text matched discriminators")
	}
	if len(usedFeatures) > 0 {
		why = append(why, "snapshot features matched card observations")
	}
	return score, uniqueStrings(why), firstN(uniqueStrings(matchedSymptoms), 3), firstN(uniqueStrings(matchedDisc), 3), firstN(uniqueStrings(usedFeatures), 6)
}

func routeRankingBoost(card Card, routes []RouteCandidate) (int, []string) {
	score := 0
	var why []string
	for _, route := range routes {
		for _, boost := range route.RankingBoosts {
			if boost.Score == 0 || !routeBoostMatches(card, boost) {
				continue
			}
			score += boost.Score
			reason := boost.Reason
			if reason == "" {
				reason = routeBoostTarget(boost)
			}
			why = append(why, fmt.Sprintf("route %s corpus ranking boost: %s", route.ID, reason))
		}
	}
	return score, why
}

func routeBoostMatches(card Card, boost RouteBoost) bool {
	switch {
	case boost.CardID != "":
		return card.ID == boost.CardID
	case boost.IDPrefix != "":
		return strings.HasPrefix(card.ID, boost.IDPrefix)
	case boost.Layer != "":
		return card.Layer == boost.Layer
	case boost.Domain != "":
		return card.Domain == boost.Domain
	default:
		return false
	}
}

func routeBoostTarget(boost RouteBoost) string {
	switch {
	case boost.CardID != "":
		return "card " + boost.CardID
	case boost.IDPrefix != "":
		return "cards with prefix " + boost.IDPrefix
	case boost.Layer != "":
		return "layer " + boost.Layer
	case boost.Domain != "":
		return "domain " + boost.Domain
	default:
		return "unspecified target"
	}
}

func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	return uniqueStrings(fields)
}

func overlap(a, b string) int {
	at := map[string]bool{}
	for _, t := range tokenize(a) {
		at[t] = true
	}
	count := 0
	for _, t := range tokenize(b) {
		if at[t] {
			count++
		}
	}
	return count
}

func containsAnyFold(a, b string) bool {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	return strings.Contains(a, b) || strings.Contains(b, a)
}

func Gate(root, id string) (GateResult, error) {
	card, err := Explain(root, id)
	if err != nil {
		return GateResult{}, err
	}
	return GateCard(card), nil
}

func GateCard(card Card) GateResult {
	g := GateResult{
		CardID:          card.ID,
		Title:           card.Title,
		Layer:           card.Layer,
		RiskClass:       card.Risk,
		RecipeType:      firstNonEmpty(card.Repair.AutomationClass, card.Repair.Mode),
		VerifierPresent: len(card.Verify) > 0,
		RollbackPresent: len(card.Rollback) > 0,
	}
	mutating := hasMutatingCommand(card)
	allText := strings.ToLower(strings.Join([]string{
		card.ID, card.Title, card.Layer, card.Domain, strings.Join(card.Tags, " "),
		strings.Join(card.Symptoms, " "), strings.Join(card.Observations, " "),
		strings.Join(card.Discriminators, " "), strings.Join(card.LikelyCauses, " "),
		strings.Join(card.Repair.Commands, " "), strings.Join(card.Repair.ManualSteps, " "),
	}, " "))
	switch {
	case !g.VerifierPresent:
		g.Result = "blocked"
		g.Reason = "no verifier is defined for this card"
	case mutating && !g.RollbackPresent:
		g.Result = "blocked"
		g.Reason = "mutating repair has no rollback"
	case strings.Contains(allText, "tls") && strings.Contains(allText, "bypass"):
		g.Result = "blocked"
		g.Reason = "TLS verification bypass is blocked as a durable repair"
	case strings.Contains(allText, "pf flush") || strings.Contains(allText, "global pf flush"):
		g.Result = "blocked"
		g.Reason = "global pf flush is blocked"
	case card.Layer == "L14_user_policy" || strings.Contains(strings.ToLower(card.Title), "mdm") || strings.Contains(strings.ToLower(card.Title), "corporate") || strings.Contains(strings.ToLower(card.Domain), "policy") || strings.Contains(allText, "corporate policy") || strings.Contains(allText, "managed profile") || strings.Contains(allText, "edr policy"):
		g.Result = "report_only"
		g.Reason = "policy/MDM/corporate network issue requires report/admin path"
		g.AllowedNextStep = "generate support report; do not mutate local policy settings"
	case strings.EqualFold(card.Repair.AutomationClass, "never_auto"):
		g.Result = "manual_only"
		g.Reason = "repair automation class never_auto forbids automatic execution"
		g.AllowedNextStep = "manual checklist/report only"
	case card.Risk == "never_auto":
		g.Result = "manual_only"
		g.Reason = "risk class never_auto forbids automatic execution"
		g.AllowedNextStep = "manual checklist/report only"
	case card.Risk == "high":
		g.Result = "manual_only"
		g.Reason = "high-risk network repair is manual-only in v0.5"
		g.AllowedNextStep = "read-only checks and human-confirmed plan only"
	case card.Layer == "L10_multicast_discovery" || card.Layer == "L11_ptp_clock" || card.Layer == "L12_aoip_media":
		g.Result = "manual_only"
		g.Reason = "AoIP/multicast/PTP/media layers are manual-only unless explicitly read-only"
		g.AllowedNextStep = "read-only observation and operator checklist"
	case containsProtectedMutationDomain(allText):
		g.Result = "manual_only"
		g.Reason = "Dante/AES67/RAVENNA/PTP/switch/QoS/VLAN changes are protected manual domains"
		g.AllowedNextStep = "report/checklist; do not auto-change topology"
	case card.Risk == "read_only" || !mutating:
		g.Result = "read_only_allowed"
		g.Reason = "card uses read-only checks/manual observation only"
		g.AllowedNextStep = firstOf(card.SafeChecks)
	case card.Risk == "low" || card.Risk == "medium":
		g.Result = "human_confirmed_allowed"
		g.Reason = "reversible local repair has verifier and rollback; requires explicit human confirmation"
		g.AllowedNextStep = "show dry-run and require explicit approval before mutation"
	default:
		g.Result = "blocked"
		g.Reason = "unknown mutation or unsupported risk class"
	}
	return g
}

func hasMutatingCommand(card Card) bool {
	for _, cmd := range card.Repair.Commands {
		s := strings.TrimSpace(cmd)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		return true
	}
	return false
}

func containsProtectedMutationDomain(text string) bool {
	for _, term := range []string{"dante", "aes67", "ravenna", "ptp", "switch", "qos", "vlan", "clock leader", "sample rate"} {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func Collect(ctx context.Context, out, privacy string) (SnapshotManifest, error) {
	if out == "" {
		return SnapshotManifest{}, errors.New("collect requires --out")
	}
	if privacy == "" {
		privacy = "standard"
	}
	rawDir := filepath.Join(out, "raw")
	redactedDir := filepath.Join(out, "redacted")
	featuresDir := filepath.Join(out, "features")
	for _, dir := range []string{rawDir, redactedDir, featuresDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return SnapshotManifest{}, err
		}
	}
	commands := map[string][]string{
		"system_uname.txt":           {"uname", "-a"},
		"system_sw_vers.txt":         {"sw_vers"},
		"host_hostname.txt":          {"hostname"},
		"network_hardware_ports.txt": {"networksetup", "-listallhardwareports"},
		"network_service_order.txt":  {"networksetup", "-listnetworkserviceorder"},
		"network_ifconfig.txt":       {"ifconfig"},
		"route_default.txt":          {"route", "-n", "get", "default"},
		"dns_scutil.txt":             {"scutil", "--dns"},
		"proxy_scutil.txt":           {"scutil", "--proxy"},
		"launchctl_http_proxy.txt":   {"launchctl", "getenv", "HTTP_PROXY"},
		"launchctl_https_proxy.txt":  {"launchctl", "getenv", "HTTPS_PROXY"},
		"git_proxy.txt":              {"git", "config", "--global", "--get", "http.proxy"},
		"npm_proxy.txt":              {"npm", "config", "get", "proxy"},
		"npm_registry.txt":           {"npm", "config", "get", "registry"},
	}
	warnings := []string{}
	allowUnredactedRaw := os.Getenv("AGENTLINK_ALLOW_UNREDACTED_RAW") == "1"
	if !allowUnredactedRaw {
		warnings = append(warnings, "raw directory is redacted by default; set AGENTLINK_ALLOW_UNREDACTED_RAW=1 only for explicit local debug archives")
	}
	for name, argv := range commands {
		text, err := runReadOnly(ctx, argv)
		if err != nil {
			text = "unavailable: " + err.Error() + "\n"
			warnings = append(warnings, name+": "+err.Error())
		}
		redactedText := Redact(text, privacy)
		if privacy == "strict" && name == "host_hostname.txt" && strings.TrimSpace(text) != "" && !strings.HasPrefix(text, "unavailable:") {
			redactedText = "<hostname>\n"
		}
		rawText := redactedText
		if allowUnredactedRaw {
			rawText = text
			if name == "docker_config_hint.txt" {
				rawText = redactedText
			}
		}
		if err := os.WriteFile(filepath.Join(rawDir, name), []byte(rawText), 0600); err != nil {
			return SnapshotManifest{}, err
		}
		if err := os.WriteFile(filepath.Join(redactedDir, name), []byte(redactedText), 0600); err != nil {
			return SnapshotManifest{}, err
		}
	}
	dockerText := dockerConfigSummary()
	if err := os.WriteFile(filepath.Join(rawDir, "docker_config_hint.txt"), []byte(dockerText), 0600); err != nil {
		return SnapshotManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(redactedDir, "docker_config_hint.txt"), []byte(dockerText), 0600); err != nil {
		return SnapshotManifest{}, err
	}
	envText := proxyEnvText()
	redactedEnv := Redact(envText, privacy)
	rawEnv := redactedEnv
	if allowUnredactedRaw {
		rawEnv = envText
	}
	_ = os.WriteFile(filepath.Join(rawDir, "shell_proxy_env.txt"), []byte(rawEnv), 0600)
	_ = os.WriteFile(filepath.Join(redactedDir, "shell_proxy_env.txt"), []byte(redactedEnv), 0600)
	featuresPath := filepath.Join(featuresDir, "snapshot_features.json")
	manifest := SnapshotManifest{
		SchemaVersion: 1,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		ToolVersion:   ToolVersion,
		PrivacyMode:   privacy,
		RawDir:        rawDir,
		RedactedDir:   redactedDir,
		FeaturesPath:  featuresPath,
		Redacted:      true,
		Warnings:      warnings,
	}
	if data, err := os.ReadFile(filepath.Join(redactedDir, "system_uname.txt")); err == nil {
		manifest.HostOS = strings.TrimSpace(string(data))
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "snapshot_manifest.json"), manifestData, 0600); err != nil {
		return SnapshotManifest{}, err
	}
	sf, err := ExtractFeatures(out)
	if err != nil {
		return SnapshotManifest{}, err
	}
	featuresData, _ := json.MarshalIndent(sf, "", "  ")
	if err := os.WriteFile(featuresPath, featuresData, 0600); err != nil {
		return SnapshotManifest{}, err
	}
	return manifest, nil
}

func dockerConfigSummary() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "docker_config_present=false\n"
	}
	path := filepath.Join(home, ".docker", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "docker_config_present=false\n"
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "docker_config_present=true\nparse_error=true\n"
	}
	auths, _ := cfg["auths"].(map[string]any)
	credsStore := false
	if _, ok := cfg["credsStore"]; ok {
		credsStore = true
	}
	credHelpers := false
	if _, ok := cfg["credHelpers"]; ok {
		credHelpers = true
	}
	proxyPresent := false
	if proxies, ok := cfg["proxies"]; ok && proxies != nil {
		proxyPresent = true
	}
	identityToken := false
	for _, v := range auths {
		if m, ok := v.(map[string]any); ok {
			if _, ok := m["identitytoken"]; ok {
				identityToken = true
			}
			if _, ok := m["identityToken"]; ok {
				identityToken = true
			}
		}
	}
	summary := map[string]any{
		"docker_config_present":       true,
		"docker_proxy_config_present": proxyPresent,
		"docker_auths_present":        len(auths) > 0,
		"docker_registry_count":       len(auths),
		"docker_creds_store_present":  credsStore,
		"docker_cred_helpers_present": credHelpers,
		"docker_has_identitytoken":    identityToken,
	}
	out, _ := json.MarshalIndent(summary, "", "  ")
	return string(out) + "\n"
}

func runReadOnly(ctx context.Context, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("empty command")
	}
	if _, err := exec.LookPath(argv[0]); err != nil && !strings.Contains(argv[0], "/") {
		return "", err
	}
	callCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(callCtx, argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
	if callCtx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("%s timed out", argv[0])
	}
	if err != nil && len(out) == 0 {
		return "", err
	}
	return string(out), nil
}

func proxyEnvText() string {
	var lines []string
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"} {
		if val := os.Getenv(key); val != "" {
			lines = append(lines, key+"="+val)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

var redactionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(api[_-]?key\s*[=:]\s*)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)(password\s*[=:]\s*)[^\\s]+`),
	regexp.MustCompile(`(?i)(token\s*[=:]\s*)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)("(?:auth|identitytoken|identityToken|credsStore|credHelpers|username|password)"\s*:\s*)("[^"]*"|\{[^}]*\})`),
	regexp.MustCompile(`(?i)(cookie:\s*)[^\n]+`),
	regexp.MustCompile(`(?i)(https?://)[^:/@\s]+:[^/@\s]+@`),
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{12,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{12,}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`),
}

func Redact(text, privacy string) string {
	out := text
	for _, re := range redactionPatterns {
		out = re.ReplaceAllStringFunc(out, func(match string) string {
			if strings.HasPrefix(strings.ToLower(match), "http://") || strings.HasPrefix(strings.ToLower(match), "https://") {
				return regexp.MustCompile(`(?i)^(https?://)`).FindString(match) + "<redacted>:<redacted>@"
			}
			parts := re.FindStringSubmatch(match)
			if len(parts) > 1 {
				return parts[1] + "<redacted>"
			}
			return "<redacted>"
		})
	}
	if privacy == "strict" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			out = strings.ReplaceAll(out, home, "/Users/<user>/<path>")
		}
		out = regexp.MustCompile(`/Users/[^/\s"',}]+(?:/[^\s"',}]*)?`).ReplaceAllString(out, "/Users/<user>/<path>")
		if u := os.Getenv("USER"); u != "" {
			out = strings.ReplaceAll(out, u, "<user>")
		}
		if host, err := os.Hostname(); err == nil && host != "" {
			out = strings.ReplaceAll(out, host, "<hostname>")
		}
		if host := os.Getenv("HOSTNAME"); host != "" {
			out = strings.ReplaceAll(out, host, "<hostname>")
		}
		out = regexp.MustCompile(`\b10(?:\.\d{1,3}){3}\b`).ReplaceAllString(out, "<private-ip>")
		out = regexp.MustCompile(`\b192\.168(?:\.\d{1,3}){2}\b`).ReplaceAllString(out, "<private-ip>")
		out = regexp.MustCompile(`\b172\.(?:1[6-9]|2\d|3[01])(?:\.\d{1,3}){2}\b`).ReplaceAllString(out, "<private-ip>")
		out = regexp.MustCompile(`\b169\.254(?:\.\d{1,3}){2}\b`).ReplaceAllString(out, "<link-local-ip>")
		out = regexp.MustCompile(`\b198\.18(?:\.\d{1,3}){2}\b`).ReplaceAllString(out, "<fake-ip-range>")
		out = regexp.MustCompile(`(?i)\b[A-Za-z0-9_-]+\.local\b`).ReplaceAllString(out, "<hostname>.local")
		out = regexp.MustCompile(`(?im)^(\s*(?:host(?:name)?|computername|localhostname)\s*[:=]?\s*)[A-Za-z0-9._-]+`).ReplaceAllString(out, `${1}<hostname>`)
	}
	return out
}

func ExtractFeatures(snapshot string) (SnapshotFeatures, error) {
	redactedDir := filepath.Join(snapshot, "redacted")
	entries, err := os.ReadDir(redactedDir)
	if err != nil {
		return SnapshotFeatures{}, err
	}
	chunks := map[string]string{}
	var all strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(redactedDir, e.Name()))
		if err != nil {
			return SnapshotFeatures{}, err
		}
		text := string(data)
		chunks[e.Name()] = text
		all.WriteString("\n--- " + e.Name() + " ---\n" + text)
	}
	text := strings.ToLower(all.String())
	features := map[string]string{}
	evidence := map[string]string{}
	set := func(key string, val bool, ev string) {
		if val {
			features[key] = "true"
			evidence[key] = ev
		} else {
			features[key] = "false"
		}
	}
	systemProxyText := strings.ToLower(chunks["proxy_scutil.txt"])
	shellProxyText := strings.ToLower(chunks["shell_proxy_env.txt"])
	effectiveShellProxyText := effectiveProxyEnvText(shellProxyText)
	proxyText := systemProxyText + "\n" + shellProxyText
	set("system_proxy_enabled", strings.Contains(systemProxyText, "enabled : 1") || strings.Contains(systemProxyText, "httpenable : 1") || strings.Contains(systemProxyText, "httpsenable : 1"), "scutil proxy enabled")
	set("system_proxy_localhost", (strings.Contains(systemProxyText, "127.0.0.1") || strings.Contains(systemProxyText, "localhost")) && features["system_proxy_enabled"] == "true", "system proxy points to localhost")
	set("shell_proxy_env_present", effectiveShellProxyText != "", "proxy env var present")
	set("shell_proxy_localhost", strings.Contains(effectiveShellProxyText, "127.0.0.1") || strings.Contains(effectiveShellProxyText, "localhost"), "shell proxy points to localhost")
	set("system_shell_proxy_mismatch", features["system_proxy_enabled"] != features["shell_proxy_env_present"], "system and shell proxy differ")
	set("pac_configured", strings.Contains(proxyText, "proxyautoconfigenable : 1") || strings.Contains(proxyText, ".pac"), "PAC configured")
	set("no_proxy_may_bypass_target", strings.Contains(proxyText, "no_proxy") || strings.Contains(proxyText, "exceptionslist"), "NO_PROXY or bypass list present")
	dnsText := strings.ToLower(chunks["dns_scutil.txt"])
	set("dns_resolver_mismatch", strings.Contains(dnsText, "scoped queries") || strings.Contains(dnsText, "resolver #"), "scoped/multiple resolvers observed")
	set("dns_resolution_failure_hint", strings.Contains(text, "dns") && strings.Contains(text, "fail"), "DNS failure text observed")
	set("ip_reachability_hint", strings.Contains(text, "1.1.1.1") || strings.Contains(text, "8.8.8.8"), "raw IP probe/hint observed")
	routeText := strings.ToLower(chunks["route_default.txt"])
	features["default_route_interface"] = defaultRouteInterface(routeText)
	evidence["default_route_interface"] = "route -n get default"
	set("default_route_through_utun", strings.Contains(routeText, "interface: utun") || strings.Contains(routeText, "utun"), "default route uses utun")
	set("utun_present", strings.Contains(strings.ToLower(chunks["network_ifconfig.txt"]), "utun"), "utun interface present")
	set("vpn_present_hint", strings.Contains(text, "vpn") || strings.Contains(text, "network extension") || features["utun_present"] == "true", "VPN/TUN hint present")
	set("lan_route_missing_under_vpn_hint", features["default_route_through_utun"] == "true" && strings.Contains(text, "lan"), "LAN route may be missing under VPN")
	set("clash_verge_mihomo_present_hint", strings.Contains(text, "clash") || strings.Contains(text, "verge") || strings.Contains(text, "mihomo"), "Clash/Verge/mihomo hint")
	set("tun_dns_route_risk_hint", features["utun_present"] == "true" && (features["dns_resolver_mismatch"] == "true" || features["default_route_through_utun"] == "true"), "TUN plus DNS/route risk")
	set("git_proxy_override_present", strings.TrimSpace(chunks["git_proxy.txt"]) != "" && !strings.Contains(chunks["git_proxy.txt"], "unavailable"), "git proxy config present")
	set("npm_proxy_override_present", strings.Contains(strings.ToLower(chunks["npm_proxy.txt"]), "http"), "npm proxy config present")
	set("npm_registry_override_present", strings.Contains(strings.ToLower(chunks["npm_registry.txt"]), "registry") || strings.Contains(strings.ToLower(chunks["npm_registry.txt"]), "http"), "npm registry configured")
	set("brew_proxy_env_present", strings.Contains(text, "homebrew") && strings.Contains(proxyText, "proxy"), "brew proxy environment hint")
	set("docker_proxy_config_present", dockerSummaryBool(chunks["docker_config_hint.txt"], "docker_proxy_config_present"), "docker proxy config present")
	set("docker_host_container_proxy_mismatch_risk", features["docker_proxy_config_present"] == "true" && features["shell_proxy_env_present"] == "true", "docker and host proxy both configured")
	set("tls_trust_issue_hint", strings.Contains(text, "certificate") || strings.Contains(text, "tls") || strings.Contains(text, "ssl"), "TLS/certificate hint")
	set("aoip_domain_requested", strings.Contains(text, "dante") || strings.Contains(text, "aes67") || strings.Contains(text, "ravenna"), "AoIP text present")
	set("dante_manual_only_domain", strings.Contains(text, "dante"), "Dante manual-only domain")
	set("aes67_manual_only_domain", strings.Contains(text, "aes67") || strings.Contains(text, "ravenna"), "AES67/RAVENNA manual-only domain")
	set("system_proxy_dead_local_port", features["system_proxy_localhost"] == "true" && strings.Contains(text, "connection refused"), "localhost proxy refused")
	set("shell_proxy_dead_local_port", features["shell_proxy_localhost"] == "true" && strings.Contains(text, "connection refused"), "shell localhost proxy refused")
	return SnapshotFeatures{SchemaVersion: 1, CreatedAt: time.Now().UTC().Format(time.RFC3339), PrivacyMode: manifestPrivacy(snapshot), Features: features, Evidence: evidence}, nil
}

func dockerSummaryBool(text, key string) bool {
	var summary map[string]any
	if err := json.Unmarshal([]byte(text), &summary); err != nil {
		return false
	}
	if v, ok := summary[key].(bool); ok {
		return v
	}
	return false
}

func effectiveProxyEnvText(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "http_proxy", "https_proxy", "all_proxy":
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func defaultRouteInterface(routeText string) string {
	for _, line := range strings.Split(routeText, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}
	return "unknown"
}

func manifestPrivacy(snapshot string) string {
	data, err := os.ReadFile(filepath.Join(snapshot, "snapshot_manifest.json"))
	if err != nil {
		return ""
	}
	var m SnapshotManifest
	_ = json.Unmarshal(data, &m)
	return m.PrivacyMode
}

func LoadFeatures(snapshot string) (SnapshotFeatures, error) {
	path := filepath.Join(snapshot, "features", "snapshot_features.json")
	data, err := os.ReadFile(path)
	if err != nil {
		sf, err := ExtractFeatures(snapshot)
		if err != nil {
			return SnapshotFeatures{}, err
		}
		data, _ := json.MarshalIndent(sf, "", "  ")
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		_ = os.WriteFile(path, data, 0600)
		return sf, nil
	}
	var sf SnapshotFeatures
	if err := json.Unmarshal(data, &sf); err != nil {
		return SnapshotFeatures{}, err
	}
	return sf, nil
}

func WriteReport(root, format, snapshot, symptom, out string) (string, error) {
	if format == "" {
		format = "markdown"
	}
	d, err := Diagnose(root, symptom, snapshot, 3)
	if err != nil {
		return "", err
	}
	rep := Report{
		Timestamp:            d.Timestamp,
		ToolVersion:          d.ToolVersion,
		CorpusVersion:        d.CorpusVersion,
		Snapshot:             map[string]any{"path": snapshot, "manifest": filepath.Join(snapshot, "snapshot_manifest.json")},
		Redaction:            map[string]any{"status": "redacted", "privacyMode": d.PrivacyMode},
		Symptom:              symptom,
		Features:             d.Features,
		Routes:               d.Routes,
		Hypotheses:           d.Hypotheses,
		RemainingUncertainty: []string{"Snapshot features are read-only observations; confirm with listed checks before repair.", "Manual-only domains require operator review."},
	}
	var data []byte
	switch format {
	case "json":
		data, err = json.MarshalIndent(rep, "", "  ")
	case "markdown", "md":
		data = []byte(markdownReport(rep))
	default:
		return "", fmt.Errorf("unsupported report format %s", format)
	}
	if err != nil {
		return "", err
	}
	privacy := d.PrivacyMode
	if privacy == "" {
		privacy = "standard"
	}
	data = []byte(Redact(string(data), privacy))
	if out == "" {
		return string(data), nil
	}
	if err := os.WriteFile(out, data, 0600); err != nil {
		return "", err
	}
	return out, nil
}

func markdownReport(rep Report) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# AgentLink Rescue Report")
	fmt.Fprintf(&b, "\n- timestamp: %s\n- tool version: %s\n- corpus version: %s\n- snapshot path: %v\n- redaction status: %v\n- privacy mode: %v\n- user symptom: %s\n", rep.Timestamp, rep.ToolVersion, rep.CorpusVersion, rep.Snapshot["path"], rep.Redaction["status"], rep.Redaction["privacyMode"], rep.Symptom)
	fmt.Fprintln(&b, "\n## Extracted Observations / Features")
	for _, k := range sortedKeys(rep.Features) {
		fmt.Fprintf(&b, "- `%s`: `%s`\n", k, rep.Features[k])
	}
	fmt.Fprintln(&b, "\n## Route Candidates")
	for _, r := range rep.Routes {
		fmt.Fprintf(&b, "- `%s`: %s; layers=%s\n", r.ID, r.Reason, strings.Join(r.CandidateLayers, ", "))
	}
	fmt.Fprintln(&b, "\n## Top Hypotheses")
	for _, h := range rep.Hypotheses {
		fmt.Fprintf(&b, "\n### %d. %s — %s\n", h.Rank, h.CardID, h.Title)
		fmt.Fprintf(&b, "- layer: `%s`\n- score: `%d`\n- risk: `%s`\n- why: %s\n", h.Layer, h.Score, h.RiskClass, strings.Join(h.WhyMatched, "; "))
		fmt.Fprintf(&b, "- next safest read-only checks: %s\n", strings.Join(h.NextReadOnlyChecks, "; "))
		fmt.Fprintf(&b, "- verifier: %s\n", strings.Join(h.Verifier, "; "))
		fmt.Fprintf(&b, "- rollback: %s\n", strings.Join(h.Rollback, "; "))
		fmt.Fprintf(&b, "- recipe gate: `%s` — %s\n", h.RecipeGate.Result, h.RecipeGate.Reason)
	}
	fmt.Fprintln(&b, "\n## Blocked / Manual-Only Actions")
	for _, h := range rep.Hypotheses {
		if h.RecipeGate.Result == "blocked" || h.RecipeGate.Result == "manual_only" || h.RecipeGate.Result == "report_only" {
			fmt.Fprintf(&b, "- `%s`: `%s` — %s\n", h.CardID, h.RecipeGate.Result, h.RecipeGate.Reason)
		}
	}
	fmt.Fprintln(&b, "\n## Remaining Uncertainty")
	for _, u := range rep.RemainingUncertainty {
		fmt.Fprintf(&b, "- %s\n", u)
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func FileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func firstN(in []string, n int) []string {
	if len(in) <= n {
		return append([]string{}, in...)
	}
	return append([]string{}, in[:n]...)
}

func firstOf(in []string) string {
	if len(in) == 0 {
		return ""
	}
	return in[0]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func stringIn(v string, values []string) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func WalkCorpusFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
