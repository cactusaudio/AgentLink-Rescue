// Package toolmanifest is the v0.3.0 typed, machine-readable tool catalog
// for model-facing AgentLink surfaces.
//
// Commercial product principle (V0300 goal): models decide from
// structured, indexed, low-entropy AgentLink evidence; they NEVER
// improvise raw system repair. This catalog is the single coherent
// surface an autonomous tool-loop operator edition or a small-memory
// recommender is allowed to see. Raw sudo/networksetup/route/ifconfig/
// launchctl and `rescue --yes` are intentionally NOT in this catalog.
package toolmanifest

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Risk classes mirror recipe risk semantics (read_only is safest).
const (
	RiskReadOnly    = "read_only"
	RiskSafePatch   = "safe_patch"
	RiskReversible  = "reversible_patch"
	RiskNetwork     = "network_action"
	RiskPrivileged  = "privileged_action"
	RiskDestructive = "destructive_action"
)

// Mutation classes describe whether a tool changes host state.
const (
	MutationNone        = "none"         // pure read / report
	MutationSandboxOnly = "sandbox_only" // mutates only a sandbox/workspace
	MutationHostTxn     = "host_txn"     // mutates host ONLY via AgentLink txn envelope
)

var validRisk = map[string]bool{
	RiskReadOnly: true, RiskSafePatch: true, RiskReversible: true,
	RiskNetwork: true, RiskPrivileged: true, RiskDestructive: true,
}
var validMutation = map[string]bool{
	MutationNone: true, MutationSandboxOnly: true, MutationHostTxn: true,
}

// ToolCard is the typed contract for one model-facing tool. Metadata
// carries enough semantic load that a small/weak model can choose
// safely WITHOUT reading long docs first (V0300 Layer 2).
type ToolCard struct {
	ID                 string         `json:"id"`
	Family             string         `json:"family"`
	Description        string         `json:"description"`
	Argv               []string       `json:"argv"` // exact AgentLink invocation (no shell)
	InputSchema        map[string]any `json:"inputSchema"`
	OutputSchema       map[string]any `json:"outputSchema"`
	RiskClass          string         `json:"riskClass"`
	MutationClass      string         `json:"mutationClass"`
	DryRunSupported    bool           `json:"dryRunSupported"`
	RollbackSupported  bool           `json:"rollbackSupported"`
	RollbackImpossible bool           `json:"rollbackImpossible"`
	Preconditions      []string       `json:"preconditions"`
	Postconditions     []string       `json:"postconditions"`
	ExampleGoodCall    string         `json:"exampleGoodCall"`
	ExampleBadCall     string         `json:"exampleBadCall"`
	PossibleErrors     []string       `json:"possibleErrors"`
	TimeoutSeconds     int            `json:"timeoutSeconds"`
	WhenNotToUse       string         `json:"whenNotToUse"`
	RelatedTools       []string       `json:"relatedTools"`
	HumanApproval      bool           `json:"humanApprovalRequired"`
	AutonomousAllowed  bool           `json:"autonomousAllowed"`
	RecommendAllowed   bool           `json:"recommendAllowed"`
}

// Manifest is the top-level catalog document.
type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	Product       string     `json:"product"`
	AgentLink     string     `json:"agentlinkVersion"`
	Note          string     `json:"note"`
	Forbidden     []string   `json:"forbiddenRawSurfaces"`
	Tools         []ToolCard `json:"tools"`
}

const forbiddenNote = "Models must call ONLY tools in this catalog via the " +
	"typed bridge. Raw sudo/networksetup/route/ifconfig/launchctl and " +
	"`agentlink rescue --yes` are NEVER model-facing. Host mutation " +
	"occurs only inside AgentLink transaction envelopes."

// ForbiddenRawSurfaces is the set of raw command STRINGS that must never
// be exposed to a model as an executable command (asserted by policy
// tests that classify free-form command strings on both repos).
var ForbiddenRawSurfaces = []string{
	"sudo", "networksetup", "route", "ifconfig", "launchctl",
	"rm -rf", "agentlink rescue --yes", "killall", "pkill",
}

