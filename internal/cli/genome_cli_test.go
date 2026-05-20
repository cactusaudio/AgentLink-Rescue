package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenomeCLIPhaseCommands(t *testing.T) {
	t.Setenv("AGENTLINK_GENOME_DIR", filepath.Join("..", "..", "agentlink_rescue_network_genome_v0_2"))
	t.Setenv("AGENTLINK_GENOME_INDEX_DIR", t.TempDir())
	run := func(args ...string) string {
		t.Helper()
		var stdout, stderr bytes.Buffer
		if code := Main(args, &stdout, &stderr); code != 0 {
			t.Fatalf("%v code=%d stderr=%s stdout=%s", args, code, stderr.String(), stdout.String())
		}
		return stdout.String()
	}
	if out := run("index", "stats"); !strings.Contains(out, "Cards: 300") {
		t.Fatalf("bad index stats: %s", out)
	}
	if out := run("index", "rebuild", "--json"); !strings.Contains(out, `"cards": 300`) {
		t.Fatalf("bad index rebuild: %s", out)
	}
	if out := run("explain", "--card", "MAC-PROXY-002"); !strings.Contains(out, "MAC-PROXY-002") || !strings.Contains(out, "Shell environment proxy") {
		t.Fatalf("bad explain: %s", out)
	}
	if out := run("diagnose", "--symptom", "browser works but codex fails", "--no-snapshot"); !strings.Contains(out, "MAC-PROXY-002") {
		t.Fatalf("bad diagnose: %s", out)
	}
	if out := run("gate", "--card", "DANTE-CLOCK-001"); !strings.Contains(out, "manual_only") {
		t.Fatalf("bad gate: %s", out)
	}
}

func TestGenomeCLIReportFromSnapshot(t *testing.T) {
	t.Setenv("AGENTLINK_GENOME_DIR", filepath.Join("..", "..", "agentlink_rescue_network_genome_v0_2"))
	dir := t.TempDir()
	redacted := filepath.Join(dir, "redacted")
	if err := os.MkdirAll(filepath.Join(dir, "features"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(redacted, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(redacted, "shell_proxy_env.txt"), []byte("HTTP_PROXY=http://127.0.0.1:7890\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "features", "snapshot_features.json"), []byte(`{"schemaVersion":1,"privacyMode":"standard","features":{"shell_proxy_env_present":"true"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "report.md")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"report", "--format", "markdown", "--snapshot", dir, "--symptom", "browser works but codex fails", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var jsonStdout, jsonStderr bytes.Buffer
	jsonOut := filepath.Join(dir, "report.json")
	code = Main([]string{"report", "--format", "json", "--snapshot", dir, "--symptom", "browser works but codex fails", "--out", jsonOut, "--json"}, &jsonStdout, &jsonStderr)
	if code != 0 {
		t.Fatalf("json code=%d stderr=%s stdout=%s", code, jsonStderr.String(), jsonStdout.String())
	}
	if !strings.Contains(jsonStdout.String(), `"reportPath"`) || !strings.Contains(jsonStdout.String(), jsonOut) {
		t.Fatalf("bad json report status: %s", jsonStdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "AgentLink Rescue Report") || !strings.Contains(string(data), "MAC-PROXY-002") {
		t.Fatalf("bad report:\n%s", data)
	}
}
