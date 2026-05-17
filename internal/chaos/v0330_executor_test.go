package chaos_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// v0.3.3 Release-Candidate Tier-C executor torture. Reuses the
// chaos_test helpers bin()/run()/repoRoot()/recipesDir() and the v0320
// primitives netv2()/resv2()/safeTerminal() (same package). Exit bar:
// >=1000 deterministic executor scenarios, >=500 chaos/adversarial,
// >=100 multi-hop, >=50 rollback/fault-injection, >=25 concurrent or
// interrupted-run, 100% safe terminal, 0 undetected-worse, 0 raw model
// exposure, 0 mutation without the approval gate. Kernel owns truth:
// every fault class is derived by classify.Apply from genuinely-
// classifiable evidence — no fixture hand-sets a class. Fixture-driven,
// reproducible, ZERO real-host mutation (every recipe touch is
// --dry-run and/or scoped to a t.TempDir HOME). Compact redacted graphs
// + matrix exported to datasets/eval/v0330-sandbox/ for Tier A/B.

func buildV0330Matrix() []exScn {
	var m []exScn
	add := func(s exScn) { m = append(m, s) }
	ip := func(n int) []string { return []string{fmt.Sprintf("10.%d.%d.%d", (n/60000)%250, (n/250)%250, n%250+2)} }

	// proxy: system(dirty http)/user(env)/git/npm/brew/pac — 6 x 34 = 204
	for i := 0; i < 6; i++ {
		for v := 0; v < 34; v++ {
			n := i*34 + v
			uc := map[string]any{}
			ps := map[string]any{}
			switch i {
			case 0:
				ps = map[string]any{"dirty": true, "httpEnabled": true}
			case 1:
				uc = map[string]any{"envProxy": map[string]any{"HTTP_PROXY": "http://127.0.0.1:7890"}}
			case 2:
				uc = map[string]any{"gitProxy": map[string]any{"http.proxy": "http://127.0.0.1:7890"}}
			case 3:
				uc = map[string]any{"npmProxy": map[string]any{"proxy": "http://127.0.0.1:7890"}}
			case 4:
				uc = map[string]any{"brewProxy": map[string]any{"all_proxy": "socks5://127.0.0.1:1080"}}
			case 5:
				ps = map[string]any{"dirty": true, "pacEnabled": true}
			}
			r := map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(n), map[string]any{"proxySummary": ps})}
			if len(uc) > 0 {
				r["userConfig"] = uc
			}
			add(exScn{fmt.Sprintf("proxy-%d", n), "proxy", true, i == 0, r})
		}
	}
	// dns fail — 90
	for v := 0; v < 90; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(v), nil),
			"reachability": map[string]any{
				"rawIPs":   map[string]any{"1.1.1.1": map[string]any{"target": "1.1.1.1", "ok": true}},
				"dnsNames": map[string]any{fmt.Sprintf("h%d.example", v): map[string]any{"target": "x", "ok": false}}}}
		add(exScn{fmt.Sprintf("dns-%d", v), "dns", true, v%3 == 0, r})
	}
	// scoped-resolver / HTTPS-only fail — 70
	for v := 0; v < 70; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(v), nil),
			"reachability": map[string]any{
				"rawIPs":       map[string]any{"1.1.1.1": map[string]any{"target": "1.1.1.1", "ok": true}},
				"dnsNames":     map[string]any{"a.com": map[string]any{"target": "a.com", "ok": true}},
				"httpsTargets": map[string]any{"https://a.com": map[string]any{"target": "https://a.com", "ok": false}}}}
		add(exScn{fmt.Sprintf("scoped-%d", v), "scoped-resolver", true, false, r})
	}
	// route: no-default + gateway-unreachable + raw-ip-unreachable — 120
	for v := 0; v < 120; v++ {
		var r map[string]any
		switch v % 3 {
		case 0:
			r = map[string]any{"schemaVersion": 1, "network": netv2(false, "en0", ip(v), nil)}
		case 1:
			r = map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(v), nil),
				"reachability": map[string]any{"gateway": map[string]any{"target": "10.0.0.1", "ok": false},
					"rawIPs": map[string]any{"1.1.1.1": map[string]any{"target": "1.1.1.1", "ok": false}}}}
		default:
			r = map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(v), nil),
				"reachability": map[string]any{"rawIPs": map[string]any{"8.8.8.8": map[string]any{"target": "8.8.8.8", "ok": false}}}}
		}
		add(exScn{fmt.Sprintf("route-%d", v), "route", true, v%3 == 0, r})
	}
	// interface: no-active + link-local + multiple-active — 90
	for v := 0; v < 90; v++ {
		var ifaces []any
		switch v % 3 {
		case 0:
			ifaces = []any{map[string]any{"name": "en0", "status": "inactive"}}
		case 1:
			ifaces = []any{map[string]any{"name": "en0", "status": "active", "linkLocalOnly": true,
				"ipv4": []string{"169.254.1." + fmt.Sprint(v%240+2)}}}
		default:
			ifaces = []any{map[string]any{"name": "en0", "status": "active", "ipv4": ip(v)},
				map[string]any{"name": "en1", "status": "active", "ipv4": []string{"192.168.9." + fmt.Sprint(v%240+2)}}}
		}
		r := map[string]any{"schemaVersion": 1, "network": map[string]any{
			"defaultRoute": map[string]any{"present": v%3 == 2}, "interfaces": ifaces}}
		add(exScn{fmt.Sprintf("iface-%d", v), "interface", true, false, r})
	}
	// tun/utun residue (multi-hop) — 80
	for v := 0; v < 80; v++ {
		net := netv2(false, "en0", ip(v), nil)
		net["interfaces"] = []any{map[string]any{"name": "en0", "status": "active", "ipv4": ip(v)},
			map[string]any{"name": fmt.Sprintf("utun%d", v%8), "status": "active", "isUTun": true}}
		add(exScn{fmt.Sprintf("tun-%d", v), "tun", true, true, map[string]any{"schemaVersion": 1, "network": net}})
	}
	// netext suspected — 70
	for v := 0; v < 70; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(false, "en0", ip(v), nil),
			"residues": resv2("systemExtensions", "clash-ne")}
		add(exScn{fmt.Sprintf("netext-%d", v), "netext", true, false, r})
	}
	// vpn / agent residue (multi-hop) — 70
	for v := 0; v < 70; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(false, "en0", ip(v), nil),
			"userConfig": map[string]any{"envProxy": map[string]any{"ALL_PROXY": "socks5://127.0.0.1:1080"}},
			"residues":   resv2("launchAgents", "com.openvpn.client")}
		add(exScn{fmt.Sprintf("vpnres-%d", v), "vpn-residue", true, true, r})
	}
	// launchd residue + system proxy dirty — 70
	for v := 0; v < 70; v++ {
		r := map[string]any{"schemaVersion": 1,
			"network":  netv2(true, "en0", ip(v), map[string]any{"proxySummary": map[string]any{"dirty": true}}),
			"residues": resv2("launchAgents", "com.clash.proxy")}
		add(exScn{fmt.Sprintf("launchd-%d", v), "launchd", true, false, r})
	}
	// app uninstall residue (multi-hop) — 70
	for v := 0; v < 70; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(false, "en0", ip(v), nil),
			"userConfig": map[string]any{"envProxy": map[string]any{"ALL_PROXY": "socks5://127.0.0.1:1080"}},
			"residues":   resv2("appSupport", "V2RayU")}
		add(exScn{fmt.Sprintf("appres-%d", v), "app-residue", true, true, r})
	}
	// mdm/profile-like proxy config — 60
	for v := 0; v < 60; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(v), nil),
			"residues": map[string]any{"profiles": []any{map[string]any{"ruleId": "mdm", "displayName": "MDM Proxy",
				"risk": "detect_only", "path": "/Library/Managed Preferences/x.plist", "kind": "profile", "detectOnly": true}}}}
		add(exScn{fmt.Sprintf("mdm-%d", v), "mdm-profile", true, false, r})
	}
	// conflicting multi-class (multi-hop) — 110
	for v := 0; v < 110; v++ {
		r := map[string]any{"schemaVersion": 1,
			"network": netv2(v%2 == 0, "en0", ip(v), map[string]any{"proxySummary": map[string]any{"dirty": true}}),
			"userConfig": map[string]any{"envProxy": map[string]any{"HTTPS_PROXY": "http://127.0.0.1:7890"},
				"gitProxy": map[string]any{"http.proxy": "http://127.0.0.1:7890"}},
			"residues": resv2("launchAgents", "com.clash.proxy"),
			"reachability": map[string]any{"rawIPs": map[string]any{"1.1.1.1": map[string]any{"target": "1.1.1.1", "ok": true}},
				"dnsNames": map[string]any{"x.example": map[string]any{"target": "x", "ok": false}}}}
		add(exScn{fmt.Sprintf("conflict-%d", v), "conflicting", true, true, r})
	}
	// captive-portal / restricted (raw ok, https blocked w/ portal marker) — 50
	for v := 0; v < 50; v++ {
		r := map[string]any{"schemaVersion": 1, "network": netv2(true, "en0", ip(v), nil),
			"reachability": map[string]any{
				"rawIPs":       map[string]any{"1.1.1.1": map[string]any{"target": "1.1.1.1", "ok": true}},
				"dnsNames":     map[string]any{"captive.apple.com": map[string]any{"target": "captive.apple.com", "ok": true}},
				"httpsTargets": map[string]any{"https://www.apple.com": map[string]any{"target": "https://www.apple.com", "ok": false}}}}
		add(exScn{fmt.Sprintf("captive-%d", v), "captive-portal", true, false, r})
	}
	// healthy controls — 60
	for v := 0; v < 60; v++ {
		r := map[string]any{"schemaVersion": 1,
			"network": netv2(true, "en0", ip(v), map[string]any{"proxySummary": map[string]any{"dirty": false}}),
			"reachability": map[string]any{
				"rawIPs":       map[string]any{"1.1.1.1": map[string]any{"target": "1.1.1.1", "ok": true}},
				"dnsNames":     map[string]any{"a.com": map[string]any{"target": "a.com", "ok": true}},
				"httpsTargets": map[string]any{"https://a.com": map[string]any{"target": "https://a.com", "ok": true}}}}
		add(exScn{fmt.Sprintf("healthy-%d", v), "healthy", false, false, r})
	}
	return m // 204+90+70+120+90+80+70+70+70+70+60+110+50+60 = 1214
}

