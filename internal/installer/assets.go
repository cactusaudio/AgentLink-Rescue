package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ClashLock struct {
	SchemaVersion int    `json:"schemaVersion"`
	Version       string `json:"version"`
	AssetName     string `json:"assetName"`
	DownloadURL   string `json:"downloadURL"`
	SHA256        string `json:"sha256"`
	SizeBytes     int64  `json:"sizeBytes"`
	FetchedAt     string `json:"fetchedAt"`
	Arch          string `json:"arch"`
}

func ClashAssetPath() string {
	arch := RuntimeArch()
	dir := "macos-" + arch
	if arch == "amd64" {
		dir = "macos-amd64"
	}
	if arch == "arm64" {
		dir = "macos-arm64"
	}
	for _, root := range []string{packageRoot(), "."} {
		if root == "" {
			continue
		}
		base := filepath.Join(root, "assets", "installers", "clash-verge-rev", dir)
		matches, _ := filepath.Glob(filepath.Join(base, "*.dmg"))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func ClashLockPath() string {
	for _, root := range []string{packageRoot(), "."} {
		if root == "" {
			continue
		}
		path := filepath.Join(root, "assets", "installers", "clash-verge-rev", "manifest.lock.json")
		if exists(path) {
			return path
		}
	}
	return ""
}

func ReadClashLock() (ClashLock, bool) {
	path := ClashLockPath()
	if path == "" {
		return ClashLock{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ClashLock{}, false
	}
	var lock ClashLock
	if json.Unmarshal(data, &lock) != nil {
		return ClashLock{}, false
	}
	return lock, true
}
