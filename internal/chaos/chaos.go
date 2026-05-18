package chaos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/system"
)

type Fixture struct {
	SchemaVersion int                       `json:"schemaVersion"`
	ID            string                    `json:"id"`
	Category      string                    `json:"category"`
	Description   string                    `json:"description"`
	Diagnostic    diagnose.DiagnosticReport `json:"diagnostic"`
	Expected      Expected                  `json:"expected"`
}

type Expected struct {
	FailureClass           string `json:"failureClass,omitempty"`
	RecommendedRepair      string `json:"recommendedRepair,omitempty"`
	RecommendedRepairLevel string `json:"recommendedRepairLevel,omitempty"`
}

type Report struct {
	SchemaVersion          int      `json:"schemaVersion"`
	ToolVersion            string   `json:"toolVersion"`
	FixturePath            string   `json:"fixturePath,omitempty"`
	ID                     string   `json:"id,omitempty"`
	Status                 string   `json:"status"`
	FailureClasses         []string `json:"failureClasses,omitempty"`
	RecommendedRepair      string   `json:"recommendedRepair,omitempty"`
	RecommendedRepairLevel string   `json:"recommendedRepairLevel,omitempty"`
	Warnings               []string `json:"warnings,omitempty"`
}

func List(root string) ([]string, error) {
	if root == "" {
		root = filepath.Join("testdata", "chaos")
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out, err
}

func Run(path string) Report {
	rep := Report{SchemaVersion: 1, ToolVersion: system.Version, FixturePath: path, Status: "passed"}
	data, err := os.ReadFile(path)
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	rep.ID = fixture.ID
	diag := fixture.Diagnostic
	classify.Apply(&diag)
	tun := diagnose.DiagnoseTun(diag)
	rep.FailureClasses = append([]string(nil), diag.Classifications...)
	rep.RecommendedRepair = tun.RecommendedRepair
	rep.RecommendedRepairLevel = diag.RecommendedRepairLevel
	if fixture.Expected.FailureClass != "" && !contains(diag.Classifications, fixture.Expected.FailureClass) {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, "expected failure class missing: "+fixture.Expected.FailureClass)
	}
	if fixture.Expected.RecommendedRepair != "" && fixture.Expected.RecommendedRepair != tun.RecommendedRepair {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, "expected recommended repair "+fixture.Expected.RecommendedRepair+", got "+tun.RecommendedRepair)
	}
	if fixture.Expected.RecommendedRepairLevel != "" && fixture.Expected.RecommendedRepairLevel != diag.RecommendedRepairLevel {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, "expected recommended repair level "+fixture.Expected.RecommendedRepairLevel+", got "+diag.RecommendedRepairLevel)
	}
	return rep
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
