package safety

import (
	"strings"
	"testing"
)

// FuzzRedactSensitive guards against panics, runaway regex backtracking,
// and silent failure to strip well-known secret shapes from any byte
// input. Failure = panic OR output that still contains a known plaintext
// secret pattern that RedactSensitive is contracted to redact.
func FuzzRedactSensitive(f *testing.F) {
	seeds := []string{
		"",
		"hello world",
		"token=abcdefghij",
		"Authorization: Bearer xyz123abc",
		"sk-AAAAAAAAAA",
		"http://user:pass@host/path",
		"AKIAIOSFODNN7EXAMPLE",
		"ghp_0123456789abcdefABCDEF0123456789abcd",
		"xoxb-123456789012-abcdefghijkl",
		"AIzaSyDxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx0",
		"eyJhbGciOiJIUzI1NiJ9.payload.signature",
		"-----BEGIN PRIVATE KEY-----abc-----END PRIVATE KEY-----",
		"api_key=zzzzzzzzz",
		"client_secret: 123abcdef",
		strings.Repeat("a", 1000),
		"line1\nline2\nline3",
		"\x00\x01\x02null bytes",
		"!@#$%^&*()=+`~",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := RedactSensitive(s)
		if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") {
			t.Errorf("AWS key not redacted in %q -> %q", s, out)
		}
		if strings.Contains(out, "ghp_0123456789abcdefABCDEF0123456789abcd") {
			t.Errorf("github PAT not redacted in %q -> %q", s, out)
		}
		if strings.Contains(out, "AIzaSyDxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx0") {
			t.Errorf("google API key not redacted in %q -> %q", s, out)
		}
		if strings.Contains(out, "xoxb-123456789012-abcdefghijkl") {
			t.Errorf("slack token not redacted in %q -> %q", s, out)
		}
		if strings.Contains(out, "sk-AAAAAAAAAA") {
			t.Errorf("OpenAI-style key not redacted in %q -> %q", s, out)
		}
		if len(s) > 0 && len(out) > len(s)*64 {
			t.Errorf("output blow-up: in=%d out=%d", len(s), len(out))
		}
	})
}