// rawBinaries are privileged system binaries that must never be argv[0]
// of a catalog tool. NOTE: "route"/"ifconfig" etc. are ALSO legitimate
// AgentLink read-only *subcommand nouns* (e.g. `agentlink diagnose
// route`); the typed bridge always execs the `agentlink` binary, so
// argv[0] is an AgentLink subcommand, never a raw binary. The real
// danger in a catalog argv is an auto-execute / privilege token.
var rawBinaries = map[string]bool{
	"sudo": true, "networksetup": true, "route": true, "ifconfig": true,
	"launchctl": true, "killall": true, "pkill": true, "rescue": true,
}

// argvUnsafe returns a reason if a catalog tool's argv would bypass the
// AgentLink envelope or auto-execute privileged repair. Precise (does
// NOT false-flag AgentLink diagnosis subcommand nouns).
func argvUnsafe(argv []string) string {
	if len(argv) == 0 {
		return "empty argv"
	}
	if rawBinaries[argv[0]] {
		return "argv[0] is a raw/privileged surface: " + argv[0]
	}
	for _, a := range argv {
		switch a {
		case "--yes", "-y":
			return "argv contains auto-execute flag " + a + " (model execution is dry-run only; real apply via approval ticket)"
		case "sudo":
			return "argv contains sudo"
		}
		if strings.HasPrefix(a, "rm") && strings.Contains(a, "-rf") {
			return "argv contains unbounded delete"
		}
	}
	return ""
}

func obj(m map[string]any) map[string]any { return m }

// Build returns the curated, deterministic catalog for the given
// AgentLink version. Pure (no host access) so it is test-stable.
func Build(agentlinkVersion string) Manifest {
	tools := append(readOnlyTools(), planningTools()...)
	tools = append(tools, executionTools()...)
	tools = append(tools, reportingTools()...)
	sort.SliceStable(tools, func(i, j int) bool { return tools[i].ID < tools[j].ID })
	return Manifest{
		SchemaVersion: 1,
		Product:       "AgentLink Rescue typed tool catalog",
		AgentLink:     agentlinkVersion,
		Note:          forbiddenNote,
		Forbidden:     ForbiddenRawSurfaces,
		Tools:         tools,
	}
}

func jsonResult() map[string]any {
	return obj(map[string]any{"type": "object",
		"required": []string{"schemaVersion"},
		"properties": obj(map[string]any{
			"schemaVersion": obj(map[string]any{"type": "integer"})})})
}
func noInput() map[string]any {
	return obj(map[string]any{"type": "object", "properties": obj(map[string]any{})})
}

