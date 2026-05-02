package brain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/recipe"
)

const PlannerSystemPrompt = `You are Cactus AgentLink Planner.
You do not execute commands.
You do not output shell scripts.
You only output JSON matching PlannerDecision.
You may select only recipes in the provided recipe catalog.
If confidence < 0.55, return intent="probe" or intent="report", not repair.
All writable repairs require rollback-capable recipes.
Verifier results override your judgment.
Never include secrets.
Never ask the runner to run raw shell.
Never invent recipe IDs.
Never claim success; only verifiers decide success.
Prefer the smallest reversible repair.
If the issue is outside local scope, return report with stopReason.
Return only valid PlannerDecision JSON.
Do not include Markdown.
Do not include code fences.
Do not include prose outside JSON.
Do not include chain-of-thought.`

func FormatPlannerPrompt(systemPrompt, userPrompt string) string {
	return "<start_of_turn>user\n" +
		"System instructions:\n" + systemPrompt + "\n\n" +
		"User request:\n" + userPrompt + "\n\n" +
		"Return only one valid PlannerDecision JSON object. No Markdown. No prose. No code fences.\n" +
		"<end_of_turn>\n<start_of_turn>model\n"
}

type PlanPromptInput struct {
	Target           string
	DryRun           bool
	Yes              bool
	Online           bool
	Facts            facts.Facts
	Recipes          []recipe.Recipe
	LastVerifierJSON string
	FailureBudget    int
}

func BuildPlannerUserPrompt(in PlanPromptInput) string {
	example := planner.Decision{
		SchemaVersion: 1,
		Intent:        "report",
		FailureClass:  "UNKNOWN",
		Confidence:    0.8,
		SelectedRecipe: planner.SelectedRecipe{
			ID:     "",
			Params: map[string]string{},
		},
		Risk:                 recipe.RiskReadOnly,
		RequiresUserApproval: false,
		ExpectedVerifiers:    []string{},
		FallbackRecipes:      []string{},
		ExplanationForUser:   "short redacted explanation",
		Evidence:             []string{},
		StopReason:           "outside_local_scope",
	}
	if len(in.Recipes) > 0 {
		example.Intent = "repair"
		example.SelectedRecipe.ID = in.Recipes[0].ID
		example.Risk = in.Recipes[0].Risk
		example.ExpectedVerifiers = verifierIDs(in.Recipes[0].Verify)
		example.StopReason = ""
	}
	exampleJSON, _ := json.Marshal(example)
	var b strings.Builder
	b.WriteString("Return exactly one PlannerDecision JSON object. Do not include Markdown, prose, code fences, or shell commands.\n")
	b.WriteString("The top-level field schemaVersion is mandatory and must be the number 1.\n")
	b.WriteString("Allowed intents: repair, probe, report, rollback.\n")
	b.WriteString("Repair confidence must be at least 0.55. If unsure, use report or probe.\n")
	b.WriteString("For target=network, do not select network rescue. You may only report that explicit network rescue is required.\n")
	b.WriteString("For target=path, ignore missing API keys, closed proxy ports, and missing opencode/claude/codex configs unless they directly affect shell PATH. Prefer macos-zsh-path-repair when it is in the catalog.\n")
	b.WriteString("For target=keys, prefer api-key-detection-redaction as a read-only report recipe when it is in the catalog.\n")
	b.WriteString("For target=codex, prefer a codex recipe. For target=proxy, prefer a proxy recipe.\n")
	b.WriteString("Risk policy: read_only and safe_patch can run without --yes; reversible_patch requires --yes for execution; network_action requires --online; privileged_action and destructive_action are refused.\n")
	b.WriteString(fmt.Sprintf("Request: target=%s dryRun=%t yes=%t online=%t failureBudget=%d\n", in.Target, in.DryRun, in.Yes, in.Online, in.FailureBudget))
	b.WriteString("Facts summary:\n")
	b.WriteString(factsPromptSummary(in.Facts))
	b.WriteString("\nAllowed recipe catalog:\n")
	for _, r := range in.Recipes {
		params, _ := json.Marshal(r.Params)
		b.WriteString(fmt.Sprintf("- id=%s risk=%s autoAllowed=%t verifiers=%s params=%s\n", r.ID, r.Risk, r.AutoAllowed, strings.Join(verifierIDs(r.Verify), ","), string(params)))
	}
	if in.LastVerifierJSON != "" {
		b.WriteString("Last verifier results: ")
		b.WriteString(in.LastVerifierJSON)
		b.WriteString("\n")
	}
	b.WriteString("Output must match this shape, using only a listed recipe ID when intent is repair:\n")
	b.WriteString(string(exampleJSON))
	return b.String()
}