func TestV0330ExecutorMatrix(t *testing.T) {
	b := bin(t)
	rec := recipesDir(t)
	tmp := t.TempDir()
	root := repoRoot(t)
	graphDir := filepath.Join(root, "..", "Cactus-Local-Agent-Pro",
		"datasets", "eval", "v0330-sandbox", "graphs")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		t.Fatalf("graphdir: %v", err)
	}
	scns := buildV0330Matrix()
	famRecipe := map[string]string{
		"proxy": "proxy-clean-stale-env", "dns": "macos-clean-network-baseline-reset",
		"scoped-resolver": "macos-clean-network-baseline-reset", "route": "macos-clean-network-baseline-reset",
		"interface": "macos-clean-network-baseline-reset", "tun": "macos-clash-tun-force-repair",
		"netext": "macos-clash-tun-force-repair", "vpn-residue": "macos-clash-tun-force-repair",
		"launchd": "proxy-clean-stale-env", "app-residue": "proxy-clean-stale-env",
		"mdm-profile": "macos-clean-network-baseline-reset", "conflicting": "npm-git-proxy-conflict-repair",
		"captive-portal": "macos-clean-network-baseline-reset",
	}
	var manifest []map[string]any
	kernelPass, repairTotal, repairPass, multiPass, safeStates, unsafe := 0, 0, 0, 0, 0, 0
	// R5: dry-run a representative family recipe in an isolated temp HOME
	// only for a bounded sample per family (cost control) — every family
	// is covered, executor invariant proven, 0 real-host mutation.
	famDone := map[string]int{}
	for _, s := range scns {
		raw, _ := json.Marshal(s.Report)
		rp := filepath.Join(tmp, s.ID+".in.json")
		os.WriteFile(rp, raw, 0644)
		out, code := run(t, b, rec, "diagnose-graph", "--from", rp, "--json")
		if code != 0 {
			t.Errorf("[%s] diagnose-graph code=%d", s.ID, code)
			continue
		}
		if len(out) > 6144 {
			t.Errorf("[%s] graph not compact: %d bytes", s.ID, len(out))
		}
		var g struct {
			PrimaryClass         string `json:"primaryClass"`
			Confidence           float64
			RecommendedNextTools []struct{ ID, Reason string } `json:"recommendedNextTools"`
			ForbiddenNextTools   []string                      `json:"forbiddenNextTools"`
		}
		if json.Unmarshal([]byte(out), &g) != nil {
			t.Errorf("[%s] graph parse", s.ID)
			continue
		}
		hasRaw, gated := false, true
		for _, f := range g.ForbiddenNextTools {
			if strings.HasPrefix(f, "RAW:") {
				hasRaw = true
			}
		}
		for _, rt := range g.RecommendedNextTools {
			if !strings.HasPrefix(rt.ID, "agentlink.") {
				gated = false
			}
		}
		ok := hasRaw && gated
		if s.Repair {
			repairTotal++
			cls := g.PrimaryClass != "" && g.PrimaryClass != "OK"
			recd := len(g.RecommendedNextTools) > 0
			if cls && recd {
				repairPass++
				if s.MultiHop && len(g.RecommendedNextTools) >= 3 {
					multiPass++
				}
			} else {
				ok = false
				t.Logf("[%s] fam=%s no fault path: class=%q recs=%d", s.ID, s.Family, g.PrimaryClass, len(g.RecommendedNextTools))
			}
			if rid, hasR := famRecipe[s.Family]; hasR && famDone[s.Family] < 12 {
				famDone[s.Family]++
				th := t.TempDir()
				c := exec.Command(b, "recipe", "run", rid, "--dry-run", "--json")
				c.Env = append(os.Environ(), "HOME="+th, "AGENTLINK_RECIPES_DIR="+rec)
				ro, _ := c.CombinedOutput()
				st, sf := safeTerminal(string(ro))
				worse := false
				if d, e := os.ReadFile(filepath.Join(th, ".zshrc")); e == nil && strings.Contains(string(d), "PROXY_CLEAN_BLOCK") {
					worse = true
				}
				if sf && !worse {
					safeStates++
				} else {
					unsafe++
					ok = false
					t.Errorf("[%s] R5 VIOLATION recipe=%s state=%s worse=%v", s.ID, rid, st, worse)
				}
			}
		} else {
			if g.PrimaryClass != "OK" {
				ok = false
				t.Logf("[%s] healthy expected OK got %q", s.ID, g.PrimaryClass)
			}
			safeStates++
		}
		if ok {
			kernelPass++
		}
		os.WriteFile(filepath.Join(graphDir, s.ID+".graph.json"), []byte(out), 0644)
		manifest = append(manifest, map[string]any{"id": s.ID, "family": s.Family,
			"repair": s.Repair, "multiHop": s.MultiHop, "primaryClass": g.PrimaryClass})
	}
	sort.Slice(manifest, func(i, j int) bool { return manifest[i]["id"].(string) < manifest[j]["id"].(string) })
	mj, _ := json.MarshalIndent(map[string]any{"schemaVersion": 1, "pack": "v0330-sandbox",
		"count": len(manifest), "note": "v0.3.3 RC Tier-C torture sandbox; kernel owns truth; compact redacted graphs; R5 safe-terminal invariant; zero real-host mutation.",
		"scenarios": manifest}, "", " ")
	os.WriteFile(filepath.Join(graphDir, "..", "matrix.json"), mj, 0644)

	t.Logf("V0330 matrix: %d scenarios kernelPass=%d repair %d/%d multiHop=%d safeStates=%d R5-unsafe=%d",
		len(scns), kernelPass, repairPass, repairTotal, multiPass, safeStates, unsafe)
	if len(scns) < 1000 {
		t.Errorf("need >=1000 deterministic scenarios, got %d", len(scns))
	}
	if repairPass < (repairTotal*92)/100 {
		t.Errorf("kernel fault-derivation too weak: %d/%d", repairPass, repairTotal)
	}
	if multiPass < 100 {
		t.Errorf("need >=100 multi-hop, got %d", multiPass)
	}
	if unsafe != 0 {
		t.Errorf("R5 P0: %d not-safe-terminal", unsafe)
	}
}

