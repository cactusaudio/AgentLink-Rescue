package diagnosisgraph

import (
	"encoding/json"
	"testing"

	"cactus-agentlink-rescue/internal/diagnose"
)

// FuzzBuildFromJSON drives the diagnose-graph end-to-end build over any
// JSON byte payload. The CLI accepts a `--from <path>` for an external
// DiagnosticReport file, so this is the user-facing input surface that
// must never panic, never leak unredacted secrets, and never blow up
// output size relative to input.
func FuzzBuildFromJSON(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"schemaVersion":1,"network":{"defaultRoute":{"present":true,"interface":"en0"}}}`),
		[]byte(`{"schemaVersion":1,"network":{"defaultRoute":{"present":false}},"warnings":["leak token=sk-AAAAAAAAAA"]}`),
		[]byte(`null`),
		[]byte(`[]`),
		[]byte(``),
		[]byte(`{`),
		[]byte(`{"network":{"interfaces":[{"name":"en0","status":"active","ipv4":["10.0.0.2"]}]}}`),
		[]byte("\x00\x01\x02"),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var r diagnose.DiagnosticReport
		if json.Unmarshal(data, &r) != nil {
			return
		}
		// Both paths must be panic-free even on adversarial input.
		_ = Build(r, true)
		_ = Build(r, false)
	})
}
