package interference

import (
	"strings"
)

const (
	ProviderClashVergeRev = "clash-verge-rev"
	ProviderClashX        = "clashx"
	ProviderMihomoGeneric = "mihomo-generic"
)

type Provider struct {
	ID                    string   `json:"id"`
	DisplayName           string   `json:"displayName"`
	ProcessPatterns       []string `json:"processPatterns"`
	UTunSignatures        []string `json:"utunSignatures"`
	LaunchItems           []string `json:"launchItems"`
	PrivilegedHelpers     []string `json:"privilegedHelpers"`
	UserResiduePaths      []string `json:"userResiduePaths"`
	SystemResiduePaths    []string `json:"systemResiduePaths"`
	NetworkExtensionHints []string `json:"networkExtensionHints"`
	RollbackPolicy        string   `json:"rollbackPolicy"`
	AutomatedRepair       bool     `json:"automatedRepair"`
	DetectOnly            bool     `json:"detectOnly"`
}

type ProviderFact struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"displayName"`
	Detected         bool     `json:"detected"`
	RuntimeActive    bool     `json:"runtimeActive"`
	StaleTun         bool     `json:"staleTunDetected"`
	ResidueDetected  bool     `json:"residueDetected"`
	RequiresAdmin    bool     `json:"requiresAdmin"`
	SafeToQuarantine bool     `json:"safeToQuarantine"`
	Evidence         []string `json:"evidence,omitempty"`
}

func Catalog() []Provider {
	return []Provider{
		{
			ID:              ProviderClashVergeRev,
			DisplayName:     "Clash Verge Rev",
			ProcessPatterns: []string{"Clash Verge", "clash-verge", "verge-mihomo", "mihomo", "clash-meta", "clash"},
			UTunSignatures:  []string{"198.18.0.1", "198.18.0.0/15", "fdfe:dcba:9876::1"},
			LaunchItems: []string{
				"io.github.clash-verge",
				"io.github.clash-verge-rev",
				"clash-verge",
			},
			PrivilegedHelpers:     []string{"io.github.clash-verge", "io.github.clash-verge-rev"},
			UserResiduePaths:      []string{"io.github.clash-verge", "io.github.clash-verge-rev", "verge-mihomo"},
			SystemResiduePaths:    []string{"io.github.clash-verge", "io.github.clash-verge-rev"},
			NetworkExtensionHints: []string{"clash", "verge", "mihomo", "tun"},
			RollbackPolicy:        "quarantine",
			AutomatedRepair:       true,
		},
		{
			ID:                    ProviderClashX,
			DisplayName:           "ClashX",
			ProcessPatterns:       []string{"ClashX", "Clash X", "clash"},
			UTunSignatures:        []string{"198.18.0.1", "198.18.0.0/15", "fdfe:dcba:9876::1"},
			LaunchItems:           []string{"com.west2online.ClashX"},
			PrivilegedHelpers:     []string{"com.west2online.ClashX"},
			UserResiduePaths:      []string{"com.west2online.ClashX"},
			SystemResiduePaths:    []string{"com.west2online.ClashX"},
			NetworkExtensionHints: []string{"clash", "tun"},
			RollbackPolicy:        "quarantine",
			AutomatedRepair:       true,
		},
		{
			ID:                    ProviderMihomoGeneric,
			DisplayName:           "Mihomo / Clash Meta",
			ProcessPatterns:       []string{"mihomo", "clash-meta", "clash"},
			UTunSignatures:        []string{"198.18.0.1", "198.18.0.0/15", "fdfe:dcba:9876::1"},
			NetworkExtensionHints: []string{"mihomo", "clash", "tun"},
			RollbackPolicy:        "quarantine",
			AutomatedRepair:       true,
		},
		{ID: "surge", DisplayName: "Surge", NetworkExtensionHints: []string{"surge"}, DetectOnly: true},
		{ID: "tailscale", DisplayName: "Tailscale", NetworkExtensionHints: []string{"tailscale"}, DetectOnly: true},
		{ID: "cloudflare-warp", DisplayName: "Cloudflare WARP", NetworkExtensionHints: []string{"cloudflare", "warp"}, DetectOnly: true},
		{ID: "adguard", DisplayName: "AdGuard", NetworkExtensionHints: []string{"adguard"}, DetectOnly: true},
		{ID: "little-snitch", DisplayName: "Little Snitch", NetworkExtensionHints: []string{"little snitch", "obdev"}, DetectOnly: true},
		{ID: "proxyman-charles", DisplayName: "Proxyman / Charles", NetworkExtensionHints: []string{"proxyman", "charles"}, DetectOnly: true},
		{ID: "corporate-vpn-generic", DisplayName: "Corporate VPN", NetworkExtensionHints: []string{"vpn", "filter"}, DetectOnly: true},
	}
}

func Facts(residueText []string, tunEvidence []string) []ProviderFact {
	var out []ProviderFact
	text := strings.ToLower(strings.Join(residueText, "\n"))
	for _, provider := range Catalog() {
		fact := ProviderFact{ID: provider.ID, DisplayName: provider.DisplayName, RequiresAdmin: true, SafeToQuarantine: provider.AutomatedRepair}
		for _, evidence := range tunEvidence {
			lower := strings.ToLower(evidence)
			for _, sig := range provider.UTunSignatures {
				if sig != "" && strings.Contains(lower, strings.ToLower(strings.TrimSuffix(sig, "/15"))) {
					fact.StaleTun = true
					fact.Evidence = append(fact.Evidence, evidence)
				}
			}
		}
		for _, token := range append(append([]string{}, provider.LaunchItems...), append(provider.PrivilegedHelpers, append(provider.UserResiduePaths, provider.SystemResiduePaths...)...)...) {
			if token != "" && strings.Contains(text, strings.ToLower(token)) {
				fact.ResidueDetected = true
				fact.Evidence = append(fact.Evidence, token)
			}
		}
		if fact.StaleTun || fact.ResidueDetected {
			fact.Detected = true
			out = append(out, fact)
		}
	}
	return out
}