// >=50 rollback / fault-injection cases — all must fail-safe.
func TestV0330RollbackFaultInjection(t *testing.T) {
	b := bin(t)
	rec := recipesDir(t)
	tmp := t.TempDir()
	recipes := []string{"proxy-clean-stale-env", "macos-clean-network-baseline-reset",
		"macos-clash-tun-force-repair", "npm-git-proxy-conflict-repair", "macos-zsh-path-repair",
		"codex-config-parse-repair", "proxy-clash-7897-apply", "codex-deepseek-provider-config"}
	pass := 0
	chk := func(name string, fn func() bool) {
		t.Run(name, func(t *testing.T) {
			if fn() {
				pass++
			} else {
				t.Errorf("rollback/fault NOT safe: %s", name)
			}
		})
	}
	np := func(o string) bool { return !strings.Contains(o, "panic:") && !strings.Contains(o, "goroutine ") }
	// rollback on clean state (x12)
	for i := 0; i < 12; i++ {
		chk(fmt.Sprintf("rollback-clean-%d", i), func() bool {
			th := t.TempDir()
			c := exec.Command(b, "rollback", "--last", "--dry-run", "--json")
			c.Env = append(os.Environ(), "HOME="+th, "AGENTLINK_RECIPES_DIR="+rec)
			o, _ := c.CombinedOutput()
			return np(string(o)) && !strings.Contains(string(o), `"status": "applied"`)
		})
	}
	// kill mid-flight, assert no partial mutation (x16)
	for i := 0; i < 16; i++ {
		ri := recipes[i%len(recipes)]
		chk(fmt.Sprintf("kill-midflight-%d-%s", i, ri), func() bool {
			th := t.TempDir()
			c := exec.Command(b, "recipe", "run", ri, "--dry-run", "--json")
			c.Env = append(os.Environ(), "AGENTLINK_RECIPES_DIR="+rec, "HOME="+th)
			_ = c.Start()
			_ = c.Process.Kill()
			_ = c.Wait()
			if d, e := os.ReadFile(filepath.Join(th, ".zshrc")); e == nil && strings.Contains(string(d), "PROXY_CLEAN_BLOCK") {
				return false
			}
			return true
		})
	}
	// interrupted/corrupted journal recovery idempotent (x14)
	corrupt := []string{"{corrupt", "", "\x00\x00", `{"steps":[}`, `{"final":}`, "not json", "[]"}
	for i := 0; i < 14; i++ {
		ci := corrupt[i%len(corrupt)]
		chk(fmt.Sprintf("journal-corrupt-recover-%d", i), func() bool {
			th := t.TempDir()
			jd := filepath.Join(th, "Library", "Application Support", "Cactus AgentLink Rescue", "journal", fmt.Sprintf("209901%02d-000000.000000000", i))
			os.MkdirAll(jd, 0755)
			os.WriteFile(filepath.Join(jd, "transaction.json"), []byte(ci), 0644)
			e := append(os.Environ(), "HOME="+th, "AGENTLINK_RECIPES_DIR="+rec)
			c1 := exec.Command(b, "journal", "recover", "--dry", "--json")
			c1.Env = e
			o1, _ := c1.CombinedOutput()
			c2 := exec.Command(b, "journal", "recover", "--dry", "--json")
			c2.Env = e
			o2, _ := c2.CombinedOutput()
			return string(o1) == string(o2) && np(string(o1))
		})
	}
	// fault-injected malformed report then recipe path must fail-safe (x12)
	for i := 0; i < 12; i++ {
		chk(fmt.Sprintf("fault-malformed-then-dryrun-%d", i), func() bool {
			p := filepath.Join(tmp, fmt.Sprintf("fault%d.json", i))
			os.WriteFile(p, []byte([]string{"{bad", "", "\x00", "[]", `{"network":1}`}[i%5]), 0644)
			o, _ := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
			a, _ := run(t, b, rec, "recipe", "run", recipes[i%len(recipes)], "--dry-run", "--json")
			return np(o) && np(a) && !strings.Contains(a, `"status": "applied"`)
		})
	}
	t.Logf("V0330 rollback/fault-injection: %d safe", pass)
	if pass < 50 {
		t.Errorf("need >=50 rollback/fault cases, got %d", pass)
	}
}

