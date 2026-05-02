package report

import (
	"encoding/json"
	"testing"

	"cactus-agentlink-rescue/internal/diagnose"
)

func TestReportJSONMarshalling(t *testing.T) {
	r := diagnose.DiagnosticReport{SchemaVersion: 1, ToolVersion: "0.3.0", Classifications: []string{"OK"}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty JSON")
	}
}
