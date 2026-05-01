package system

import (
	"context"
	"os/user"
	"path/filepath"
	"testing"

	"cactus-agentlink-rescue/internal/command"
)

func TestAllowedMutationPathAllowlist(t *testing.T) {
	home := "/Users/tester"
	opts := MutationOptions{RealUserHome: home}
	allowed := []string{
		"/Library/LaunchDaemons/io.github.clashverge.helper.plist",
		"/Library/LaunchAgents/com.example.agent.plist",
		"/Library/PrivilegedHelperTools/io.github.clashverge.helper",
		"/Library/Preferences/SystemConfiguration/preferences.plist",
		home + "/Library/LaunchAgents/com.example.agent.plist",
		home + "/Library/Group Containers/group.example",
		home + "/Library/Application Support/Clash Verge",
		home + "/Library/Caches/io.github.clashverge",
		home + "/Library/Preferences/io.github.clashverge.plist",
	}
	for _, path := range allowed {
		if err := IsAllowedMutationPath(path, opts); err != nil {
			t.Fatalf("expected allowed %s: %v", path, err)
		}
	}
	refused := []string{
		"/",
		"/etc/hosts",
		"/Library/Application Support/RootApp",
		"/System/Library/file",
		"/Library/LaunchDaemons/../Preferences/SystemConfiguration/preferences.plist",
		"/Users",
		"/usr/local/bin/tool",
	}
	for _, path := range refused {
		if err := IsAllowedMutationPath(path, opts); err == nil {
			t.Fatalf("expected refused %s", path)
		}
	}
}

func TestNetworkExtensionPlistRequiresExplicitOption(t *testing.T) {
	path := "/Library/Preferences/com.apple.networkextension.test.plist"
	if err := IsAllowedMutationPath(path, MutationOptions{}); err == nil {
		t.Fatal("networkextension plist allowed without option")
	}
	if err := IsAllowedMutationPath(path, MutationOptions{IncludeNetworkExtensionPlists: true}); err != nil {
		t.Fatalf("networkextension plist refused with option: %v", err)
	}
}

func TestRealConsoleUserSUDOUserIncludesLookupIDs(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Skip(err)
	}
	name := filepath.Base(current.Username)
	t.Setenv("SUDO_USER", name)
	key := command.Render("/usr/bin/dscl", ".", "-read", "/Users/"+name, "NFSHomeDirectory")
	runner := &command.MockRunner{Results: map[string]command.Result{
		key: {ExitCode: 0, Stdout: "NFSHomeDirectory: " + current.HomeDir + "\n"},
	}}
	got := RealConsoleUser(context.Background(), runner)
	if got.Home != current.HomeDir || got.UID == "" || got.GID == "" {
		t.Fatalf("unexpected user info: %+v current=%+v", got, current)
	}
}
