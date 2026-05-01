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
