package cli

import "testing"

func TestNonDarwinCloudAllowed(t *testing.T) {
	allowed := []struct {
		cmd  string
		args []string
	}{
		{"version", nil},
		{"selftest", nil},
		{"doctor", []string{"--json"}},
		{"manifest", []string{"validate"}},
		{"chaos", []string{"list", "--root", "testdata/chaos", "--json"}},
		{"chaos", []string{"run", "--fixture", "fixture.json", "--json"}},
		{"diagnose", []string{"--json"}},
		{"diagnose-graph", []string{"--from", "report.json", "--json"}},
		{"diagnose-graph", []string{"--from", "-", "--json"}},
		{"diagnose-graph", []string{"--from=report.json", "--json"}},
		{"recipe", []string{"list", "--json"}},
		{"recipe", []string{"inspect", "proxy-clean-stale-env", "--json"}},
		{"recipe", []string{"run", "proxy-clean-stale-env", "--dry-run", "--json"}},
		{"recipe", []string{"run", "proxy-clean-stale-env", "--json"}},
		{"support", []string{"bundle", "--json"}},
		{"package", []string{"doctor", "--json"}},
		{"package", []string{"repair", "--dry-run", "--json"}},
		{"journal", []string{"list", "--json"}},
		{"journal", []string{"inspect", "tx1", "--json"}},
		{"journal", []string{"recover", "--dry", "--json"}},
		{"readiness", []string{"doctor", "--json"}},
		{"readiness", []string{"--json"}},
		{"dev", []string{"doctor", "--json"}},
		{"rollback", []string{"--last", "--dry-run", "--json"}},
		{"last-good", []string{"list", "--json"}},
		{"brain", []string{"doctor", "--json"}},
	}
	for _, tc := range allowed {
		if !nonDarwinCloudAllowed(tc.cmd, tc.args) {
			t.Fatalf("%s %v should be allowed in non-Darwin cloud gate", tc.cmd, tc.args)
		}
	}
}

func TestNonDarwinCloudDenied(t *testing.T) {
	denied := []struct {
		cmd  string
		args []string
	}{
		{"diagnose-graph", []string{"--json"}},
		{"recipe", []string{"run", "proxy-clean-stale-env", "--yes", "--json"}},
		{"package", []string{"repair", "--yes", "--json"}},
		{"journal", []string{"recover", "--yes", "--json"}},
		{"rollback", []string{"--last", "--json"}},
		{"guided", []string{"rescue", "--yes", "--json"}},
		{"orchestrator", []string{"rescue", "--yes", "--json"}},
		{"rescue", []string{"--level", "tun", "--yes"}},
		{"repair", []string{"--target", "proxy", "--yes"}},
		{"restore", []string{"last", "--json"}},
		{"snapshot", nil},
		{"diff", []string{"--snapshot", "x"}},
		{"proxy", []string{"clean", "--yes"}},
		{"keys", []string{"scan"}},
		{"installer", []string{"install"}},
		{"verify", []string{"network", "--json"}},
	}
	for _, tc := range denied {
		if nonDarwinCloudAllowed(tc.cmd, tc.args) {
			t.Fatalf("%s %v should be denied in non-Darwin cloud gate", tc.cmd, tc.args)
		}
	}
}
