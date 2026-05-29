package genomekernel

import (
	"testing"

	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/genome"
)

// TestEveryClassCoveredByGenome is the P0.1 drift guard: every deterministic
// kernel class (except the healthy OK state) must anchor to a genome layer that
// actually holds cards. Adding a classify class without genome coverage — or
// pointing it at an empty layer — fails here, so the kernel taxonomy can never
// silently diverge from the distilled genome.
func TestEveryClassCoveredByGenome(t *testing.T) {
	corpus, err := genome.Load("")
	if err != nil {
		t.Fatalf("load genome corpus: %v", err)
	}
	if len(corpus.Cards) == 0 {
		t.Fatal("genome corpus loaded zero cards")
	}
	for _, c := range classify.AllClasses() {
		if c == classify.OK {
			if _, ok := ClassLayer[c]; ok {
				t.Fatalf("OK must not map to a failure layer")
			}
			continue
		}
		layer, ok := LayerForClass(c)
		if !ok {
			t.Fatalf("classify class %s has no genome layer mapping (taxonomy drift)", c)
		}
		if !corpus.Layers[layer] {
			t.Fatalf("class %s maps to layer %s which is not present in the corpus", c, layer)
		}
		cards := CoverageForClass(corpus, c)
		if len(cards) == 0 {
			t.Fatalf("class %s maps to layer %s but the genome holds no cards there", c, layer)
		}
	}
}

func TestAdvisoryForClassSurfacesGenomeKnowledge(t *testing.T) {
	corpus, err := genome.Load("")
	if err != nil {
		t.Fatalf("load genome corpus: %v", err)
	}
	adv, ok := AdvisoryForClass(corpus, classify.SystemProxyDirty, 3)
	if !ok {
		t.Fatal("expected genome advisory for SYSTEM_PROXY_DIRTY")
	}
	if adv.Layer != "L05_proxy" {
		t.Fatalf("SYSTEM_PROXY_DIRTY advisory layer = %q, want L05_proxy", adv.Layer)
	}
	if len(adv.CardIDs) == 0 {
		t.Fatalf("advisory carried no card ids: %+v", adv)
	}
	if len(adv.CardIDs) > 3 {
		t.Fatalf("advisory exceeded maxCards: %d", len(adv.CardIDs))
	}
}

func TestUnmappedClassHasNoCoverage(t *testing.T) {
	corpus, err := genome.Load("")
	if err != nil {
		t.Fatalf("load genome corpus: %v", err)
	}
	if cards := CoverageForClass(corpus, classify.OK); cards != nil {
		t.Fatalf("OK must have no genome coverage, got %d cards", len(cards))
	}
	if _, ok := AdvisoryForClass(corpus, "NOT_A_REAL_CLASS", 3); ok {
		t.Fatal("unknown class must not produce an advisory")
	}
}
