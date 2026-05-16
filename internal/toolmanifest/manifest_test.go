package toolmanifest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCatalogSchemaComplete(t *testing.T) {
	m := Build("0.5.1-test")
	if probs := Validate(m); len(probs) != 0 {
		t.Fatalf("manifest not schema-complete:\n  %s", strings.Join(probs, "\n  "))
	}
}

func TestCatalogCoversRequiredFamilies(t *testing.T) {
	m := Build("x")
	fams := map[string]int{}
	for _, c := range m.Tools {
		fams[c.Family]++
	}
	for _, want := range []string{"diagnosis", "planning", "execution", "reporting"} {
		if fams[want] == 0 {
			t.Errorf("required family %q has no tools", want)
		}
	}
	if len(m.Tools) < 30 {
		t.Errorf("catalog too small: %d tools (<30)", len(m.Tools))
	}
}

func TestRequiredToolIDsPresent(t *testing.T) {
	m := Build("x")
	have := map[string]bool{}
	for _, c := range m.Tools {
		have[c.ID] = true
	}
	required := []string{
		"agentlink.version", "agentlink.doctor", "agentlink.readiness_doctor",
		"agentlink.support_bundle", "agentlink.verify_network",
		"agentlink.dns_snapshot", "agentlink.tun_snapshot",
		"agentlink.network_extension_snapshot", "agentlink.launchd_snapshot",
		"agentlink.classify_incident", "agentlink.diagnosis_graph",
		"agentlink.recommend_recipes", "agentlink.recipe_dry_run",
		"agentlink.approval_ticket_create", "agentlink.execute_recipe",
		"agentlink.network_baseline_reset", "agentlink.remove_tun_residue",
		"agentlink.restore_last_good", "agentlink.rollback_last",
		"agentlink.incident_report", "agentlink.support_bundle_redacted",
		"agentlink.after_action_report",
	}
	for _, id := range required {
		if !have[id] {
			t.Errorf("required tool id missing from catalog: %s", id)
		}
	}
}

func TestHostMutatingToolsAreGated(t *testing.T) {
	m := Build("x")
	for _, c := range m.Tools {
		if c.MutationClass != MutationHostTxn {
			continue
		}
		if !c.HumanApproval {
			t.Errorf("%s mutates host but has no human-approval gate", c.ID)
		}
		if !c.DryRunSupported {
			t.Errorf("%s mutates host but has no dry-run", c.ID)
		}
		if c.RecommendAllowed {
			t.Errorf("%s is host-mutating execution but marked Gemma-allowed (must be recommend-only)", c.ID)
		}
		if !c.RollbackSupported && !c.RollbackImpossible {
			t.Errorf("%s does not declare rollback support OR impossibility", c.ID)
		}
	}
}

func TestNoForbiddenRawSurfaceInCatalog(t *testing.T) {
	m := Build("x")
	for _, c := range m.Tools {
		// precise argv safety: no raw-binary argv[0], no --yes/sudo/rm -rf.
		// AgentLink diagnosis subcommand nouns (e.g. "route") are legit.
		if reason := argvUnsafe(c.Argv); reason != "" {
			t.Errorf("%s argv unsafe: %s (argv=%v)", c.ID, reason, c.Argv)
		}
		if strings.Contains(strings.ToLower(c.ExampleGoodCall), "sudo ") {
			t.Errorf("%s exampleGoodCall must not contain sudo", c.ID)
		}
		// Model-facing execution is BOUNDED: a host_txn tool's argv must
		// be EITHER `recipe run <id> --dry-run --json` (model can ONLY
		// dry-run; real apply is a user-approved ticket) OR a recognized
		// AgentLink transactional subcommand (rollback / journal recover
		// / last-good restore / package repair — journal+snapshot-backed,
		// no --yes). NEVER a raw --yes/-y auto-execute.
		if c.MutationClass == MutationHostTxn {
			txnCLI := map[string]bool{"rollback": true, "journal": true,
				"last-good": true, "package": true}
			bounded := false
			if len(c.Argv) >= 2 && c.Argv[0] == "recipe" && c.Argv[1] == "run" {
				for _, a := range c.Argv {
					if a == "--dry-run" {
						bounded = true
					}
				}
			} else if len(c.Argv) >= 1 && txnCLI[c.Argv[0]] {
				bounded = true // AgentLink's own journaled transactional command
			}
			for _, a := range c.Argv {
				if a == "--yes" || a == "-y" {
					t.Errorf("%s execution tool exposes raw auto-execute %q", c.ID, a)
				}
			}
			if !bounded {
				t.Errorf("%s host_txn tool must be model-dry-run (recipe run <id> --dry-run) OR an AgentLink transactional subcommand; argv=%v", c.ID, c.Argv)
			}
		}
	}
}

func TestManifestJSONStableRoundTrip(t *testing.T) {
	m := Build("rt")
	b, err := m.JSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Manifest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Tools) != len(m.Tools) || back.SchemaVersion != m.SchemaVersion {
		t.Fatalf("round-trip mismatch: %d vs %d tools", len(back.Tools), len(m.Tools))
	}
	if probs := Validate(back); len(probs) != 0 {
		t.Fatalf("round-tripped manifest invalid: %v", probs)
	}
}
