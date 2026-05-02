package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Catalog struct {
	SchemaVersion int         `json:"schemaVersion"`
	Installers    []Installer `json:"installers"`
}

type Installer struct {
	ID            string       `json:"id"`
	DisplayName   string       `json:"displayName"`
	Risk          string       `json:"risk"`
	DefaultMethod string       `json:"defaultMethod"`
	Methods       []Method     `json:"methods"`
	Verify        []VerifySpec `json:"verify"`
	Dependencies  []Dependency `json:"dependencies"`
	OfficialDocs  []string     `json:"officialDocs"`
	Warnings      []string     `json:"warnings"`
	Notes         []string     `json:"notes"`
}

type Method struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	RequiresNetwork bool     `json:"requiresNetwork"`
	RequiresAdmin   bool     `json:"requiresAdmin"`
	Command         []string `json:"command"`
	OfficialURL     string   `json:"officialURL"`
}

type VerifySpec struct {
	Type    string   `json:"type"`
	Path    string   `json:"path,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type Dependency struct {
	Command        string `json:"command"`
	Required       bool   `json:"required"`
	MinimumVersion string `json:"minimumVersion,omitempty"`
}

func FindCatalogPath() string {
	candidates := []string{}
	if root := packageRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "assets", "installers", "catalog.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "assets", "installers", "catalog.json"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join("assets", "installers", "catalog.json")
}

func LoadCatalog(path string) (Catalog, error) {
	if path == "" {
		path = FindCatalogPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return Catalog{}, err
	}
	if c.SchemaVersion != 1 {
		return Catalog{}, fmt.Errorf("unsupported installer catalog schema: %d", c.SchemaVersion)
	}
	seen := map[string]bool{}
	for _, in := range c.Installers {
		if in.ID == "" {
			return Catalog{}, fmt.Errorf("installer id is required")
		}
		if seen[in.ID] {
			return Catalog{}, fmt.Errorf("duplicate installer id: %s", in.ID)
		}
		seen[in.ID] = true
		if err := validateInstaller(in); err != nil {
			return Catalog{}, err
		}
	}
	return c, nil
}

func validateInstaller(in Installer) error {
	if len(in.Methods) == 0 {
		return fmt.Errorf("%s has no install methods", in.ID)
	}
	methodIDs := map[string]bool{}
	for _, m := range in.Methods {
		if m.ID == "" || m.Type == "" {
			return fmt.Errorf("%s has invalid method", in.ID)
		}
		methodIDs[m.ID] = true
		if err := validateMethodCommand(in.ID, m); err != nil {
			return err
		}
	}
	if in.DefaultMethod != "" && !methodIDs[in.DefaultMethod] {
		return fmt.Errorf("%s default method %s not found", in.ID, in.DefaultMethod)
	}
	return nil
}

func validateMethodCommand(id string, m Method) error {
	if len(m.Command) == 0 {
		if m.Type == "embedded_dmg" {
			return nil
		}
		return fmt.Errorf("%s method %s has empty command", id, m.ID)
	}
	joined := strings.Join(m.Command, " ")
	forbidden := []string{"curl |", "curl|", "bash -c", "sh -c", "sudo", "eval"}
	for _, f := range forbidden {
		if strings.Contains(joined, f) {
			return fmt.Errorf("%s method %s contains forbidden command surface: %s", id, m.ID, f)
		}
	}
	switch id {
	case "codex-cli":
		if m.Type == "npm" && !containsExact(m.Command, "@openai/codex") {
			return fmt.Errorf("codex-cli npm method must use @openai/codex")
		}
		if m.Type == "homebrew" && !containsExact(m.Command, "codex") {
			return fmt.Errorf("codex-cli homebrew method must use codex")
		}
	case "claude-code-cli":
		if !containsExact(m.Command, "@anthropic-ai/claude-code") {
			return fmt.Errorf("claude-code-cli must use @anthropic-ai/claude-code")
		}
	case "gemini-cli":
		if m.Type == "npm" && !containsExact(m.Command, "@google/gemini-cli") {
			return fmt.Errorf("gemini-cli npm method must use @google/gemini-cli")
		}
		if m.Type == "homebrew" && !containsExact(m.Command, "gemini-cli") {
			return fmt.Errorf("gemini-cli homebrew method must use gemini-cli")
		}
	case "codex-app":
		if m.OfficialURL != "" && !strings.HasPrefix(m.OfficialURL, "https://developers.openai.com/") {
			return fmt.Errorf("codex-app official URL must be on developers.openai.com")
		}
	}
	return nil
}

func containsExact(items []string, want string) bool {
	for _, it := range items {
		if it == want {
			return true
		}
	}
	return false
}

func (c Catalog) Find(id string) (Installer, bool) {
	for _, in := range c.Installers {
		if in.ID == id {
			return in, true
		}
	}
	return Installer{}, false
}

func (in Installer) Method(id string) (Method, bool) {
	if id == "" {
		id = in.DefaultMethod
	}
	for _, m := range in.Methods {
		if m.ID == id {
			return m, true
		}
	}
	return Method{}, false
}

func packageRoot() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if filepath.Base(dir) == "bin" {
			if root := rootIfPackage(filepath.Dir(dir)); root != "" {
				return root
			}
		}
		if root := rootIfPackage(dir); root != "" {
			return root
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for i := 0; i < 5; i++ {
			if root := rootIfPackage(cwd); root != "" {
				return root
			}
			next := filepath.Dir(cwd)
			if next == cwd {
				break
			}
			cwd = next
		}
	}
	return ""
}

func rootIfPackage(dir string) string {
	if dir == "" {
		return ""
	}
	if exists(filepath.Join(dir, "assets")) && exists(filepath.Join(dir, "recipes")) {
		return dir
	}
	if exists(filepath.Join(dir, "assets", "manifests")) && exists(filepath.Join(dir, "bin")) {
		return dir
	}
	return ""
}

func RuntimeArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	default:
		return runtime.GOARCH
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
