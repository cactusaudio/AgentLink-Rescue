package planner

import (
	"encoding/json"
	"testing"
)

// FuzzDecisionDecode asserts the JSON decode path tolerates any byte
// payload without panicking. A planner-decision JSON may arrive from a
// local model and must never crash the host process.
func FuzzDecisionDecode(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"schemaVersion":1,"intent":"repair","confidence":0.6,"selectedRecipe":{"id":"x","params":{"a":"b"}}}`),
		[]byte(`{"schemaVersion":99}`),
		[]byte(`null`),
		[]byte(`[]`),
		[]byte(`{`),
		[]byte(`{"intent":"\x00"}`),
		[]byte(`{"confidence": 1e308}`),
		[]byte(`{"selectedRecipe":{"params":{"k":` + string(make([]byte, 1000)) + `}}}`),
		[]byte(""),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var d Decision
		_ = json.Unmarshal(data, &d)
	})
}
