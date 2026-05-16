package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/diagnosisgraph"
)

// runDiagnoseGraph emits the compact V0300 Layer-3 diagnosis graph.
//
//	agentlink diagnose-graph --json                 (live collect)
//	agentlink diagnose-graph --from report.json --json
//	agentlink diagnose --json | agentlink diagnose-graph --from - --json
//
// --from is the deterministic product + test path (consume a
// DiagnosticReport produced by `agentlink diagnose --json`).
func runDiagnoseGraph(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diagnose-graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	from := fs.String("from", "", "read a DiagnosticReport JSON from file or '-' (stdin)")
	noRedact := fs.Bool("no-redact", false, "do not redact free-text (debug only)")
	if err := fs.Parse(args); err != nil {
		return 50
	}

	var r diagnose.DiagnosticReport
	if *from != "" {
		var raw []byte
		var err error
		if *from == "-" {
			raw, err = io.ReadAll(os.Stdin)
		} else {
			raw, err = os.ReadFile(*from)
		}
		if err != nil {
			fmt.Fprintf(stderr, "diagnose-graph: cannot read report: %v\n", err)
			return 50
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			fmt.Fprintf(stderr, "diagnose-graph: malformed DiagnosticReport JSON: %v\n", err)
			return 50
		}
	} else {
		engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: rulesDir})
		r = engine.Run(ctx)
	}
	classify.Apply(&r)

	g := diagnosisgraph.Build(r, !*noRedact)
	if *jsonOut {
		b, err := g.JSON()
		if err != nil {
			fmt.Fprintf(stderr, "diagnose-graph: marshal error: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(b))
		return 0
	}
	fmt.Fprintf(stdout, "Diagnosis graph (schemaVersion %d, redacted=%v)\n", g.SchemaVersion, g.Redacted)
	fmt.Fprintf(stdout, "primaryClass=%s confidence=%.2f\n", g.PrimaryClass, g.Confidence)
	if len(g.Symptoms) > 0 {
		fmt.Fprintln(stdout, "symptoms:")
		for _, s := range g.Symptoms {
			fmt.Fprintf(stdout, "  - %s\n", s)
		}
	}
	fmt.Fprintln(stdout, "recommended next tools:")
	for _, tr := range g.RecommendedNextTools {
		fmt.Fprintf(stdout, "  -> %-38s %s\n", tr.ID, tr.Reason)
	}
	fmt.Fprintln(stdout, "forbidden (until dry-run + approval / never):")
	for _, f := range g.ForbiddenNextTools {
		fmt.Fprintf(stdout, "  x %s\n", f)
	}
	return 0
}
