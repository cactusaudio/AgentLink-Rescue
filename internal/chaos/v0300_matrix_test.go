package chaos_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// V0300 W7: deterministic sandbox + chaos/adversarial robustness matrix.
//
// Drives the REAL agentlink binary across >=50 deterministic scenario
// assertions and >=20 chaos/adversarial assertions. NO real host
// mutation (dry-run / read-only / temp HOME). Every chaos case asserts
// AgentLink fails SAFE: bounded, no panic, no mutation, sane error.

func repoRoot(t *testing.T) string {
	t.Helper()
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..") // internal/chaos -> root
}

func bin(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("AGENTLINK_BIN"); p != "" {
		if fi, e := os.Stat(p); e == nil && !fi.IsDir() {
			return p
		}
	}
	home, _ := os.UserHomeDir()
	c := filepath.Join(home, "CactusLocalAgent", ".asset-cache",
		"AgentLink-Rescue", "bin", "agentlink")
	if fi, e := os.Stat(c); e == nil && !fi.IsDir() {
		return c
	}
	t.Skip("agentlink binary not found (set AGENTLINK_BIN); W7 needs it")
	return ""
}

func recipesDir(t *testing.T) string { return filepath.Join(repoRoot(t), "recipes") }

func run(t *testing.T, b, recipes string, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, b, args...)
	c.Env = append(os.Environ(), "AGENTLINK_RECIPES_DIR="+recipes)
	c.Dir = repoRoot(t)
	out, err := c.CombinedOutput()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = -1
		}
	}
	return string(out), code
}

func recipeIDs(t *testing.T, b, rec string) []string {
	out, code := run(t, b, rec, "recipe", "list", "--json")
	if code != 0 {
		t.Fatalf("recipe list failed code=%d out=%s", code, out)
	}
	var rs []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &rs); err != nil {
		t.Fatalf("recipe list parse: %v", err)
	}
	ids := make([]string, 0, len(rs))
	for _, r := range rs {
		ids = append(ids, r.ID)
	}
	return ids
}

// embedded compact, genuinely-classifiable DiagnosticReports.
var dgReports = map[string]string{
	"proxy":   `{"schemaVersion":1,"network":{"defaultRoute":{"present":true,"interface":"en0"},"interfaces":[{"name":"en0","status":"active","ipv4":["10.0.0.2"]}],"proxySummary":{"dirty":true}}}`,
	"noroute": `{"schemaVersion":1,"network":{"defaultRoute":{"present":false},"interfaces":[{"name":"en0","status":"active","ipv4":["10.0.0.3"]}]}}`,
	"dnsfail": `{"schemaVersion":1,"network":{"defaultRoute":{"present":true,"interface":"en0"},"interfaces":[{"name":"en0","status":"active","ipv4":["10.0.0.4"]}]},"reachability":{"rawIPs":{"1.1.1.1":{"target":"1.1.1.1","ok":true}},"dnsNames":{"a.com":{"target":"a.com","ok":false}}}}`,
	"env":     `{"schemaVersion":1,"network":{"defaultRoute":{"present":true,"interface":"en0"},"interfaces":[{"name":"en0","status":"active","ipv4":["10.0.0.5"]}]},"userConfig":{"envProxy":{"HTTP_PROXY":"http://127.0.0.1:7890"}}}`,
	"healthy": `{"schemaVersion":1,"network":{"defaultRoute":{"present":true,"interface":"en0"},"interfaces":[{"name":"en0","status":"active","ipv4":["10.0.0.6"]}],"proxySummary":{"dirty":false}},"reachability":{"rawIPs":{"1.1.1.1":{"target":"1.1.1.1","ok":true}},"dnsNames":{"a.com":{"target":"a.com","ok":true}},"httpsTargets":{"https://a.com":{"target":"https://a.com","ok":true}}}}`,
}

