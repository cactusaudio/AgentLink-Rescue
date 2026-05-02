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
	for _, want := range []string{"COPYFILE_DISABLE=1", "zip -r -X", "unzip -l", "AppleDouble", "/\\._", "Cactus-AgentLink-Rescue-v0.3.1-core.zip", "assets/manifests", "core package must not contain GGUF", "core package must not contain llama.cpp"} {
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

func TestReleaseCheckCoversDogfoodAndForbiddenCodexField(t *testing.T) {
	data, err := os.ReadFile("release_check.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"dogfood_temp_home.sh", "dogfood_proxy_config.sh", "dogfood_runtime_core.sh", "dogfood_runtime_assets.sh", "dogfood_runtime_brain.sh", "brain assets missing; run scripts/fetch_brain_assets.sh or scripts/package_brain.sh DOWNLOAD=1", "FORBIDDEN_FIELD", "Cactus-AgentLink-Rescue-v0.3.1-core.zip", "Mach-O universal binary"} {
		if !strings.Contains(text, want) {
			t.Fatalf("release check missing %s", want)
		}
	}
}

func TestBrainPackageScriptRequiresAssetsAndHygiene(t *testing.T) {
	data, err := os.ReadFile("package_brain.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"Cactus-AgentLink-Rescue-v0.3.1-brain-qwen3-4b-q4km.zip", "DOWNLOAD=1", "assets/models", "assets/runtimes/llama.cpp", "MODEL_SHA_EXPECTED", "COPYFILE_DISABLE=1", "zip -r -X"} {
		if !strings.Contains(text, want) {
			t.Fatalf("brain package script missing %s", want)
		}
	}
}

func TestRuntimeDogfoodScriptsExist(t *testing.T) {
	for _, name := range []string{"dogfood_runtime_core.sh", "dogfood_runtime_brain.sh", "dogfood_runtime_assets.sh"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{"0.3.1", "brain"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %s", name, want)
			}
		}
	}
}
