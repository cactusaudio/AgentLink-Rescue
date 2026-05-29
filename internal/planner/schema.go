package planner

type Decision struct {
	SchemaVersion        int            `json:"schemaVersion"`
	Intent               string         `json:"intent"`
	FailureClass         string         `json:"failureClass"`
	Confidence           float64        `json:"confidence"`
	SelectedRecipe       SelectedRecipe `json:"selectedRecipe"`
	// RepairLevel names a privileged network-state repair tier (safe / standard
	// / tun / deep) executed by the repair engine, used as an ALTERNATIVE to a
	// registry recipe when the routed class mutates system network state rather
	// than user config/files. Exactly one of SelectedRecipe.ID or RepairLevel
	// is set on a repair decision.
	RepairLevel          string         `json:"repairLevel,omitempty"`
	Risk                 string         `json:"risk"`
	RequiresUserApproval bool           `json:"requiresUserApproval"`
	ExpectedVerifiers    []string       `json:"expectedVerifiers"`
	FallbackRecipes      []string       `json:"fallbackRecipes"`
	ExplanationForUser   string         `json:"explanationForUser"`
	Evidence             []string       `json:"evidence,omitempty"`
	StopReason           string         `json:"stopReason,omitempty"`
}

// Repair levels the planner may route a network-state class to. These mirror
// the repair-engine level strings (and classify.RecommendedRepairLevel output)
// — kept as planner constants to avoid importing the repair package.
const (
	LevelSafe     = "safe"
	LevelStandard = "standard"
	LevelDeep     = "deep"
	LevelTun      = "tun"
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