func readOnlyTools() []ToolCard {
	ro := func(id, fam, desc string, argv []string, secs int, related []string) ToolCard {
		return ToolCard{ID: id, Family: fam, Description: desc, Argv: argv,
			InputSchema: noInput(), OutputSchema: jsonResult(),
			RiskClass: RiskReadOnly, MutationClass: MutationNone,
			DryRunSupported: false, RollbackSupported: false,
			Preconditions:   []string{"agentlink binary discoverable"},
			Postconditions:  []string{"no host state changed"},
			ExampleGoodCall: "agentlink " + joinArgv(argv) + " --json",
			ExampleBadCall:  "sudo agentlink " + joinArgv(argv) + "  # never prefix with sudo",
			PossibleErrors:  []string{"binary_not_found", "timeout", "malformed_json"},
			TimeoutSeconds:  secs,
			WhenNotToUse:    "Do not use to mutate state; read-only diagnosis only.",
			RelatedTools:    related, HumanApproval: false,
			AutonomousAllowed: true, RecommendAllowed: true}
	}
	return []ToolCard{
		ro("agentlink.version", "diagnosis", "Print AgentLink version + capability summary.", []string{"version"}, 15, []string{"agentlink.doctor"}),
		ro("agentlink.doctor", "diagnosis", "Comprehensive read-only network/system diagnostic report.", []string{"doctor"}, 60, []string{"agentlink.diagnosis_graph", "agentlink.classify_incident"}),
		ro("agentlink.readiness_doctor", "diagnosis", "Offline readiness doctor: are rescue prerequisites present.", []string{"readiness"}, 45, []string{"agentlink.doctor"}),
		ro("agentlink.support_bundle", "diagnosis", "Generate a redacted support bundle of host evidence.", []string{"support", "create"}, 90, []string{"agentlink.support_bundle_redacted"}),
		ro("agentlink.verify_network", "diagnosis", "Probe network/DNS/HTTPS reachability (read-only).", []string{"verify", "network"}, 60, []string{"agentlink.doctor"}),
		ro("agentlink.network_snapshot", "diagnosis", "Capture a structured snapshot of network state.", []string{"snapshot", "--json"}, 60, []string{"agentlink.diff"}),
		ro("agentlink.proxy_snapshot", "diagnosis", "Snapshot system/user/git/npm/brew proxy configuration.", []string{"proxy", "--json"}, 45, []string{"agentlink.recipe_dry_run"}),
		ro("agentlink.dns_snapshot", "diagnosis", "Snapshot DNS resolvers and scoped resolver state.", []string{"diagnose", "dns", "--json"}, 45, []string{"agentlink.verify_network"}),
		ro("agentlink.route_snapshot", "diagnosis", "Snapshot the route table summary.", []string{"diagnose", "route", "--json"}, 45, []string{"agentlink.tun_snapshot"}),
		ro("agentlink.tun_snapshot", "diagnosis", "Snapshot TUN/utun/VPN interface + route ownership state.", []string{"diagnose", "tun", "--json"}, 45, []string{"agentlink.network_extension_snapshot"}),
		ro("agentlink.network_extension_snapshot", "diagnosis", "Snapshot Network Extension / system extension residue.", []string{"diagnose", "netext", "--json"}, 45, []string{"agentlink.launchd_snapshot"}),
		ro("agentlink.launchd_snapshot", "diagnosis", "Snapshot launchd agent/daemon residue relevant to network.", []string{"diagnose", "launchd", "--json"}, 45, []string{"agentlink.app_residue_snapshot"}),
		ro("agentlink.app_residue_snapshot", "diagnosis", "Snapshot known proxy/VPN app residue (no vendor endorsement).", []string{"diagnose", "residue", "--json"}, 45, []string{"agentlink.recommend_recipes"}),
		ro("agentlink.package_doctor", "diagnosis", "Doctor for AgentLink package/runtime integrity.", []string{"package", "doctor"}, 45, []string{"agentlink.readiness_doctor"}),
		ro("agentlink.journal_list", "diagnosis", "List mutation transaction journal entries.", []string{"journal", "list", "--json"}, 30, []string{"agentlink.rollback_last"}),
		ro("agentlink.last_good_list", "diagnosis", "List saved last-good network profiles.", []string{"last-good", "list", "--json"}, 30, []string{"agentlink.restore_last_good"}),
	}
}

