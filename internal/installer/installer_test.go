package installer

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestCatalogParsesAndUsesOfficialPackages(t *testing.T) {
	cat, err := LoadCatalog(filepath.Join("..", "..", "assets", "installers", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"codex-cli", "codex-app", "claude-code-cli", "gemini-cli", "clash-verge-rev"} {
		if _, ok := cat.Find(id); !ok {
			t.Fatalf("missing installer %s", id)
		}
	}
	codex, _ := cat.Find("codex-cli")
	assertMethodContains(t, codex, "npm", "@openai/codex")
	assertMethodContains(t, codex, "homebrew", "codex")
	claude, _ := cat.Find("claude-code-cli")
	assertMethodContains(t, claude, "npm", "@anthropic-ai/claude-code")
	gemini, _ := cat.Find("gemini-cli")
	assertMethodContains(t, gemini, "npm", "@google/gemini-cli")
	assertMethodContains(t, gemini, "homebrew", "gemini-cli")
}

func TestCatalogRejectsFakeGeminiPackage(t *testing.T) {
	bad := Installer{
		ID:            "gemini-cli",
		DefaultMethod: "npm",
		Methods: []Method{{
			ID:      "npm",
			Type:    "npm",
			Command: []string{"npm", "install", "-g", "gemini"},
		}},
	}
	if err := validateInstaller(bad); err == nil {
		t.Fatal("expected fake Gemini package to be rejected")
	}
}

func TestDryRunDoesNotExecuteInstallCommand(t *testing.T) {
	cat, err := LoadCatalog(filepath.Join("..", "..", "assets", "installers", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &command.MockRunner{}
	rep, err := Run(context.Background(), runner, cat, Options{Action: ActionDryRun, ID: "codex-cli", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != "dry_run" {
		t.Fatalf("status = %s", rep.Status)
	}
	for _, call := range runner.Calls {
		if strings.Contains(call, "install -g @openai/codex") {
			t.Fatalf("dry-run executed install command: %s", call)
		}
	}
}

func TestInstallRequiresYes(t *testing.T) {
	cat, err := LoadCatalog(filepath.Join("..", "..", "assets", "installers", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Run(context.Background(), &command.MockRunner{}, cat, Options{Action: ActionInstall, ID: "codex-cli", Version: "test"})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected --yes error, got %v", err)
	}
}

func TestCodexAppOfficialURLOnly(t *testing.T) {
	cat, err := LoadCatalog(filepath.Join("..", "..", "assets", "installers", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	codexApp, _ := cat.Find("codex-app")
	method, ok := codexApp.Method("")
	if !ok {
		t.Fatal("missing codex app method")
	}
	if !strings.HasPrefix(method.OfficialURL, "https://developers.openai.com/") {
		t.Fatalf("non-official Codex app URL: %s", method.OfficialURL)
	}
}

func TestClashArchSelection(t *testing.T) {
	arch := RuntimeArch()
	if arch != "arm64" && arch != "amd64" {
		t.Fatalf("unexpected arch: %s", arch)
	}
}

func assertMethodContains(t *testing.T, in Installer, methodID, token string) {
	t.Helper()
	method, ok := in.Method(methodID)
	if !ok {
		t.Fatalf("%s method missing for %s", methodID, in.ID)
	}
	if !containsExact(method.Command, token) {
		t.Fatalf("%s %s command missing %s: %#v", in.ID, methodID, token, method.Command)
	}
}
