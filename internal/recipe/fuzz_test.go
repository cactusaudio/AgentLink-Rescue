package recipe

import (
	"encoding/json"
	"testing"
)

// FuzzRecipeDecode asserts the bundled recipe loader's JSON decode path
// is panic-free for any byte input. Recipes ship inside the package so
// untrusted JSON normally cannot reach this code, but a corrupted asset
// directory must fail gracefully, not crash.
func FuzzRecipeDecode(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"id":"x","risk":"read_only"}`),
		[]byte(`{"patches":[{"type":"unknown"}]}`),
		[]byte(`{"params":{"k":{"type":"string"}}}`),
		[]byte(`{`),
		[]byte(`[]`),
		[]byte(`null`),
		[]byte(``),
		[]byte("\xff\xfe\x00\x00"),
		[]byte(`{"id":"` + string(make([]byte, 5000)) + `"}`),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var r Recipe
		_ = json.Unmarshal(data, &r)
	})
}