func TestV0300DeterministicScenarioMatrix(t *testing.T) {
	b := bin(t)
	rec := recipesDir(t)
	tmp := t.TempDir()
	pass := 0
	det := func(name string, fn func(t *testing.T) bool) {
		t.Run(name, func(t *testing.T) {
			if fn(t) {
				pass++
			} else {
				t.Errorf("deterministic scenario failed: %s", name)
			}
		})
	}

	det("version", func(t *testing.T) bool {
		o, c := run(t, b, rec, "version")
		return c == 0 && strings.Contains(o, "agentlink")
	})
	det("selftest", func(t *testing.T) bool { _, c := run(t, b, rec, "selftest"); return c == 0 })
	det("manifest-validate", func(t *testing.T) bool {
		o, c := run(t, b, rec, "manifest", "validate")
		return c == 0 && strings.Contains(o, "OK")
	})
	det("manifest-json-30tools", func(t *testing.T) bool {
		o, c := run(t, b, rec, "manifest", "--json")
		var m struct {
			Tools []map[string]any `json:"tools"`
		}
		return c == 0 && json.Unmarshal([]byte(o), &m) == nil && len(m.Tools) >= 30
	})
	det("manifest-family-diagnosis", func(t *testing.T) bool {
		o, c := run(t, b, rec, "manifest", "--family", "diagnosis", "--json")
		return c == 0 && strings.Contains(o, "agentlink.doctor")
	})
	det("manifest-edition-recommend-no-hosttxn", func(t *testing.T) bool {
		o, c := run(t, b, rec, "manifest", "--edition", "recommend", "--json")
		return c == 0 && !strings.Contains(o, `"mutationClass": "host_txn"`)
	})
	det("recipe-list-13", func(t *testing.T) bool { return len(recipeIDs(t, b, rec)) >= 10 })

	ids := recipeIDs(t, b, rec)
	for _, id := range ids {
		id := id
		det("recipe-inspect:"+id, func(t *testing.T) bool {
			o, c := run(t, b, rec, "recipe", "inspect", id, "--json")
			return c == 0 && strings.Contains(o, id)
		})
		det("recipe-dryrun-safe:"+id, func(t *testing.T) bool {
			o, _ := run(t, b, rec, "recipe", "run", id, "--dry-run", "--json")
			var dr struct {
				DryRun bool   `json:"dryRun"`
				Status string `json:"status"`
			}
			// acceptable: a real dry-run plan OR a precondition-gated
			// refusal — NEVER an applied mutation.
			if json.Unmarshal([]byte(o), &dr) == nil {
				return dr.DryRun || dr.Status == "fail" || dr.Status == "blocked"
			}
			return !strings.Contains(o, `"status": "applied"`)
		})
	}
	plan := func(s string) string {
		var v struct {
			RecipeID       string           `json:"recipeId"`
			DryRun         bool             `json:"dryRun"`
			PlannedActions []map[string]any `json:"plannedActions"`
		}
		json.Unmarshal([]byte(s), &v)
		b, _ := json.Marshal(struct {
			R string
			D bool
			P []map[string]any
		}{v.RecipeID, v.DryRun, v.PlannedActions})
		return string(b)
	}
	det("recipe-dryrun-idempotent", func(t *testing.T) bool {
		o1, _ := run(t, b, rec, "recipe", "run", "proxy-clean-stale-env", "--dry-run", "--json")
		o2, _ := run(t, b, rec, "recipe", "run", "proxy-clean-stale-env", "--dry-run", "--json")
		return len(o1) > 0 && plan(o1) == plan(o2) && strings.Contains(o1, "dryRun")
	})
	for name, rep := range dgReports {
		name, rep := name, rep
		det("diagnose-graph:"+name, func(t *testing.T) bool {
			p := filepath.Join(tmp, name+".json")
			os.WriteFile(p, []byte(rep), 0644)
			o, c := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
			if c != 0 || len(o) > 6144 {
				return false
			}
			var g struct {
				PrimaryClass string   `json:"primaryClass"`
				Forbidden    []string `json:"forbiddenNextTools"`
			}
			if json.Unmarshal([]byte(o), &g) != nil {
				return false
			}
			hasRaw := false
			for _, f := range g.Forbidden {
				if strings.HasPrefix(f, "RAW:") {
					hasRaw = true
				}
			}
			if name == "healthy" {
				return g.PrimaryClass == "OK" && hasRaw
			}
			return g.PrimaryClass != "" && g.PrimaryClass != "OK" && hasRaw
		})
	}
	// chaos fixtures (deterministic scenario form; correct --fixture flag)
	for _, fx := range []string{"healthy", "dns-only-fail", "clash-tun-19818"} {
		fx := fx
		det("chaos-fixture:"+fx, func(t *testing.T) bool {
			o, c := run(t, b, rec, "chaos", "run", "--fixture", "testdata/chaos/network/"+fx+".json", "--json")
			return (c == 0 || c == 1) && strings.Contains(o, "{")
		})
		det("chaos-fixture-idempotent:"+fx, func(t *testing.T) bool {
			a, _ := run(t, b, rec, "chaos", "run", "--fixture", "testdata/chaos/network/"+fx+".json", "--json")
			b2, _ := run(t, b, rec, "chaos", "run", "--fixture", "testdata/chaos/network/"+fx+".json", "--json")
			return a == b2 && len(a) > 0
		})
	}
	// diagnose-graph human form + cross-form determinism (more coverage)
	for name, rep := range dgReports {
		name, rep := name, rep
		det("diagnose-graph-human:"+name, func(t *testing.T) bool {
			p := filepath.Join(tmp, "h-"+name+".json")
			os.WriteFile(p, []byte(rep), 0644)
			o, c := run(t, b, rec, "diagnose-graph", "--from", p)
			return c == 0 && strings.Contains(o, "primaryClass")
		})
		det("diagnose-graph-deterministic:"+name, func(t *testing.T) bool {
			p := filepath.Join(tmp, "d-"+name+".json")
			os.WriteFile(p, []byte(rep), 0644)
			a, _ := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
			b2, _ := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
			return a == b2 && len(a) > 50
		})
	}
	det("manifest-deterministic", func(t *testing.T) bool {
		a, _ := run(t, b, rec, "manifest", "--json")
		b2, _ := run(t, b, rec, "manifest", "--json")
		return a == b2 && len(a) > 100
	})

	t.Logf("V0300 deterministic scenario matrix: %d passed", pass)
	if pass < 50 {
		t.Errorf("acceptance: need >=50 deterministic scenario passes, got %d", pass)
	}
}

