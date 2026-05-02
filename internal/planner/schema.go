package planner

type Decision struct {
	SchemaVersion        int            `json:"schemaVersion"`
	Intent               string         `json:"intent"`
	FailureClass         string         `json:"failureClass"`
	Confidence           float64        `json:"confidence"`
	SelectedRecipe       SelectedRecipe `json:"selectedRecipe"`
	Risk                 string         `json:"risk"`
	RequiresUserApproval bool           `json:"requiresUserApproval"`
	ExpectedVerifiers    []string       `json:"expectedVerifiers"`
	FallbackRecipes      []string       `json:"fallbackRecipes"`
	ExplanationForUser   string         `json:"explanationForUser"`
	Evidence             []string       `json:"evidence,omitempty"`
	StopReason           string         `json:"stopReason,omitempty"`
}

type SelectedRecipe struct {
	ID     string            `json:"id"`
	Params map[string]string `json:"params"`
}
