package interference

import "testing"

func TestCatalogNonEmpty(t *testing.T) {
	if len(Catalog()) == 0 {
		t.Fatal("provider catalog is empty")
	}
}

// TestFactsDetectsClashFromResidueAndTun proves the interference detector fires
// on both residue (launch/helper paths) and a stale TUN signature, and marks
// an automated-repair provider safe to quarantine.
func TestFactsDetectsClashFromResidueAndTun(t *testing.T) {
	facts := Facts(
		[]string{"/Library/LaunchAgents/io.github.clash-verge-rev.plist"},
		[]string{"utun4 inet 198.18.0.1 --> 198.18.0.1"},
	)
	var got *ProviderFact
	for i := range facts {
		if facts[i].ID == ProviderClashVergeRev {
			got = &facts[i]
		}
	}
	if got == nil {
		t.Fatalf("clash-verge-rev not detected: %+v", facts)
	}
	if !got.Detected || !got.ResidueDetected || !got.StaleTun {
		t.Fatalf("clash-verge-rev detection incomplete: %+v", *got)
	}
	if !got.SafeToQuarantine {
		t.Fatalf("automated-repair provider should be safe to quarantine: %+v", *got)
	}
}

// TestFactsDetectOnlyProviderNotFalsePositive ensures a clean environment
// yields no detections.
func TestFactsNoFalsePositiveOnCleanInput(t *testing.T) {
	if facts := Facts(nil, nil); len(facts) != 0 {
		t.Fatalf("clean input produced detections: %+v", facts)
	}
}
