package verifier

import "testing"

func TestEnvKeyPresentOptionalWarnsWhenMissing(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")
	res := envKeyPresent(Context{}, map[string]string{"env": "DEEPSEEK_API_KEY", "required": "false"})
	if res.Status != "warn" {
		t.Fatalf("status=%s result=%+v", res.Status, res)
	}
}

func TestEnvKeyPresentRequiredFailsWhenMissing(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "")
	res := envKeyPresent(Context{}, map[string]string{"env": "DEEPSEEK_API_KEY", "required": "true"})
	if res.Status != "fail" {
		t.Fatalf("status=%s result=%+v", res.Status, res)
	}
}

func TestEnvKeyPresentRedactsWhenPresent(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-secret-value")
	res := envKeyPresent(Context{}, map[string]string{"env": "DEEPSEEK_API_KEY", "required": "true"})
	if res.Status != "pass" {
		t.Fatalf("status=%s result=%+v", res.Status, res)
	}
	if res.Evidence != "DEEPSEEK_API_KEY present: REDACTED" {
		t.Fatalf("unexpected evidence: %q", res.Evidence)
	}
}