func oldBuildPlannerUserPrompt(in PlanPromptInput) string {
	payload := map[string]any{
		"target":     in.Target,
		"dryRun":     in.DryRun,
		"yes":        in.Yes,
		"online":     in.Online,
		"riskPolicy": "auto brain may execute only read_only, safe_patch, and reversible_patch with --yes; network_action requires --online; privileged_action and destructive_action are refused",
		"facts":      in.Facts,
		"schema": planner.Decision{
			SchemaVersion: 1,
			Intent:        "repair|probe|report|rollback",
			FailureClass:  "string",
			Confidence:    0.55,
			SelectedRecipe: planner.SelectedRecipe{
				ID:     "recipe-id",
				Params: map[string]string{},
			},
			Risk:                 "read_only|safe_patch|reversible_patch|network_action|privileged_action|destructive_action",
			RequiresUserApproval: false,
			ExpectedVerifiers:    []string{},
			FallbackRecipes:      []string{},
			ExplanationForUser:   "short redacted explanation",
			Evidence:             []string{},
			StopReason:           "",
		},
		"lastVerifierResults": in.LastVerifierJSON,
		"failureBudget":       in.FailureBudget,
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	return "Return only one PlannerDecision JSON object. Do not wrap it in Markdown.\n" + string(data)
}

func factsPromptSummary(f facts.Facts) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- os=%s arch=%s shell=%s home=%s\n", f.OS, f.Arch, f.Shell, f.Home))
	b.WriteString(fmt.Sprintf("- likelyFailures=%s\n", strings.Join(f.LikelyFailures, ",")))
	b.WriteString(fmt.Sprintf("- proxyEnvCount=%d\n", len(f.ProxyEnv)))
	b.WriteString("- commands=")
	toolNames := sortedToolNames(f.Commands)
	for i, name := range toolNames {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(name)
		if f.Commands[name].Exists {
			b.WriteString(":yes")
		} else {
			b.WriteString(":no")
		}
	}
	b.WriteString("\n- configPaths=")
	pathNames := sortedPathNames(f.ConfigPaths)
	for i, name := range pathNames {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(name)
		if f.ConfigPaths[name].Exists {
			b.WriteString(":exists")
		} else {
			b.WriteString(":missing")
		}
	}
	b.WriteString("\n- apiKeys=")
	keyNames := sortedKeyNames(f.APIKeys)
	for i, name := range keyNames {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(name)
		if f.APIKeys[name].Present {
			b.WriteString(":present")
		} else {
			b.WriteString(":missing")
		}
	}
	b.WriteString("\n- ports=")
	portNames := sortedPortNames(f.Ports)
	for i, port := range portNames {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(port)
		if f.Ports[port].Listening {
			b.WriteString(":listening")
		} else {
			b.WriteString(":closed")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func sortedToolNames(m map[string]facts.ToolFact) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedPathNames(m map[string]facts.PathFact) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeyNames(m map[string]facts.KeyFact) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedPortNames(m map[string]facts.PortFact) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func verifierIDs(refs []recipe.VerifierRef) []string {
	var out []string
	for _, ref := range refs {
		out = append(out, ref.ID)
	}
	return out
}

func targetRecipes(reg recipe.Registry, target string) []recipe.Recipe {
	var out []recipe.Recipe
	for _, r := range reg.SupportedList() {
		if !r.AutoAllowed && r.Risk != recipe.RiskReadOnly {
			continue
		}
		if target == "" || recipeMatchesTarget(r, target) {
			out = append(out, r)
		}
	}
	return out
}

func recipeMatchesTarget(r recipe.Recipe, target string) bool {
	id := r.ID
	switch target {
	case "path":
		return strings.Contains(id, "path")
	case "proxy":
		return strings.Contains(id, "proxy")
	case "codex":
		return strings.Contains(id, "codex")
	case "keys":
		return strings.Contains(id, "key")
	case "network":
		return false
	default:
		return true
	}
}
