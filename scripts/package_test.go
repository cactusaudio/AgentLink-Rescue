package scripts_test

import (
	"io/fs"
	"os"
	"path/filepath"
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
	for _, want := range []string{"COPYFILE_DISABLE=1", "zip -r -X", "unzip -l", "AppleDouble", "/\\._", "Cactus-AgentLink-Rescue-v0.4.4-core.zip", "assets/manifests", "assets/installers", "fetch_clash_verge_rev.sh", "core package must not contain GGUF", "core package must not contain llama.cpp"} {
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
	for _, want := range []string{"dogfood_temp_home.sh", "dogfood_proxy_config.sh", "dogfood_runtime_core.sh", "dogfood_guided_rescue.sh", "dogfood_readiness.sh", "dogfood_installer_center.sh", "dogfood_brain_chat.sh", "dogfood_runtime_assets.sh", "dogfood_runtime_brain.sh", "dogfood_gui_core.sh", "dogfood_gui_brain.sh", "dogfood_gui_screenshots.sh", "build_gui.sh", "readiness doctor --json", "support bundle", "manifest.lock.json changed during package/dogfood", "brain assets missing; run scripts/fetch_brain_assets.sh or scripts/package_brain.sh DOWNLOAD=1", "FORBIDDEN_FIELD", "active legacy model reference found", "Cactus-AgentLink-Rescue-v0.4.4-core.zip", "Cactus-AgentLink-Rescue-v0.4.4-core-gui.zip", "Mach-O universal binary"} {
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
	for _, want := range []string{"Cactus-AgentLink-Rescue-v0.4.4-brain-gemma4-e4b-q4km.zip", "DOWNLOAD=1", "assets/models/gemma-4-E4B-it-Q4_K_M.gguf", "assets/runtimes/llama.cpp", "assets/installers", "manifest.lock.json", "COPYFILE_DISABLE=1", "zip -r -X"} {
		if !strings.Contains(text, want) {
			t.Fatalf("brain package script missing %s", want)
		}
	}
}

func TestRuntimeDogfoodScriptsExist(t *testing.T) {
	for _, name := range []string{"dogfood_runtime_core.sh", "dogfood_runtime_brain.sh", "dogfood_runtime_assets.sh", "dogfood_guided_rescue.sh", "dogfood_readiness.sh", "dogfood_installer_center.sh", "dogfood_brain_chat.sh", "dogfood_gui_core.sh", "dogfood_gui_brain.sh"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		required := []string{"0.4.4"}
		if name != "dogfood_guided_rescue.sh" && name != "dogfood_installer_center.sh" && name != "dogfood_readiness.sh" {
			required = append(required, "brain")
		}
		for _, want := range required {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %s", name, want)
			}
		}
	}
}

func TestGUIPackageScriptsEmbedAgentlinkResources(t *testing.T) {
	for _, name := range []string{"package_gui_core.sh", "package_gui_brain.sh"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, want := range []string{"Contents/Resources/agentlink", "Cactus AgentLink Rescue.app", "COPYFILE_DISABLE=1", "zip -r -X", "Info.plist"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %s", name, want)
			}
		}
	}
}

func TestNoActiveLegacyModelReferences(t *testing.T) {
	legacyTerms := []string{"Q" + "wen", "q" + "wen", "Q" + "WEN"}
	roots := []string{"../README.md", "../packaging", "../docs/offline", "../internal", "../scripts", "../assets/manifests", "../assets/README.md", "../gui/CactusAgentLinkRescue/Sources/CactusAgentLinkRescue"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.Contains(path, "assets/models") || strings.Contains(path, "assets/runtimes") {
					return filepath.SkipDir
				}
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			for _, term := range legacyTerms {
				if strings.Contains(text, term) {
					t.Fatalf("active legacy model reference %q found in %s", term, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
