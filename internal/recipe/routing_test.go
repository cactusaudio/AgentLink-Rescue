package recipe

import "testing"

func routingRegistry() Registry {
	return RegistryForTest(map[string]Recipe{
		"reversible-proxy": {
			SchemaVersion:  1,
			ID:             "reversible-proxy",
			Title:          "Reversible proxy clean",
			FailureClasses: []string{"USER_PROXY_DIRTY"},
			Risk:           RiskReversiblePatch,
			Patches:        []Patch{{Type: "append_managed_block_if_missing", Path: "~/.zshrc", Marker: "X", Content: "x"}},
			Rollback:       []RollbackSpec{{Type: "restore_file", Path: "~/.zshrc"}},
		},
		"privileged-net": {
			SchemaVersion:  1,
			ID:             "privileged-net",
			Title:          "Privileged net reset",
			FailureClasses: []string{"USER_PROXY_DIRTY", "NO_DEFAULT_ROUTE"},
			Risk:           RiskPrivilegedAction,
			RequiresRoot:   true,
		},
		"readonly-keys": {
			SchemaVersion:  1,
			ID:             "readonly-keys",
			Title:          "Read-only keys",
			FailureClasses: []string{"KEYS_STATUS_UNKNOWN"},
			Risk:           RiskReadOnly,
		},
		"destructive-x": {
			SchemaVersion:  1,
			ID:             "destructive-x",
			Title:          "Destructive",
			FailureClasses: []string{"USER_PROXY_DIRTY"},
			Risk:           RiskDestructiveAction,
		},
	})
}

func TestRecipesForClassOrdersLeastInvasiveFirst(t *testing.T) {
	reg := routingRegistry()
	got := reg.RecipesForClass("USER_PROXY_DIRTY")
	// destructive excluded -> 2 candidates, reversible before privileged.
	if len(got) != 2 {
		t.Fatalf("expected 2 non-destructive candidates, got %d: %+v", len(got), ids(got))
	}
	if got[0].ID != "reversible-proxy" || got[1].ID != "privileged-net" {
		t.Fatalf("least-invasive ordering violated: %v", ids(got))
	}
}

func TestRecipesForClassExcludesDestructive(t *testing.T) {
	reg := routingRegistry()
	for _, rec := range reg.RecipesForClass("USER_PROXY_DIRTY") {
		if rec.Risk == RiskDestructiveAction {
			t.Fatalf("destructive recipe %q was routed", rec.ID)
		}
	}
}

func TestRecipesForClassEmptyAndUnknown(t *testing.T) {
	reg := routingRegistry()
	if got := reg.RecipesForClass(""); got != nil {
		t.Fatalf("empty class should return nil, got %v", ids(got))
	}
	if got := reg.RecipesForClass("NEVER_DECLARED"); len(got) != 0 {
		t.Fatalf("unknown class should return none, got %v", ids(got))
	}
}

func TestLowestRiskRecipeForClass(t *testing.T) {
	reg := routingRegistry()
	rec, ok := reg.LowestRiskRecipeForClass("USER_PROXY_DIRTY")
	if !ok || rec.ID != "reversible-proxy" {
		t.Fatalf("expected reversible-proxy as lowest risk, got ok=%v id=%q", ok, rec.ID)
	}
	if _, ok := reg.LowestRiskRecipeForClass("NEVER_DECLARED"); ok {
		t.Fatal("unknown class must not resolve a recipe")
	}
}

func ids(recs []Recipe) []string {
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.ID)
	}
	return out
}
