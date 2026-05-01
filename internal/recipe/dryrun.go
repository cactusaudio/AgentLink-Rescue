package recipe

type DryRunAction struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Path        string `json:"path,omitempty"`
}