func planningTools() []ToolCard {
	return []ToolCard{
		{ID: "agentlink.classify_incident", Family: "planning",
			Description: "Classify the current network failure into a deterministic failure class.",
			Argv:        []string{"classify", "--json"},
			InputSchema: noInput(), OutputSchema: jsonResult(),
			RiskClass: RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"diagnosis evidence available"},
			Postconditions:  []string{"failure class + confidence emitted"},
			ExampleGoodCall: "agentlink classify --json",
			ExampleBadCall:  "agentlink classify && sudo networksetup ...  # never chain raw repair",
			PossibleErrors:  []string{"insufficient_evidence", "timeout"},
			TimeoutSeconds:  45, WhenNotToUse: "Not for execution; selection input only.",
			RelatedTools:      []string{"agentlink.recommend_recipes", "agentlink.diagnosis_graph"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.diagnosis_graph", Family: "planning",
			Description: "Compact structured diagnosis graph (symptoms, topology constraints, repair corridor, confidence, recommended + forbidden next tools). Small enough for a small-memory model, precise for an autonomous operator.",
			Argv:        []string{"diagnose-graph", "--json"},
			InputSchema: noInput(), OutputSchema: jsonResult(),
			RiskClass: RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"agentlink binary discoverable"},
			Postconditions:  []string{"compact diagnosis graph emitted; no host change"},
			ExampleGoodCall: "agentlink diagnose-graph --json",
			ExampleBadCall:  "agentlink diagnose-graph --raw-dump  # do not request raw host dumps for small models",
			PossibleErrors:  []string{"timeout", "malformed_json"},
			TimeoutSeconds:  60, WhenNotToUse: "Do not use as an execution surface.",
			RelatedTools:      []string{"agentlink.classify_incident", "agentlink.recommend_recipes"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.recommend_recipes", Family: "planning",
			Description: "Rank candidate repair recipes for the classified incident with short reasons.",
			Argv:        []string{"orchestrator", "rescue", "--dry-run", "--json"},
			InputSchema: noInput(), OutputSchema: jsonResult(),
			RiskClass: RiskReadOnly, MutationClass: MutationNone, DryRunSupported: true,
			Preconditions:   []string{"incident classified"},
			Postconditions:  []string{"ranked recipe list emitted; nothing executed"},
			ExampleGoodCall: "agentlink orchestrator rescue --dry-run --json",
			ExampleBadCall:  "agentlink orchestrator rescue --yes  # --yes is never model-facing",
			PossibleErrors:  []string{"no_candidate_recipe", "timeout"},
			TimeoutSeconds:  90, WhenNotToUse: "Never pass --yes; this is planning only.",
			RelatedTools:      []string{"agentlink.recipe_inspect", "agentlink.recipe_dry_run"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.recipe_list", Family: "planning",
			Description: "List bounded repair recipes (id, title, risk, failure classes).",
			Argv:        []string{"recipe", "list", "--json"},
			InputSchema: noInput(), OutputSchema: jsonResult(),
			RiskClass: RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"recipes directory present"},
			Postconditions:  []string{"recipe index emitted"},
			ExampleGoodCall: "agentlink recipe list --json",
			ExampleBadCall:  "agentlink recipe run X --yes  # execution must go through the typed execution tool",
			PossibleErrors:  []string{"recipes_dir_missing"},
			TimeoutSeconds:  30, WhenNotToUse: "Not execution.",
			RelatedTools:      []string{"agentlink.recipe_inspect"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.recipe_inspect", Family: "planning",
			Description: "Inspect one recipe: preconditions, patches, verify, rollback, docs.",
			Argv:        []string{"recipe", "inspect", "<recipe_id>", "--json"},
			InputSchema: obj(map[string]any{"type": "object", "required": []string{"recipe_id"},
				"properties": obj(map[string]any{"recipe_id": obj(map[string]any{"type": "string"})})}),
			OutputSchema: jsonResult(),
			RiskClass:    RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"recipe_id from recipe_list"},
			Postconditions:  []string{"recipe detail emitted"},
			ExampleGoodCall: "agentlink recipe inspect proxy-clean-stale-env --json",
			ExampleBadCall:  "agentlink recipe inspect $(rm -rf ~)  # never interpolate shell",
			PossibleErrors:  []string{"recipe_not_found"},
			TimeoutSeconds:  30, WhenNotToUse: "Not execution.",
			RelatedTools:      []string{"agentlink.recipe_dry_run"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.recipe_dry_run", Family: "planning",
			Description: "Dry-run a recipe: render exactly what WOULD change, no mutation.",
			Argv:        []string{"recipe", "run", "<recipe_id>", "--dry-run", "--json"},
			InputSchema: obj(map[string]any{"type": "object", "required": []string{"recipe_id"},
				"properties": obj(map[string]any{"recipe_id": obj(map[string]any{"type": "string"})})}),
			OutputSchema: jsonResult(),
			RiskClass:    RiskReadOnly, MutationClass: MutationNone, DryRunSupported: true,
			Preconditions:   []string{"recipe_id valid"},
			Postconditions:  []string{"planned actions emitted; zero host mutation"},
			ExampleGoodCall: "agentlink recipe dry-run dns-resolver-baseline --json",
			ExampleBadCall:  "agentlink recipe run dns-resolver-baseline --yes  # use execute tool w/ approval",
			PossibleErrors:  []string{"recipe_not_found", "precondition_failed"},
			TimeoutSeconds:  60, WhenNotToUse: "Not for actual repair.",
			RelatedTools:      []string{"agentlink.execute_recipe"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.repair_plan_validate", Family: "planning",
			Description: "Validate a planner/brain JSON decision against the safety contract.",
			Argv:        []string{"planner", "validate", "--json"},
			InputSchema: obj(map[string]any{"type": "object", "required": []string{"decisionJson"},
				"properties": obj(map[string]any{"decisionJson": obj(map[string]any{"type": "string"})})}),
			OutputSchema: jsonResult(),
			RiskClass:    RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"a planner decision JSON"},
			Postconditions:  []string{"valid/invalid + reasons emitted"},
			ExampleGoodCall: "agentlink planner validate --json < decision.json",
			ExampleBadCall:  "agentlink planner validate --exec  # validation never executes",
			PossibleErrors:  []string{"malformed_decision", "schema_violation"},
			TimeoutSeconds:  20, WhenNotToUse: "Not execution.",
			RelatedTools:      []string{"agentlink.recommend_recipes"},
			AutonomousAllowed: true, RecommendAllowed: false},
		{ID: "agentlink.risk_score", Family: "planning",
			Description: "Return the deterministic risk class + confidence for a candidate recipe.",
			Argv:        []string{"recipe", "inspect", "<recipe_id>", "--json"},
			InputSchema: obj(map[string]any{"type": "object", "required": []string{"recipe_id"},
				"properties": obj(map[string]any{"recipe_id": obj(map[string]any{"type": "string"})})}),
			OutputSchema: jsonResult(),
			RiskClass:    RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"recipe_id valid"},
			Postconditions:  []string{"risk class + confidence emitted"},
			ExampleGoodCall: "agentlink recipe inspect tun-residue-clean --json | jq .risk",
			ExampleBadCall:  "ignore risk and run anyway  # never bypass risk gating",
			PossibleErrors:  []string{"recipe_not_found"},
			TimeoutSeconds:  30, WhenNotToUse: "Not execution.",
			RelatedTools:      []string{"agentlink.recipe_inspect"},
			AutonomousAllowed: true, RecommendAllowed: true},
		{ID: "agentlink.approval_ticket_create", Family: "planning",
			Description: "Create a terminal repair ticket (.command) for a user-approved privileged repair. The USER runs it; the model never does.",
			Argv:        []string{"ticket", "--type", "<ticket_type>"},
			InputSchema: obj(map[string]any{"type": "object", "required": []string{"ticket_type"},
				"properties": obj(map[string]any{"ticket_type": obj(map[string]any{"type": "string",
					"enum": []string{"clash-tun-fix", "rollback", "verify-network"}})})}),
			OutputSchema: jsonResult(),
			RiskClass:    RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"a repair requiring user-approved privilege"},
			Postconditions:  []string{"a ticket file path emitted; nothing executed by the model"},
			ExampleGoodCall: "agentlink ticket --type verify-network",
			ExampleBadCall:  "bash $(agentlink ticket ...)  # the model must NEVER execute the ticket",
			PossibleErrors:  []string{"unknown_ticket_type"},
			TimeoutSeconds:  20, WhenNotToUse: "Do not auto-run the ticket; user approval is mandatory.",
			RelatedTools:  []string{"agentlink.execute_recipe"},
			HumanApproval: true, AutonomousAllowed: true, RecommendAllowed: true},
	}
}

