package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"cactus-agentlink-rescue/internal/system"
	"cactus-agentlink-rescue/internal/toolmanifest"
)

// runManifest emits / validates the v0.3.0 typed model-facing tool
// catalog. Pure (no host access): safe for safe-mode and tests.
//
//	agentlink manifest            -> human summary
//	agentlink manifest --json     -> full machine-readable catalog
//	agentlink manifest validate   -> schema-completeness gate (exit 60 on problems)
func runManifest(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "validate" {
		m := toolmanifest.Build(system.Version)
		probs := toolmanifest.Validate(m)
		if len(probs) == 0 {
			fmt.Fprintf(stdout, "manifest OK: %d tools, schema-complete\n", len(m.Tools))
			return 0
		}
		fmt.Fprintf(stderr, "manifest INVALID (%d problems):\n", len(probs))
		for _, p := range probs {
			fmt.Fprintf(stderr, "  - %s\n", p)
		}
		return 60
	}
	fs := flag.NewFlagSet("manifest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print full JSON catalog")
	family := fs.String("family", "", "filter by family (diagnosis|planning|execution|reporting)")
	edition := fs.String("edition", "", "filter by edition (qwen|gemma)")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	m := toolmanifest.Build(system.Version)
	if *family != "" || *edition != "" {
		var keep []toolmanifest.ToolCard
		for _, t := range m.Tools {
			if *family != "" && t.Family != *family {
				continue
			}
			if *edition == "qwen" && !t.QwenAutonomousOK {
				continue
			}
			if *edition == "gemma" && !t.GemmaRecommendOK {
				continue
			}
			keep = append(keep, t)
		}
		m.Tools = keep
	}
	if *jsonOut {
		b, err := m.JSON()
		if err != nil {
			fmt.Fprintf(stderr, "manifest marshal error: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(b))
		return 0
	}
	fmt.Fprintf(stdout, "AgentLink typed tool catalog (schemaVersion %d, agentlink %s)\n",
		m.SchemaVersion, m.AgentLink)
	fmt.Fprintf(stdout, "%d model-facing tools. Forbidden raw surfaces: %s\n\n",
		len(m.Tools), strings.Join(m.Forbidden, ", "))
	fam := ""
	for _, t := range m.Tools {
		if t.Family != fam {
			fam = t.Family
			fmt.Fprintf(stdout, "[%s]\n", strings.ToUpper(fam))
		}
		appr := ""
		if t.HumanApproval {
			appr = " (user-approval)"
		}
		fmt.Fprintf(stdout, "  %-38s risk=%-16s mut=%-12s qwen=%v gemma=%v%s\n    %s\n",
			t.ID, t.RiskClass, t.MutationClass, t.QwenAutonomousOK,
			t.GemmaRecommendOK, appr, t.Description)
	}
	return 0
}
