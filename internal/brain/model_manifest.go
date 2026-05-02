package brain

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultModelID = "gemma-4-e4b-it-q4km"
const DefaultModelFilename = "gemma-4-E4B-it-Q4_K_M.gguf"

type ModelManifest struct {
	SchemaVersion      int      `json:"schemaVersion"`
	ID                 string   `json:"id"`
	Family             string   `json:"family"`
	Generation         string   `json:"generation,omitempty"`
	Model              string   `json:"model"`
	Architecture       string   `json:"architecture,omitempty"`
	EffectiveParams    string   `json:"effectiveParameters,omitempty"`
	Quant              string   `json:"quant"`
	Format             string   `json:"format"`
	Filename           string   `json:"filename"`
	Repo               string   `json:"repo"`
	DownloadURL        string   `json:"downloadURL"`
	SHA256             string   `json:"sha256"`
	SizeBytesApprox    int64    `json:"sizeBytesApprox"`
	License            string   `json:"license"`
	PromptFormat       string   `json:"promptFormat"`
	DefaultContext     int      `json:"defaultContext"`
	DefaultTemperature float64  `json:"defaultTemperature"`
	DefaultMaxTokens   int      `json:"defaultMaxTokens"`
	JSONOnlyPolicy     string   `json:"jsonOnlyPolicy,omitempty"`
	RecommendedFor     []string `json:"recommendedFor,omitempty"`
}

type ManifestLock struct {
	SchemaVersion       int    `json:"schemaVersion"`
	CreatedAt           string `json:"createdAt"`
	DefaultBrainProfile string `json:"defaultBrainProfile"`
	Model               struct {
		Profile   string `json:"profile"`
		Path      string `json:"path"`
		Filename  string `json:"filename"`
		SHA256    string `json:"sha256"`
		SizeBytes int64  `json:"sizeBytes"`
	} `json:"model"`
	Runtime struct {
		Backend string `json:"backend"`
		Path    string `json:"path"`
		Binary  string `json:"binary"`
		SHA256  string `json:"sha256"`
		Arch    string `json:"arch"`
	} `json:"runtime"`
}

type RuntimeManifest struct {
	SchemaVersion           int      `json:"schemaVersion"`
	ID                      string   `json:"id"`
	Backend                 string   `json:"backend"`
	Source                  string   `json:"source"`
	Platforms               []string `json:"platforms"`
	RequiredBinaries        []string `json:"requiredBinaries"`
	OptionalBinaries        []string `json:"optionalBinaries"`
	RequiredSharedLibraries bool     `json:"requiredSharedLibraries"`
}

func DefaultModelManifest() ModelManifest {
	return ModelManifest{
		SchemaVersion:      1,
		ID:                 DefaultModelID,
		Family:             "gemma",
		Generation:         "gemma-4",
		Model:              "gemma-4-E4B-it",
		Architecture:       "dense-effective",
		EffectiveParams:    "4.5B",
		Quant:              "Q4_K_M",
		Format:             "GGUF",
		Filename:           DefaultModelFilename,
		Repo:               "unsloth/gemma-4-E4B-it-GGUF",
		DownloadURL:        "https://huggingface.co/unsloth/gemma-4-E4B-it-GGUF/resolve/main/gemma-4-E4B-it-Q4_K_M.gguf?download=true",
		SHA256:             "",
		SizeBytesApprox:    5070000000,
		License:            "Apache-2.0",
		PromptFormat:       "gemma4",
		DefaultContext:     8192,
		DefaultTemperature: 0.0,
		DefaultMaxTokens:   1024,
		JSONOnlyPolicy:     "extract_first_json_object",
		RecommendedFor:     []string{"16gb-mac", "brain-default", "offline-planner", "agentlink"},
	}
}

func LoadModelManifest(path string) (ModelManifest, error) {
	var m ModelManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(data, &m)
	return m, err
}

func ResolveModelManifest(home string, base ModelManifest) ModelManifest {
	if base.ID == "" {
		base = DefaultModelManifest()
	}
	if base.SHA256 != "" {
		return base
	}
	for _, path := range manifestLockCandidates(home) {
		lock, err := LoadManifestLock(path)
		if err != nil {
			continue
		}
		if lock.DefaultBrainProfile != "" && lock.DefaultBrainProfile != base.ID {
			continue
		}
		if lock.Model.Profile != "" && lock.Model.Profile != base.ID {
			continue
		}
		if lock.Model.Filename != "" && lock.Model.Filename != base.Filename {
			continue
		}
		if lock.Model.SHA256 != "" {
			base.SHA256 = lock.Model.SHA256
		}
		if lock.Model.SizeBytes > 0 {
			base.SizeBytesApprox = lock.Model.SizeBytes
		}
		return base
	}
	return base
}

func LoadManifestLock(path string) (ManifestLock, error) {
	var lock ManifestLock
	data, err := os.ReadFile(path)
	if err != nil {
		return lock, err
	}
	err = json.Unmarshal(data, &lock)
	return lock, err
}

func manifestLockCandidates(home string) []string {
	var out []string
	if root := packageRoot(); root != "" {
		out = append(out, filepath.Join(root, "assets", "manifests", "manifest.lock.json"))
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if home != "" {
		out = append(out, filepath.Join(brainHome(home), "manifests", "manifest.lock.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		out = append(out, filepath.Join(cwd, "assets", "manifests", "manifest.lock.json"))
	}
	return out
}