// RecipeBackedExecTools maps execution tool id -> the REAL recipe id it
// runs. Asserted against the live recipe registry by W4 tests (no
// dangling). CLI-backed execution tools (rollback/journal-recover/
// last-good/package) are listed in CLIBackedExecTools.
var RecipeBackedExecTools = map[string]string{
	"agentlink.clean_stale_proxy_baseline": "proxy-clean-stale-env",
	"agentlink.network_baseline_reset":     "macos-clean-network-baseline-reset",
	"agentlink.remove_tun_residue":         "macos-clash-tun-force-repair",
	"agentlink.clean_npm_git_proxy":        "npm-git-proxy-conflict-repair",
}

// CLIBackedExecTools maps execution tool id -> its real AgentLink
// transactional subcommand argv (journal/snapshot-backed, not a recipe).
var CLIBackedExecTools = map[string][]string{
	"agentlink.restore_last_good":           {"last-good", "restore", "--json"},
	"agentlink.rollback_last":               {"rollback", "--json"},
	"agentlink.recover_interrupted_journal": {"journal", "recover", "--json"},
	"agentlink.repair_package_runtime":      {"package", "repair", "--json"},
}

func execCard(id, desc string, argv []string, risk string, rollback bool) ToolCard {
	return ToolCard{ID: id, Family: "execution", Description: desc, Argv: argv,
		InputSchema: obj(map[string]any{"type": "object",
			"required": []string{"confirmed_dry_run", "user_approved"},
			"properties": obj(map[string]any{
				"confirmed_dry_run": obj(map[string]any{"type": "boolean"}),
				"user_approved":     obj(map[string]any{"type": "boolean"})})}),
		OutputSchema: jsonResult(),
		RiskClass:    risk, MutationClass: MutationHostTxn,
		DryRunSupported: true, RollbackSupported: rollback,
		RollbackImpossible: !rollback,
		Preconditions: []string{"dry-run inspected first", "snapshot captured",
			"user approval obtained", "incident class matches"},
		Postconditions: []string{"post-verify passed OR automatic rollback",
			"journal entry written"},
		ExampleGoodCall: "after dry-run + approval: agentlink " + joinArgv(argv),
		ExampleBadCall:  "agentlink " + joinArgv(argv) + " --yes  # raw --yes is never model-facing",
		PossibleErrors: []string{"precondition_failed", "postverify_failed_rolled_back",
			"timeout_safe_abort", "rollback_failed_fail_closed"},
		TimeoutSeconds: 180,
		WhenNotToUse:   "Never without a prior dry-run + explicit user approval.",
		RelatedTools:   []string{"agentlink.recipe_dry_run", "agentlink.rollback_last"},
		HumanApproval:  true, AutonomousAllowed: true, RecommendAllowed: false}
}

