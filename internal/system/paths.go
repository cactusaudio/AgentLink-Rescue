package system

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"cactus-agentlink-rescue/internal/command"
)

const AppName = "Cactus AgentLink Rescue"

type UserInfo struct {
	Name string `json:"name"`
	Home string `json:"home"`
	UID  string `json:"uid,omitempty"`
	GID  string `json:"gid,omitempty"`
}

func UserReportDir(home string) string {
	return filepath.Join(home, "Library", "Application Support", AppName, "reports")
}

func UserSessionDir(home string) string {
	return filepath.Join(home, "Library", "Application Support", AppName, "sessions")
}

func UserRestorePointsDir(home string) string {
	return filepath.Join(home, "Library", "Application Support", AppName, "restore-points")
}

func SystemBaseDir() string {
	return filepath.Join(string(os.PathSeparator), "Library", "Application Support", AppName)
}

func RestorePointsDir() string {
	return filepath.Join(SystemBaseDir(), "restore-points")
}

func QuarantineDir() string {
	return filepath.Join(SystemBaseDir(), "quarantine")
}

func IsDarwin() bool {
	return runtime.GOOS == "darwin"
}

func IsRoot() bool {
	return os.Geteuid() == 0
}

func RealConsoleUser(ctx context.Context, runner command.Runner) UserInfo {
	if sudo := os.Getenv("SUDO_USER"); sudo != "" && sudo != "root" {
		if home := homeFromDSCL(ctx, runner, sudo); home != "" {
			return lookupUserWithFallback(sudo, home)
		}
		if u, err := user.Lookup(sudo); err == nil {
			return UserInfo{Name: u.Username, Home: u.HomeDir, UID: u.Uid, GID: u.Gid}
		}
	}
	if runner != nil {
		res := runner.Run(ctx, "/usr/bin/stat", "-f", "%Su", "/dev/console")
		name := strings.TrimSpace(res.Stdout)
		if res.ExitCode == 0 && name != "" && name != "root" {
			if home := homeFromDSCL(ctx, runner, name); home != "" {
				return lookupUserWithFallback(name, home)
			}
			if u, err := user.Lookup(name); err == nil {
				return UserInfo{Name: u.Username, Home: u.HomeDir, UID: u.Uid, GID: u.Gid}
			}
		}
	}
	if u, err := user.Current(); err == nil {
		return UserInfo{Name: u.Username, Home: u.HomeDir, UID: u.Uid, GID: u.Gid}
	}
	home, _ := os.UserHomeDir()
	return UserInfo{Name: os.Getenv("USER"), Home: home}
}

func lookupUserWithFallback(name, home string) UserInfo {
	info := UserInfo{Name: name, Home: home}
	if u, err := user.Lookup(name); err == nil {
		info.Name = u.Username
		info.UID = u.Uid
		info.GID = u.Gid
		if info.Home == "" {
			info.Home = u.HomeDir
		}
	}
	return info
}

func homeFromDSCL(ctx context.Context, runner command.Runner, name string) string {
	if runner == nil || name == "" {
		return ""
	}
	res := runner.Run(ctx, "/usr/bin/dscl", ".", "-read", "/Users/"+name, "NFSHomeDirectory")
	if res.ExitCode != 0 {
		return ""
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NFSHomeDirectory:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "NFSHomeDirectory:"))
		}
	}
	return ""
}

func ExpandUserPath(path string, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func CommandExists(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Mode()&0111 != 0
}

func FindFirstExisting(paths ...string) (string, bool) {
	for _, path := range paths {
		if CommandExists(path) {
			return path, true
		}
	}
	return "", false
}

type MutationOptions struct {
	RealUserHome                  string
	IncludeNetworkExtensionPlists bool
	ExtraAllowedPaths             []string
}

func IsAllowedMutationPath(path string, opts MutationOptions) error {
	if path == "" {
		return errors.New("empty mutation path refused")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("relative mutation path refused: %s", path)
	}
	if hasTraversal(path) {
		return fmt.Errorf("path traversal refused: %s", path)
	}
	clean := filepath.Clean(path)
	if clean != path {
		return fmt.Errorf("ambiguous mutation path refused: %s", path)
	}
	if clean == string(os.PathSeparator) {
		return errors.New("root mutation path refused")
	}
	for _, protected := range []string{"/System", "/bin", "/sbin", "/usr"} {
		if clean == protected || strings.HasPrefix(clean, protected+"/") {
			return fmt.Errorf("protected path refused: %s", clean)
		}
	}

	if childOf(clean, "/Library/LaunchDaemons") ||
		childOf(clean, "/Library/LaunchAgents") ||
		childOf(clean, "/Library/PrivilegedHelperTools") ||
		childOf(clean, "/Library/Preferences/SystemConfiguration") {
		return nil
	}
	if opts.IncludeNetworkExtensionPlists && childOf(clean, "/Library/Preferences") {
		base := filepath.Base(clean)
		if ok, _ := filepath.Match("com.apple.networkextension*.plist", base); ok {
			return nil
		}
	}
	if opts.RealUserHome != "" {
		home := filepath.Clean(opts.RealUserHome)
		for _, suffix := range []string{
			filepath.Join("Library", "LaunchAgents"),
			filepath.Join("Library", "Group Containers"),
			filepath.Join("Library", "Application Support"),
			filepath.Join("Library", "Caches"),
			filepath.Join("Library", "Preferences"),
		} {
			if childOf(clean, filepath.Join(home, suffix)) {
				return nil
			}
		}
	}
	for _, allowed := range opts.ExtraAllowedPaths {
		cleanAllowed := filepath.Clean(allowed)
		if clean == cleanAllowed {
			return nil
		}
	}
	if clean == "/Users" || strings.HasPrefix(clean, "/Users/") {
		return fmt.Errorf("user path outside real-user allowlist refused: %s", clean)
	}
	return fmt.Errorf("mutation path outside allowlist: %s", clean)
}

func EnsureInsideAllowedSystemPath(path string) error {
	return IsAllowedMutationPath(path, MutationOptions{})
}

func childOf(path, parent string) bool {
	parent = filepath.Clean(parent)
	return path != parent && strings.HasPrefix(path, parent+string(os.PathSeparator))
}

func hasTraversal(path string) bool {
	for _, part := range strings.Split(path, string(os.PathSeparator)) {
		if part == ".." {
			return true
		}
	}
	return false
}
