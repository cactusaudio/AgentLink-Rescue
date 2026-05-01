package scripts_test

import (
	"os"
	"strings"
	"testing"
)

func TestPackageScriptRemovesMetadataFiles(t *testing.T) {
	data, err := os.ReadFile("package.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{".DS_Store", "__MACOSX", "._*", ".AppleDouble", "AppleDouble"} {
		if !strings.Contains(text, want) {
			t.Fatalf("package cleanup missing %s", want)
		}
	}
	for _, want := range []string{"COPYFILE_DISABLE=1", "zip -r -X", "unzip -l", "AppleDouble", "/\\._", "Cactus-AgentLink-Rescue-v0.2.1.zip"} {
		if !strings.Contains(text, want) {
			t.Fatalf("package zip hardening missing %s", want)
		}
	}
}

func TestFallbackRescueRecordsNetworkLocationForRollback(t *testing.T) {
	data, err := os.ReadFile("../packaging/rescue.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"previous_location.txt", "-getcurrentlocation", "-switchtolocation \"$previous\"", "network location restore failed"} {
		if !strings.Contains(text, want) {
			t.Fatalf("fallback rollback missing %s", want)
		}
	}
}
