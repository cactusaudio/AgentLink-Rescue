package brain

import (
	"encoding/json"
	"os"
)

const DefaultModelID = "qwen3-4b-instruct-2507-q4km"

type ModelManifest struct {
	SchemaVersion      int     `json:"schemaVersion"`
	ID                 string  `json:"id"`
	Family             string  `json:"family"`
	Model              string  `json:"model"`
	Quant              string  `json:"quant"`
	Format             string  `json:"format"`
	Filename           string  `json:"filename"`
	Repo               string  `json:"repo"`
	DownloadURL        string  `json:"downloadURL"`
	SHA256             string  `json:"sha256"`
	SizeBytesApprox    int64   `json:"sizeBytesApprox"`
	License            string  `json:"license"`
	PromptFormat       string  `json:"promptFormat"`
	DefaultContext     int     `json:"defaultContext"`
	DefaultTemperature float64 `json:"defaultTemperature"`
	DefaultMaxTokens   int     `json:"defaultMaxTokens"`
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
		Family:             "qwen",
		Model:              "Qwen3-4B-Instruct-2507",
		Quant:              "Q4_K_M",
		Format:             "GGUF",
		Filename:           "Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf",
		Repo:               "bartowski/Qwen_Qwen3-4B-Instruct-2507-GGUF",
		DownloadURL:        "https://huggingface.co/bartowski/Qwen_Qwen3-4B-Instruct-2507-GGUF/resolve/main/Qwen_Qwen3-4B-Instruct-2507-Q4_K_M.gguf?download=true",
		SHA256:             "2fde00ce69dd4899c70d020845e2638353015bba0fdf161b3eb965f2bca4464e",
		SizeBytesApprox:    2500000000,
		License:            "Apache-2.0 via base model",
		PromptFormat:       "qwen-chatml",
		DefaultContext:     8192,
		DefaultTemperature: 0.0,
		DefaultMaxTokens:   1024,
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
