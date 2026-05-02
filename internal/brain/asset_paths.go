package brain

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"cactus-agentlink-rescue/internal/system"
)

type AssetLocations struct {
	PackageRoot        string `json:"packageRoot,omitempty"`
	BrainHome          string `json:"brainHome,omitempty"`
	ModelPath          string `json:"modelPath,omitempty"`
	ModelExists        bool   `json:"modelExists"`
	ModelSHA256OK      bool   `json:"modelSha256OK"`
	RuntimePath        string `json:"runtimePath,omitempty"`
	RuntimeExists      bool   `json:"runtimeExists"`
	RuntimeExecutable  bool   `json:"runtimeExecutable"`
	PackageLocalAssets bool   `json:"packageLocalAssets"`
	UserCacheAssets    bool   `json:"userCacheAssets"`
}

func LocateAssets(home string, manifest ModelManifest) AssetLocations {
	if manifest.Filename == "" {
		manifest = DefaultModelManifest()
	}
	loc := AssetLocations{PackageRoot: packageRoot(), BrainHome: brainHome(home)}
	var packageLocal, userCache bool
	loc.ModelPath, packageLocal, userCache = locateModel(loc, manifest.Filename)
	loc.PackageLocalAssets = loc.PackageLocalAssets || packageLocal
	loc.UserCacheAssets = loc.UserCacheAssets || userCache
	loc.ModelExists = loc.ModelPath != "" && system.Exists(loc.ModelPath)
	if loc.ModelExists && manifest.SHA256 != "" {
		loc.ModelSHA256OK = sha256File(loc.ModelPath) == manifest.SHA256
	}
	loc.RuntimePath, packageLocal, userCache = locateRuntime(loc)
	loc.PackageLocalAssets = loc.PackageLocalAssets || packageLocal
	loc.UserCacheAssets = loc.UserCacheAssets || userCache
	loc.RuntimeExists = loc.RuntimePath != "" && system.Exists(loc.RuntimePath)
	loc.RuntimeExecutable = system.CommandExists(loc.RuntimePath)
	return loc
}

func packageRoot() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if filepath.Base(dir) == "bin" {
			return filepath.Dir(dir)
		}
		return dir
	}
	if cwd, err := os.Getwd(); err == nil && system.Exists(filepath.Join(cwd, "assets")) {
		return cwd
	}
	return ""
}

func brainHome(home string) string {
	if override := os.Getenv("AGENTLINK_BRAIN_HOME"); override != "" {
		return override
	}
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, "Library", "Application Support", system.AppName, "assets")
}

func darwinRuntimeArch() string {
	if runtime.GOARCH == "amd64" {
		return "x86_64"
	}
	return runtime.GOARCH
}

func locateModel(loc AssetLocations, filename string) (string, bool, bool) {
	if override := os.Getenv("AGENTLINK_MODEL_PATH"); override != "" {
		return override, false, false
	}
	if loc.PackageRoot != "" {
		p := filepath.Join(loc.PackageRoot, "assets", "models", filename)
		if system.Exists(p) {
			return p, true, false
		}
	}
	p := filepath.Join(loc.BrainHome, "models", filename)
	if system.Exists(p) {
		return p, false, true
	}
	return p, false, false
}

func locateRuntime(loc AssetLocations) (string, bool, bool) {
	if override := os.Getenv("AGENTLINK_LLAMA_CLI"); override != "" {
		return override, false, false
	}
	arch := darwinRuntimeArch()
	if loc.PackageRoot != "" {
		p := filepath.Join(loc.PackageRoot, "assets", "runtimes", "llama.cpp", arch, "llama-cli")
		if system.Exists(p) {
			return p, true, false
		}
	}
	p := filepath.Join(loc.BrainHome, "runtimes", "llama.cpp", arch, "llama-cli")
	if system.Exists(p) {
		return p, false, true
	}
	if path, ok := findOnPath("llama-cli"); ok {
		return path, false, false
	}
	return p, false, false
}

func findOnPath(name string) (string, bool) {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, name)
		if system.CommandExists(path) {
			return path, true
		}
	}
	return "", false
}

func sha256File(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}
