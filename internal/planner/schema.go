package planner

type Decision struct {
	SchemaVersion  int            `json:"schemaVersion"`
	Intent         string         `json:"intent"`
	FailureClass   string         `json:"failureClass"`
	Confidence     float64        `json:"confidence"`
	SelectedRecipe SelectedRecipe `json:"selectedRecipe"`
	// RepairLevel names a privileged network-state repair tier (safe / standard
	// / tun / deep) executed by the repair engine, used as an ALTERNATIVE to a
	// registry recipe when the routed class mutates system network state rather
	// than user config/files. Exactly one of SelectedRecipe.ID or RepairLevel
	// is set on a repair decision.
	RepairLevel          string   `json:"repairLevel,omitempty"`
	Risk                 string   `json:"risk"`
	RequiresUserApproval bool     `json:"requiresUserApproval"`
	ExpectedVerifiers    []string `json:"expectedVerifiers"`
	FallbackRecipes      []string `json:"fallbackRecipes"`
	// Escalation is the graduated next-tier ladder if this repair does not
	// resolve the fault (e.g. safe -> standard -> deep -> last-resort clean
	// baseline). Distilled escalation judgment; advisory, not auto-executed.
	Escalation []string `json:"escalation,omitempty"`
	// CoFaults names recipes for OTHER independent reversible faults detected
	// alongside the primary class (multi-fault awareness). The kernel executes
	// the primary decision; co-faults are surfaced for follow-up.
	CoFaults           []string `json:"coFaults,omitempty"`
	ExplanationForUser string   `json:"explanationForUser"`
	Evidence           []string `json:"evidence,omitempty"`
	StopReason         string   `json:"stopReason,omitempty"`
}

// escalationFor returns the graduated next-tier ladder beyond the chosen repair
// (level or reversible recipe) — the distilled "if this does not resolve it,
// escalate to..." judgment.
func escalationFor(level string) []string {
	switch level {
	case LevelSafe:
		return []string{LevelStandard, LevelDeep, LevelNuclear}
	case LevelStandard:
		return []string{LevelDeep, LevelNuclear}
	case LevelDeep:
		return []string{LevelNuclear}
	case LevelNuclear:
		return nil // terminal: nothing escalates beyond the connectivity-first full reset
	default: // reversible user-config recipe: escalate into the network tiers
		return []string{LevelSafe, LevelStandard, LevelNuclear}
	}
}

// Repair levels the planner may route a network-state class to. These mirror
// the repair-engine level strings (and classify.RecommendedRepairLevel output)
// — kept as planner constants to avoid importing the repair package.
const (
	LevelSafe     = "safe"
	LevelStandard = "standard"
	LevelDeep     = "deep"
	LevelTun      = "tun"
	// LevelNuclear is the connectivity-first full reset — the terminal rung of
	// every escalation ladder. It always restores an online-capable baseline on
	// any Apple Silicon Mac.
	LevelNuclear = "nuclear"
)

func validRepairLevel(level string) bool {
	switch level {
	case LevelSafe, LevelStandard, LevelDeep, LevelTun:
		return true
	default:
		return false
	}
}

type SelectedRecipe struct {
	ID     string            `json:"id"`
	Params map[string]string `json:"params"`
}
