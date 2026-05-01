package recipe

type Recipe struct {
	SchemaVersion  int            `json:"schemaVersion"`
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	SupportedOS    []string       `json:"supportedOS"`
	FailureClasses []string       `json:"failureClasses"`
	Risk           string         `json:"risk"`
	RequiresRoot   bool           `json:"requiresRoot"`
	AutoAllowed    bool           `json:"autoAllowed"`
	Params         []ParamSpec    `json:"params,omitempty"`
	Preconditions  []Precondition `json:"preconditions"`
	Probes         []Probe        `json:"probes"`
	Patches        []Patch        `json:"patches"`
	Verify         []VerifierRef  `json:"verify"`
	Rollback       []RollbackSpec `json:"rollback"`
	Docs           []string       `json:"docs"`
}

type ParamSpec struct {
	Name    string   `json:"name"`
	Default string   `json:"default,omitempty"`
	Allowed []string `json:"allowed,omitempty"`
}

type Precondition struct {
	Type            string `json:"type"`
	Path            string `json:"path,omitempty"`
	Command         string `json:"command,omitempty"`
	Name            string `json:"name,omitempty"`
	Value           string `json:"value,omitempty"`
	Port            string `json:"port,omitempty"`
	OptionalWithYes bool   `json:"optionalWithYes,omitempty"`
}

type Probe struct {
	ID   string            `json:"id"`
	Type string            `json:"type"`
	Args map[string]string `json:"args,omitempty"`
}

type Patch struct {
	ID                  string            `json:"id"`
	Type                string            `json:"type"`
	Path                string            `json:"path,omitempty"`
	Marker              string            `json:"marker,omitempty"`
	Content             string            `json:"content,omitempty"`
	Template            string            `json:"template,omitempty"`
	Key                 string            `json:"key,omitempty"`
	Value               string            `json:"value,omitempty"`
	Section             string            `json:"section,omitempty"`
	Fields              map[string]string `json:"fields,omitempty"`
	Args                map[string]string `json:"args,omitempty"`
	OnlyIfTomlMalformed bool              `json:"onlyIfTomlMalformed,omitempty"`
	WhenParamEquals     map[string]string `json:"whenParamEquals,omitempty"`
}

type VerifierRef struct {
	ID              string            `json:"id"`
	Type            string            `json:"type,omitempty"`
	Args            map[string]string `json:"args,omitempty"`
	WhenParamEquals map[string]string `json:"whenParamEquals,omitempty"`
}

type RollbackSpec struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Key  string `json:"key,omitempty"`
}

const (
	RiskReadOnly          = "read_only"
	RiskSafePatch         = "safe_patch"
	RiskReversiblePatch   = "reversible_patch"
	RiskNetworkAction     = "network_action"
	RiskPrivilegedAction  = "privileged_action"
	RiskDestructiveAction = "destructive_action"
)

func WritableRisk(risk string) bool {
	return risk == RiskSafePatch || risk == RiskReversiblePatch || risk == RiskPrivilegedAction
}