func executionTools() []ToolCard {
	cards := []ToolCard{
		execCard("agentlink.execute_recipe",
			"Execute a named bounded repair recipe inside an AgentLink transaction envelope (snapshot→preflight→mutate→postverify→journal, auto-rollback on failure). recipe_id MUST come from agentlink.recipe_list.",
			[]string{"recipe", "run", "<recipe_id>", "--dry-run", "--json"}, RiskReversible, true),
		execCard("agentlink.clean_stale_proxy_baseline",
			"Clean stale shell proxy env residue (recipe proxy-clean-stale-env, reversible).",
			[]string{"recipe", "run", "proxy-clean-stale-env", "--dry-run", "--json"}, RiskReversible, true),
		execCard("agentlink.network_baseline_reset",
			"Reset macOS network to a clean baseline (recipe macos-clean-network-baseline-reset; PRIVILEGED, root — user-approved ticket).",
			[]string{"recipe", "run", "macos-clean-network-baseline-reset", "--dry-run", "--json"}, RiskPrivileged, true),
		execCard("agentlink.remove_tun_residue",
			"Force-repair stale Clash/TUN/route ownership residue (recipe macos-clash-tun-force-repair; PRIVILEGED).",
			[]string{"recipe", "run", "macos-clash-tun-force-repair", "--dry-run", "--json"}, RiskPrivileged, true),
		execCard("agentlink.clean_npm_git_proxy",
			"Resolve npm/git proxy conflict residue (recipe npm-git-proxy-conflict-repair, reversible).",
			[]string{"recipe", "run", "npm-git-proxy-conflict-repair", "--dry-run", "--json"}, RiskReversible, true),
		execCard("agentlink.restore_last_good",
			"Restore the last-good saved network profile (AgentLink transactional restore).",
			CLIBackedExecTools["agentlink.restore_last_good"], RiskNetwork, true),
		execCard("agentlink.rollback_last",
			"Roll back the last AgentLink transaction from the durable journal.",
			CLIBackedExecTools["agentlink.rollback_last"], RiskReversible, true),
		execCard("agentlink.recover_interrupted_journal",
			"Recover an interrupted repair journal to a consistent state (journal-driven).",
			CLIBackedExecTools["agentlink.recover_interrupted_journal"], RiskReversible, true),
		execCard("agentlink.repair_package_runtime",
			"Repair AgentLink package/runtime integrity (package doctor→repair).",
			CLIBackedExecTools["agentlink.repair_package_runtime"], RiskSafePatch, true),
	}
	return cards
}

