package diagnose

type DiagnosticReport struct {
	SchemaVersion          int               `json:"schemaVersion"`
	ToolVersion            string            `json:"toolVersion"`
	CreatedAt              string            `json:"createdAt"`
	Host                   HostInfo          `json:"host"`
	Network                NetworkInfo       `json:"network"`
	Reachability           ReachabilityInfo  `json:"reachability"`
	UserConfig             UserConfigInfo    `json:"userConfig"`
	Residues               ResidueInfo       `json:"residues"`
	Classifications        []string          `json:"classifications"`
	RecommendedRepairLevel string            `json:"recommendedRepairLevel"`
	Warnings               []string          `json:"warnings"`
	ReportPath             string            `json:"reportPath,omitempty"`
	Raw                    map[string]string `json:"raw,omitempty"`
}

type HostInfo struct {
	Hostname     string `json:"hostname"`
	MacOSVersion string `json:"macOSVersion"`
	BuildVersion string `json:"buildVersion"`
	Arch         string `json:"arch"`
	RealUser     string `json:"realUser"`
	RealUserHome string `json:"realUserHome,omitempty"`
	RealUserUID  string `json:"realUserUid,omitempty"`
	RealUserGID  string `json:"realUserGid,omitempty"`
	EUID         int    `json:"euid"`
}

type NetworkInfo struct {
	CurrentLocation string             `json:"currentLocation"`
	Services        []NetworkService   `json:"services"`
	HardwarePorts   []HardwarePort     `json:"hardwarePorts"`
	Interfaces      []NetworkInterface `json:"interfaces"`
	DefaultRoute    DefaultRoute       `json:"defaultRoute"`
	DNSSummary      DNSSummary         `json:"dnsSummary"`
	ProxySummary    ProxySummary       `json:"proxySummary"`
	DHCP            []DHCPInfo         `json:"dhcp"`
}

type NetworkService struct {
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

type HardwarePort struct {
	Port            string `json:"port"`
	Device          string `json:"device"`
	EthernetAddress string `json:"ethernetAddress"`
}

type NetworkInterface struct {
	Name          string   `json:"name"`
	Status        string   `json:"status"`
	IPv4          []string `json:"ipv4"`
	IPv6          []string `json:"ipv6"`
	LinkLocalOnly bool     `json:"linkLocalOnly"`
	IsUTun        bool     `json:"isUTun"`
	IsBridge      bool     `json:"isBridge"`
}

type DHCPInfo struct {
	Device        string `json:"device"`
	IPv4          string `json:"ipv4,omitempty"`
	HasLease      bool   `json:"hasLease"`
	LinkLocalOnly bool   `json:"linkLocalOnly"`
	PacketSummary string `json:"packetSummary,omitempty"`
}

type DefaultRoute struct {
	Present   bool   `json:"present"`
	Gateway   string `json:"gateway,omitempty"`
	Interface string `json:"interface,omitempty"`
	Raw       string `json:"raw,omitempty"`
}

type DNSSummary struct {
	Nameservers []string               `json:"nameservers"`
	Search      []string               `json:"search"`
	Resolution  map[string]ProbeResult `json:"resolution"`
	Raw         string                 `json:"raw,omitempty"`
}

type ProxySummary struct {
	HTTPEnabled  bool     `json:"httpEnabled"`
	HTTPSEnabled bool     `json:"httpsEnabled"`
	SOCKSEnabled bool     `json:"socksEnabled"`
	PACEnabled   bool     `json:"pacEnabled"`
	Exceptions   []string `json:"exceptions"`
	Dirty        bool     `json:"dirty"`
	Raw          string   `json:"raw,omitempty"`
}

type ReachabilityInfo struct {
	Gateway      ProbeResult            `json:"gateway"`
	RawIPs       map[string]ProbeResult `json:"rawIPs"`
	DNSNames     map[string]ProbeResult `json:"dnsNames"`
	HTTPSTargets map[string]ProbeResult `json:"httpsTargets"`
	AgentTargets map[string]ProbeResult `json:"agentTargets"`
}

type ProbeResult struct {
	Target       string `json:"target"`
	OK           bool   `json:"ok"`
	Status       string `json:"status,omitempty"`
	HTTPStatus   string `json:"httpStatus,omitempty"`
	Error        string `json:"error,omitempty"`
	DurationMS   int64  `json:"durationMs,omitempty"`
	RedactedInfo string `json:"redactedInfo,omitempty"`
}

type UserConfigInfo struct {
	EnvProxy  map[string]string `json:"envProxy"`
	GitProxy  map[string]string `json:"gitProxy"`
	NpmProxy  map[string]string `json:"npmProxy"`
	BrewProxy map[string]string `json:"brewProxy"`
}

type ResidueInfo struct {
	LaunchDaemons     []ResidueMatch `json:"launchDaemons"`
	LaunchAgents      []ResidueMatch `json:"launchAgents"`
	PrivilegedHelpers []ResidueMatch `json:"privilegedHelpers"`
	SystemExtensions  []ResidueMatch `json:"systemExtensions"`
	GroupContainers   []ResidueMatch `json:"groupContainers"`
	AppSupport        []ResidueMatch `json:"appSupport"`
	Caches            []ResidueMatch `json:"caches"`
	Preferences       []ResidueMatch `json:"preferences"`
	Profiles          []ResidueMatch `json:"profiles"`
}

type ResidueMatch struct {
	RuleID                string `json:"ruleId"`
	DisplayName           string `json:"displayName"`
	Risk                  string `json:"risk"`
	Path                  string `json:"path"`
	Kind                  string `json:"kind"`
	AutoQuarantineAllowed bool   `json:"autoQuarantineAllowed"`
	DetectOnly            bool   `json:"detectOnly"`
	Raw                   string `json:"raw,omitempty"`
}