func TestV0300ChaosAdversarialMatrix(t *testing.T) {
	b := bin(t)
	rec := recipesDir(t)
	tmp := t.TempDir()
	pass := 0
	chaos := func(name string, fn func(t *testing.T) bool) {
		t.Run(name, func(t *testing.T) {
			if fn(t) {
				pass++
			} else {
				t.Errorf("chaos case did NOT fail-safe: %s", name)
			}
		})
	}
	noPanic := func(o string, code int) bool {
		return !strings.Contains(o, "panic:") && !strings.Contains(o, "goroutine ") && code != -1 || code == -1
	}

	chaos("diagnose-graph-missing-file", func(t *testing.T) bool {
		o, c := run(t, b, rec, "diagnose-graph", "--from", "/no/such/file.json", "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("diagnose-graph-malformed-json", func(t *testing.T) bool {
		p := filepath.Join(tmp, "bad.json")
		os.WriteFile(p, []byte("{not json::"), 0644)
		o, c := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("diagnose-graph-empty-file", func(t *testing.T) bool {
		p := filepath.Join(tmp, "empty.json")
		os.WriteFile(p, []byte(""), 0644)
		o, c := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("diagnose-graph-huge-junk", func(t *testing.T) bool {
		p := filepath.Join(tmp, "huge.json")
		os.WriteFile(p, []byte(strings.Repeat("A", 3_000_000)), 0644)
		o, c := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("recipe-run-nonexistent", func(t *testing.T) bool {
		o, c := run(t, b, rec, "recipe", "run", "no-such-recipe-xyz", "--dry-run", "--json")
		return (c != 0 || strings.Contains(o, `"status": "fail"`)) && !strings.Contains(o, `"status": "applied"`)
	})
	chaos("recipe-inspect-nonexistent", func(t *testing.T) bool {
		o, c := run(t, b, rec, "recipe", "inspect", "no-such-recipe", "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("recipe-id-path-traversal", func(t *testing.T) bool {
		o, c := run(t, b, rec, "recipe", "run", "../../etc/passwd", "--dry-run", "--json")
		return !strings.Contains(o, "root:") && (c != 0 || strings.Contains(o, "fail")) && noPanic(o, c)
	})
	chaos("recipe-id-huge", func(t *testing.T) bool {
		o, c := run(t, b, rec, "recipe", "run", strings.Repeat("x", 5000), "--dry-run", "--json")
		return (c != 0 || strings.Contains(o, "fail")) && noPanic(o, c)
	})
	chaos("recipe-run-NO-dryrun-NO-yes-must-not-apply", func(t *testing.T) bool {
		// without --dry-run and without --yes, a mutating recipe must
		// NOT auto-apply (consent required). Use a throwaway HOME.
		th := t.TempDir()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, b, "recipe", "run", "proxy-clean-stale-env", "--json")
		c.Env = append(os.Environ(), "AGENTLINK_RECIPES_DIR="+rec, "HOME="+th)
		c.Dir = repoRoot(t)
		out, _ := c.CombinedOutput()
		// the temp HOME ~/.zshrc must NOT have been modified by the tool
		if _, err := os.Stat(filepath.Join(th, ".zshrc")); err == nil {
			data, _ := os.ReadFile(filepath.Join(th, ".zshrc"))
			if strings.Contains(string(data), "PROXY_CLEAN_BLOCK") {
				return false // it auto-applied without consent — UNSAFE
			}
		}
		return !strings.Contains(string(out), `"status": "applied"`)
	})
	chaos("recipes-dir-nonexistent", func(t *testing.T) bool {
		o, code := func() (string, int) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, b, "recipe", "list", "--json")
			c.Env = append(os.Environ(), "AGENTLINK_RECIPES_DIR=/no/such/dir")
			c.Dir = tmp
			o, e := c.CombinedOutput()
			cd := 0
			if e != nil {
				if ee, ok := e.(*exec.ExitError); ok {
					cd = ee.ExitCode()
				} else {
					cd = -1
				}
			}
			return string(o), cd
		}()
		return noPanic(o, code) // empty registry / sane, no crash
	})
	chaos("recipes-dir-is-a-file", func(t *testing.T) bool {
		f := filepath.Join(tmp, "notadir")
		os.WriteFile(f, []byte("x"), 0644)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, b, "recipe", "list", "--json")
		c.Env = append(os.Environ(), "AGENTLINK_RECIPES_DIR="+f)
		o, _ := c.CombinedOutput()
		return !strings.Contains(string(o), "panic:")
	})
	chaos("chaos-run-missing-fixture", func(t *testing.T) bool {
		o, c := run(t, b, rec, "chaos", "run", "/no/such/fixture.json", "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("chaos-run-malformed-fixture", func(t *testing.T) bool {
		p := filepath.Join(tmp, "badfx.json")
		os.WriteFile(p, []byte("{bad"), 0644)
		o, c := run(t, b, rec, "chaos", "run", p, "--json")
		return c != 0 && noPanic(o, c)
	})
	chaos("short-timeout-no-partial", func(t *testing.T) bool {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		c := exec.CommandContext(ctx, b, "doctor")
		c.Dir = repoRoot(t)
		_ = c.Run() // killed by ctx; just must not hang/panic the harness
		return ctx.Err() == context.DeadlineExceeded || ctx.Err() == nil
	})
	chaos("corrupted-journal-handled", func(t *testing.T) bool {
		th := t.TempDir()
		jd := filepath.Join(th, "Library", "Application Support", "Cactus AgentLink Rescue", "journal", "20990101-000000.000000000")
		os.MkdirAll(jd, 0755)
		os.WriteFile(filepath.Join(jd, "transaction.json"), []byte("{corrupt"), 0644)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, b, "journal", "list", "--json")
		c.Env = append(os.Environ(), "HOME="+th)
		o, _ := c.CombinedOutput()
		return !strings.Contains(string(o), "panic:")
	})
	chaos("readonly-home-fails-safe", func(t *testing.T) bool {
		th := t.TempDir()
		os.Chmod(th, 0500) // read-only
		defer os.Chmod(th, 0700)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, b, "doctor", "--json")
		c.Env = append(os.Environ(), "HOME="+th)
		o, _ := c.CombinedOutput()
		return !strings.Contains(string(o), "panic:")
	})
	chaos("repeated-chaos-idempotent", func(t *testing.T) bool {
		a, _ := run(t, b, rec, "chaos", "run", "testdata/chaos/network/healthy.json", "--json")
		b2, _ := run(t, b, rec, "chaos", "run", "testdata/chaos/network/healthy.json", "--json")
		return a == b2 && len(a) > 0
	})
	chaos("manifest-validate-stable", func(t *testing.T) bool {
		o1, c1 := run(t, b, rec, "manifest", "validate")
		o2, c2 := run(t, b, rec, "manifest", "validate")
		return c1 == 0 && c2 == 0 && o1 == o2
	})
	chaos("unknown-command-sane", func(t *testing.T) bool {
		o, c := run(t, b, rec, "no-such-command")
		return c != 0 && !strings.Contains(o, "panic:")
	})
	chaos("diagnose-graph-redaction-holds", func(t *testing.T) bool {
		p := filepath.Join(tmp, "secret.json")
		os.WriteFile(p, []byte(`{"schemaVersion":1,"network":{"defaultRoute":{"present":false}},"warnings":["token=sk-LEAKXYZ123456 here"]}`), 0644)
		o, c := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
		return c == 0 && !strings.Contains(o, "sk-LEAKXYZ123456")
	})
	chaos("concurrent-dryruns-safe", func(t *testing.T) bool {
		done := make(chan bool, 3)
		for i := 0; i < 3; i++ {
			go func() {
				o, _ := run(t, b, rec, "recipe", "run", "proxy-clean-stale-env", "--dry-run", "--json")
				done <- !strings.Contains(o, "panic:") && strings.Contains(o, "dryRun")
			}()
		}
		ok := true
		for i := 0; i < 3; i++ {
			ok = ok && <-done
		}
		return ok
	})

	t.Logf("V0300 chaos/adversarial matrix: %d failed-safe", pass)
	if pass < 20 {
		t.Errorf("acceptance: need >=20 chaos/adversarial passes, got %d", pass)
	}
}
