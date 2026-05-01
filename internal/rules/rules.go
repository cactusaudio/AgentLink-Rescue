package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type AgentRule struct {
	ID                    string   `json:"id"`
	DisplayName           string   `json:"displayName"`
	Risk                  string   `json:"risk"`
	AutoQuarantineAllowed bool     `json:"autoQuarantineAllowed"`
	Patterns              []string `json:"patterns"`
	Paths                 []string `json:"paths"`
}

type EndpointRules struct {
	AgentTargets []string `json:"agentTargets"`
	HTTPSTargets []string `json:"httpsTargets"`
	RawIPs       []string `json:"rawIPs"`
	DNSNames     []string `json:"dnsNames"`
}

func DefaultEndpointRules() EndpointRules {
	return EndpointRules{
		RawIPs:       []string{"1.1.1.1", "8.8.8.8"},
		DNSNames:     []string{"apple.com", "github.com"},
		HTTPSTargets: []string{"https://www.apple.com/", "https://github.com/"},
		AgentTargets: []string{"https://api.openai.com/", "https://chatgpt.com/", "https://api.anthropic.com/", "https://github.com/", "https://objects.githubusercontent.com/"},
	}
}

func LoadAgentRules(path string) ([]AgentRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []AgentRule
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func LoadEndpointRules(path string) (EndpointRules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultEndpointRules(), err
	}
	out := DefaultEndpointRules()
	if err := json.Unmarshal(data, &out); err != nil {
		return DefaultEndpointRules(), err
	}
	if len(out.RawIPs) == 0 {
		out.RawIPs = DefaultEndpointRules().RawIPs
	}
	if len(out.DNSNames) == 0 {
		out.DNSNames = DefaultEndpointRules().DNSNames
	}
	if len(out.HTTPSTargets) == 0 {
		out.HTTPSTargets = DefaultEndpointRules().HTTPSTargets
	}
	if len(out.AgentTargets) == 0 {
		out.AgentTargets = DefaultEndpointRules().AgentTargets
	}
	return out, nil
}

func MatchRule(rule AgentRule, path string) bool {
	base := strings.ToLower(filepath.Base(path))
	full := strings.ToLower(path)
	for _, pattern := range rule.Patterns {
		p := strings.ToLower(pattern)
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
		if ok, _ := filepath.Match(p, full); ok {
			return true
		}
		if strings.Contains(base, strings.Trim(p, "*")) && strings.Contains(p, "*") {
			return true
		}
	}
	return false
}
