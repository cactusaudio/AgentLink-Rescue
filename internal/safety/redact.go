package safety

import (
	"net/url"
	"regexp"
	"strings"
)

var tokenPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`(?i)(token=)[^&\s]+`), "${1}REDACTED"},
	{regexp.MustCompile(`(?i)(access_token=)[^&\s]+`), "${1}REDACTED"},
	{regexp.MustCompile(`(?i)(password=)[^&\s]+`), "${1}REDACTED"},
	{regexp.MustCompile(`(?i)(passwd=)[^&\s]+`), "${1}REDACTED"},
	{regexp.MustCompile(`(?i)(Authorization:\s*)(Bearer\s+)?[^\s]+`), "${1}REDACTED"},
	{regexp.MustCompile(`(?i)sk-[A-Za-z0-9._-]{8,}`), "REDACTED"},
	// v0.3.2 hardening (R2/R3: 100% model-facing secret redaction). The
	// model-facing diagnosis graph routes ALL free-text through
	// RedactSensitive (diagnosisgraph/graph.go:109); the v0.3.2 executor
	// sandbox proved AWS / GitHub / Slack / Google / bare-JWT / PEM /
	// generic api-key secrets bypassed the 6 narrow patterns above.
	// Additive, well-anchored, specific secret shapes — NO existing
	// coverage removed (redact_test.go stays green).
	{regexp.MustCompile(`(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`), "REDACTED"},
	{regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`), "REDACTED"},
	{regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{8,}`), "REDACTED"},
	{regexp.MustCompile(`AIza[0-9A-Za-z_\-]{20,}`), "REDACTED"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{4,}\.[A-Za-z0-9_\-]{3,}\.[A-Za-z0-9_\-]{2,}`), "REDACTED"},
	{regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._\-]{6,}`), "Bearer REDACTED"},
	{regexp.MustCompile(`-----BEGIN[ A-Z0-9]*PRIVATE KEY-----[A-Za-z0-9+/=\s]*?(-----END[ A-Z0-9]*PRIVATE KEY-----)?`), "REDACTED"},
	{regexp.MustCompile(`(?i)\b(api[_-]?key|client[_-]?secret|aws_secret_access_key)\b\s*[=:]\s*[^\s&"']+`), "${1}=REDACTED"},
}

func RedactURLCredentials(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err == nil && u.User != nil && u.Host != "" {
		u.User = url.User("REDACTED")
		return u.String()
	}
	at := strings.LastIndex(raw, "@")
	scheme := strings.Index(raw, "://")
	if at > -1 && scheme > -1 && at > scheme+3 {
		return raw[:scheme+3] + "REDACTED@" + raw[at+1:]
	}
	return raw
}

func RedactTokens(s string) string {
	out := s
	for _, p := range tokenPatterns {
		out = p.re.ReplaceAllString(out, p.repl)
	}
	return out
}

func RedactSensitive(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		line = RedactURLCredentials(line)
		line = RedactTokens(line)
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func RedactProxyValue(s string) string {
	return RedactSensitive(RedactURLCredentials(s))
}
