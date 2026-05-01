package safety

import (
	"strings"
	"testing"
)

func TestRedactURLCredentials(t *testing.T) {
	got := RedactURLCredentials("http://user:pass@host.local:8080")
	if got != "http://REDACTED@host.local:8080" {
		t.Fatalf("got %q", got)
	}
}

func TestRedactTokens(t *testing.T) {
	got := RedactTokens("Authorization: Bearer abc123 token=secret&x=1 password=hunter2")
	if got == "Authorization: Bearer abc123 token=secret&x=1 password=hunter2" {
		t.Fatalf("not redacted: %q", got)
	}
	if want := "REDACTED"; !strings.Contains(got, want) {
		t.Fatalf("missing redaction: %q", got)
	}
}

func TestRedactAPIKeyShape(t *testing.T) {
	secret := "sk-testsecretvalue12345"
	got := RedactSensitive("key=" + secret)
	if strings.Contains(got, secret) {
		t.Fatalf("API key leaked: %q", got)
	}
}
