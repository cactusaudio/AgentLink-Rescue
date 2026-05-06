package chaos

import (
	"path/filepath"
	"testing"
)

func TestChaosFixturesPass(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "chaos")
	fixtures, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatalf("no chaos fixtures found")
	}
	for _, fixture := range fixtures {
		rep := Run(fixture)
		if rep.Status != "passed" {
			t.Fatalf("%s failed: %+v", fixture, rep)
		}
	}
}