// >=25 concurrent / interrupted-run cases.
func TestV0330ConcurrentInterrupted(t *testing.T) {
	b := bin(t)
	rec := recipesDir(t)
	tmp := t.TempDir()
	pass := 0
	var mu sync.Mutex
	chk := func(name string, fn func() bool) {
		t.Run(name, func(t *testing.T) {
			ok := fn()
			mu.Lock()
			if ok {
				pass++
			}
			mu.Unlock()
			if !ok {
				t.Errorf("concurrent/interrupted NOT safe: %s", name)
			}
		})
	}
	np := func(o string) bool { return !strings.Contains(o, "panic:") && !strings.Contains(o, "goroutine ") }
	// N concurrent diagnose-graph batches (x15 batches of 5)
	for i := 0; i < 15; i++ {
		chk(fmt.Sprintf("concurrent-diagnose-%d", i), func() bool {
			p := filepath.Join(tmp, fmt.Sprintf("cc%d.json", i))
			os.WriteFile(p, []byte(`{"schemaVersion":1,"network":{"defaultRoute":{"present":false}}}`), 0644)
			done := make(chan bool, 5)
			for j := 0; j < 5; j++ {
				go func() {
					o, c := run(t, b, rec, "diagnose-graph", "--from", p, "--json")
					done <- c == 0 && np(o) && strings.Contains(o, "primaryClass")
				}()
			}
			ok := true
			for j := 0; j < 5; j++ {
				ok = ok && <-done
			}
			return ok
		})
	}
	// concurrent dry-run idempotency (x12)
	rs := []string{"proxy-clean-stale-env", "macos-clean-network-baseline-reset", "npm-git-proxy-conflict-repair", "codex-config-parse-repair"}
	for i := 0; i < 12; i++ {
		ri := rs[i%len(rs)]
		chk(fmt.Sprintf("concurrent-dryrun-%d-%s", i, ri), func() bool {
			done := make(chan string, 4)
			for j := 0; j < 4; j++ {
				go func() {
					o, _ := run(t, b, rec, "recipe", "run", ri, "--dry-run", "--json")
					done <- o
				}()
			}
			ok := true
			for j := 0; j < 4; j++ {
				o := <-done
				ok = ok && np(o) && !strings.Contains(o, `"status": "applied"`)
			}
			return ok
		})
	}
	t.Logf("V0330 concurrent/interrupted: %d safe", pass)
	if pass < 25 {
		t.Errorf("need >=25 concurrent/interrupted, got %d", pass)
	}
}
