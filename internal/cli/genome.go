package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"cactus-agentlink-rescue/internal/genome"
)

func runGenomeIndex(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "index requires rebuild or stats")
		return 50
	}
	switch args[0] {
	case "rebuild":
		fs := flag.NewFlagSet("index rebuild", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		root := fs.String("root", "", "genome corpus root")
		indexDir := fs.String("index-dir", "", "index output dir")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		stats, err := genome.RebuildIndex(*root, *indexDir)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		return printGenomeStats(stats, *jsonOut, stdout)
	case "stats":
		fs := flag.NewFlagSet("index stats", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		root := fs.String("root", "", "genome corpus root")
		indexDir := fs.String("index-dir", "", "index dir")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		stats, err := genome.IndexStats(*root, *indexDir)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		return printGenomeStats(stats, *jsonOut, stdout)
	default:
		fmt.Fprintf(stderr, "unknown index command: %s\n", args[0])
		return 50
	}
}

func printGenomeStats(stats genome.Stats, jsonOut bool, stdout io.Writer) int {
	if jsonOut {
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Corpus version: %s\nCards: %d\nLayers: %d\nFTS rows: %d\nFTS IDs usable: %t\nFTS null IDs: %d\nGraph edges: %d\nSQLite: %s\n", stats.CorpusVersion, stats.Cards, stats.Layers, stats.FTSRows, stats.FTSIDsUsable, stats.FTSNullIDs, stats.GraphEdges, stats.SQLitePath)
	return 0
}

func runGenomeExplain(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cardID := fs.String("card", "", "card ID")
	jsonOut := fs.Bool("json", false, "print JSON")
	root := fs.String("root", "", "genome corpus root")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *cardID == "" {
		fmt.Fprintln(stderr, "explain requires --card")
		return 50
	}
	card, err := genome.Explain(*root, *cardID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(card, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	printCard(card, stdout)
	return 0
}

func printCard(card genome.Card, stdout io.Writer) {
	fmt.Fprintf(stdout, "%s — %s\n", card.ID, card.Title)
	fmt.Fprintf(stdout, "Layer: %s\nDomain: %s\nRisk: %s\n", card.Layer, card.Domain, card.Risk)
	fmt.Fprintf(stdout, "Symptoms:\n%s", bullet(card.Symptoms))
	fmt.Fprintf(stdout, "Observations:\n%s", bullet(card.Observations))
	fmt.Fprintf(stdout, "Discriminators:\n%s", bullet(card.Discriminators))
	fmt.Fprintf(stdout, "Likely causes:\n%s", bullet(card.LikelyCauses))
	fmt.Fprintf(stdout, "Safe checks:\n%s", bullet(card.SafeChecks))
	fmt.Fprintf(stdout, "Repair mode: %s / %s\n", card.Repair.Mode, card.Repair.AutomationClass)
	if len(card.Repair.Commands) > 0 {
		fmt.Fprintf(stdout, "Repair commands:\n%s", bullet(card.Repair.Commands))
	}
	fmt.Fprintf(stdout, "Verifier:\n%s", bullet(card.Verify))
	fmt.Fprintf(stdout, "Rollback:\n%s", bullet(card.Rollback))
	fmt.Fprintf(stdout, "False positives:\n%s", bullet(card.FalsePositives))
	fmt.Fprintf(stdout, "Related: %s\n", strings.Join(card.Related, ", "))
}

func runGenomeCollect(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "snapshot output dir")
	privacy := fs.String("privacy", "standard", "standard or strict")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	manifest, err := genome.Collect(ctx, *out, *privacy)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(manifest, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Snapshot written: %s\nFeatures: %s\nPrivacy: %s\n", *out, manifest.FeaturesPath, manifest.PrivacyMode)
	if len(manifest.Warnings) > 0 {
		fmt.Fprintf(stdout, "Warnings: %d\n", len(manifest.Warnings))
	}
	return 0
}

func runGenomeFeatures(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("features", flag.ContinueOnError)
	fs.SetOutput(stderr)
	snapshot := fs.String("snapshot", "", "snapshot dir")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *snapshot == "" {
		fmt.Fprintln(stderr, "features requires --snapshot")
		return 50
	}
	sf, err := genome.LoadFeatures(*snapshot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(sf, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	for _, k := range sortedFeatureKeys(sf.Features) {
		fmt.Fprintf(stdout, "%s=%s\n", k, sf.Features[k])
	}
	return 0
}

func hasGenomeDiagnoseArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--symptom" || strings.HasPrefix(arg, "--symptom=") || arg == "--snapshot" || strings.HasPrefix(arg, "--snapshot=") || arg == "--no-snapshot" {
			return true
		}
	}
	return false
}

func runGenomeDiagnose(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diagnose", flag.ContinueOnError)
	fs.SetOutput(stderr)
	symptom := fs.String("symptom", "", "symptom text")
	snapshot := fs.String("snapshot", "", "snapshot dir")
	noSnapshot := fs.Bool("no-snapshot", false, "diagnose from symptom only")
	top := fs.Int("top", 3, "number of hypotheses")
	jsonOut := fs.Bool("json", false, "print JSON")
	root := fs.String("root", "", "genome corpus root")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *symptom == "" {
		fmt.Fprintln(stderr, "diagnose requires --symptom")
		return 50
	}
	if *noSnapshot {
		*snapshot = ""
	}
	d, err := genome.Diagnose(*root, *symptom, *snapshot, *top)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(d, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	printDiagnosis(d, stdout)
	return 0
}

func printDiagnosis(d genome.Diagnosis, stdout io.Writer) {
	fmt.Fprintf(stdout, "Symptom: %s\nCorpus: %s\n", d.Symptom, d.CorpusVersion)
	fmt.Fprintln(stdout, "Routes:")
	for _, r := range d.Routes {
		fmt.Fprintf(stdout, "- %s: %s\n", r.ID, r.Reason)
	}
	fmt.Fprintln(stdout, "Top hypotheses:")
	for _, h := range d.Hypotheses {
		fmt.Fprintf(stdout, "%d. %s — %s (score=%d risk=%s gate=%s)\n", h.Rank, h.CardID, h.Title, h.Score, h.RiskClass, h.RecipeGate.Result)
		if len(h.WhyMatched) > 0 {
			fmt.Fprintf(stdout, "   why: %s\n", strings.Join(h.WhyMatched, "; "))
		}
		if len(h.NextReadOnlyChecks) > 0 {
			fmt.Fprintf(stdout, "   next read-only: %s\n", strings.Join(h.NextReadOnlyChecks, "; "))
		}
		fmt.Fprintf(stdout, "   gate reason: %s\n", h.RecipeGate.Reason)
	}
}

func runGenomeGate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cardID := fs.String("card", "", "card ID")
	jsonOut := fs.Bool("json", false, "print JSON")
	root := fs.String("root", "", "genome corpus root")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *cardID == "" {
		fmt.Fprintln(stderr, "gate requires --card")
		return 50
	}
	g, err := genome.Gate(*root, *cardID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(g, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "%s — %s\nLayer: %s\nRisk: %s\nRecipe type: %s\nVerifier present: %v\nRollback present: %v\nGate: %s\nReason: %s\n", g.CardID, g.Title, g.Layer, g.RiskClass, g.RecipeType, g.VerifierPresent, g.RollbackPresent, g.Result, g.Reason)
	if g.AllowedNextStep != "" {
		fmt.Fprintf(stdout, "Allowed next step: %s\n", g.AllowedNextStep)
	}
	return 0
}

func isGenomeReportArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--format" || strings.HasPrefix(arg, "--format=") || arg == "--symptom" || strings.HasPrefix(arg, "--symptom=") {
			return true
		}
	}
	return false
}

func runGenomeReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "markdown", "markdown or json")
	snapshot := fs.String("snapshot", "", "snapshot dir")
	symptom := fs.String("symptom", "", "symptom text")
	out := fs.String("out", "", "output path")
	root := fs.String("root", "", "genome corpus root")
	jsonOut := fs.Bool("json", false, "print JSON status")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *symptom == "" {
		fmt.Fprintln(stderr, "report requires --symptom")
		return 50
	}
	textOrPath, err := genome.WriteReport(*root, *format, *snapshot, *symptom, *out)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		status := map[string]string{
			"format":   *format,
			"snapshot": *snapshot,
			"symptom":  *symptom,
		}
		if *out == "" {
			status["inline"] = "true"
		} else {
			status["reportPath"] = textOrPath
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	if *out == "" {
		fmt.Fprint(stdout, textOrPath)
	} else {
		fmt.Fprintf(stdout, "Report written: %s\n", textOrPath)
	}
	return 0
}

func bullet(items []string) string {
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	return b.String()
}

func sortedFeatureKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