func reportingTools() []ToolCard {
	rep := func(id, desc string, argv []string, gemma bool) ToolCard {
		return ToolCard{ID: id, Family: "reporting", Description: desc, Argv: argv,
			InputSchema: noInput(), OutputSchema: jsonResult(),
			RiskClass: RiskReadOnly, MutationClass: MutationNone,
			Preconditions:   []string{"an incident session exists"},
			Postconditions:  []string{"report emitted; no host change"},
			ExampleGoodCall: "agentlink " + joinArgv(argv),
			ExampleBadCall:  "include raw secrets in the report  # reports MUST be redacted",
			PossibleErrors:  []string{"no_session", "timeout"},
			TimeoutSeconds:  45, WhenNotToUse: "Not execution.",
			RelatedTools:      []string{"agentlink.support_bundle_redacted"},
			AutonomousAllowed: true, RecommendAllowed: gemma}
	}
	return []ToolCard{
		rep("agentlink.incident_report", "Generate a structured incident report (human or codex form).", []string{"report", "--for-human"}, true),
		rep("agentlink.user_summary", "Generate a short plain-language user-facing result summary.", []string{"report", "--for-human", "--summary"}, true),
		rep("agentlink.operator_log", "Emit the redacted operator/mutation log for the session.", []string{"journal", "list", "--json"}, false),
		rep("agentlink.support_bundle_redacted", "Generate a redacted support bundle safe to share.", []string{"support", "create", "--redacted"}, true),
		rep("agentlink.after_action_report", "Generate the post-repair after-action report.", []string{"report", "--for-codex", "--after-action"}, true),
	}
}

func joinArgv(a []string) string {
	out := ""
	for i, s := range a {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

// Validate enforces schema completeness. Returns a list of problems
// (empty => the catalog is product-complete). Pure + deterministic.
func Validate(m Manifest) []string {
	var probs []string
	seen := map[string]bool{}
	add := func(id, msg string) { probs = append(probs, fmt.Sprintf("%s: %s", id, msg)) }
	if m.SchemaVersion < 1 {
		probs = append(probs, "manifest: schemaVersion must be >=1")
	}
	if len(m.Tools) == 0 {
		probs = append(probs, "manifest: empty tool catalog")
	}
	for _, t := range m.Tools {
		if t.ID == "" {
			add("<noid>", "empty id")
			continue
		}
		if seen[t.ID] {
			add(t.ID, "duplicate id")
		}
		seen[t.ID] = true
		if t.Description == "" {
			add(t.ID, "empty description")
		}
		if len(t.Argv) == 0 {
			add(t.ID, "empty argv")
		}
		if !validRisk[t.RiskClass] {
			add(t.ID, "invalid riskClass "+t.RiskClass)
		}
		if !validMutation[t.MutationClass] {
			add(t.ID, "invalid mutationClass "+t.MutationClass)
		}
		if t.InputSchema == nil {
			add(t.ID, "nil inputSchema")
		}
		if t.OutputSchema == nil {
			add(t.ID, "nil outputSchema")
		}
		if t.ExampleGoodCall == "" {
			add(t.ID, "missing exampleGoodCall")
		}
		if t.ExampleBadCall == "" {
			add(t.ID, "missing exampleBadCall")
		}
		if len(t.PossibleErrors) == 0 {
			add(t.ID, "missing possibleErrors")
		}
		if t.TimeoutSeconds <= 0 {
			add(t.ID, "timeoutSeconds must be > 0")
		}
		if t.WhenNotToUse == "" {
			add(t.ID, "missing whenNotToUse")
		}
		// Safety invariants: host-mutating tools MUST require approval,
		// support dry-run, and NOT be recommend-edition executable.
		if t.MutationClass == MutationHostTxn {
			if !t.HumanApproval {
				add(t.ID, "host_txn tool must require human approval")
			}
			if !t.DryRunSupported {
				add(t.ID, "host_txn tool must support dry-run")
			}
			if t.RecommendAllowed {
				add(t.ID, "host_txn execution must NOT be recommend-edition allowed (recommend-only)")
			}
			if !t.RollbackSupported && !t.RollbackImpossible {
				add(t.ID, "host_txn tool must declare rollback support or impossibility")
			}
		}
		// No catalog tool's argv may bypass the AgentLink envelope or
		// auto-execute privileged repair (precise; AgentLink diagnosis
		// subcommand nouns like "route" are legitimate).
		if reason := argvUnsafe(t.Argv); reason != "" {
			add(t.ID, reason)
		}
	}
	return probs
}

// JSON renders the manifest as stable indented JSON.
func (m Manifest) JSON() ([]byte, error) { return json.MarshalIndent(m, "", "  ") }
