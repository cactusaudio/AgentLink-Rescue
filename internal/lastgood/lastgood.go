package lastgood

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion     int         `json:"schemaVersion"`
	ToolVersion       string      `json:"toolVersion"`
	Action            string      `json:"action"`
	Status            string      `json:"status"`
	ProfileID         string      `json:"profileID,omitempty"`
	ProfilePath       string      `json:"profilePath,omitempty"`
	SavedItems        []SavedItem `json:"savedItems,omitempty"`
	Profiles          []Manifest  `json:"profiles,omitempty"`
	SnapshotID        string      `json:"snapshotID,omitempty"`
	SnapshotPath      string      `json:"snapshotPath,omitempty"`
	RollbackAvailable bool        `json:"rollbackAvailable"`
	Warnings          []string    `json:"warnings,omitempty"`
	NextAction        string      `json:"nextAction,omitempty"`
}

type Manifest struct {
	SchemaVersion int         `json:"schemaVersion"`
	ToolVersion   string      `json:"toolVersion"`
	ID            string      `json:"id"`
	Name          string      `json:"name,omitempty"`
	CreatedAt     string      `json:"createdAt"`
	Home          string      `json:"home"`
	Items         []SavedItem `json:"items"`
	Warnings      []string    `json:"warnings,omitempty"`
}

type SavedItem struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	OriginalPath string `json:"originalPath,omitempty"`
	StoredPath   string `json:"storedPath,omitempty"`
	Exists       bool   `json:"exists"`
	SHA256       string `json:"sha256,omitempty"`
	Mode         string `json:"mode,omitempty"`
	Restorable   bool   `json:"restorable"`
}

type Options struct {
	Home    string
	Version string
	Name    string
	ID      string
	Last    bool
	Yes     bool
}

func Save(ctx context.Context, runner command.Runner, opts Options) Report {
	_ = ctx
	_ = runner
	home := opts.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	version := opts.Version
	if version == "" {
		version = system.Version
	}
	id := time.Now().Format("20060102-150405")
	root := filepath.Join(system.UserLastGoodDir(home), id)
	rep := Report{SchemaVersion: 1, ToolVersion: version, Action: "save", Status: "saved", ProfileID: id, ProfilePath: root}
	if err := os.MkdirAll(filepath.Join(root, "files"), 0700); err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	m := Manifest{SchemaVersion: 1, ToolVersion: version, ID: id, Name: opts.Name, CreatedAt: time.Now().Format(time.RFC3339), Home: home}
	for _, spec := range itemSpecs(home) {
		item := saveItem(root, spec)
		m.Items = append(m.Items, item)
		if item.Exists {
			rep.SavedItems = append(rep.SavedItems, item)
		}
	}
	f := facts.Collect(ctx, runner, home, false)
	proxyPath := filepath.Join(root, "proxy-readiness.json")
	if data, err := json.MarshalIndent(map[string]any{"proxyEnv": f.ProxyEnv, "ports": f.Ports}, "", "  "); err == nil {
		_ = os.WriteFile(proxyPath, data, 0600)
		m.Items = append(m.Items, SavedItem{ID: "proxy-readiness", Kind: "report", StoredPath: proxyPath, Exists: true, Restorable: false})
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	_ = os.WriteFile(filepath.Join(system.UserLastGoodDir(home), "latest"), []byte(id), 0600)
	rep.NextAction = "Use last-good inspect " + id + " or last-good restore --id " + id + " --yes."
	return rep
}

func List(home, version string) Report {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	rep := Report{SchemaVersion: 1, ToolVersion: version, Action: "list", Status: "ok"}
	entries, err := os.ReadDir(system.UserLastGoodDir(home))
	if err != nil {
		rep.Status = "empty"
		return rep
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if m, err := Load(home, entry.Name()); err == nil {
			rep.Profiles = append(rep.Profiles, m)
		}
	}
	sort.Slice(rep.Profiles, func(i, j int) bool { return rep.Profiles[i].ID < rep.Profiles[j].ID })
	return rep
}

func Inspect(home, id, version string) Report {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	m, err := Load(home, id)
	rep := Report{SchemaVersion: 1, ToolVersion: version, Action: "inspect", ProfileID: id, ProfilePath: filepath.Join(system.UserLastGoodDir(home), filepath.Base(id))}
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	rep.Status = "ok"
	rep.SavedItems = m.Items
	rep.Profiles = []Manifest{m}
	return rep
}

func Restore(ctx context.Context, runner command.Runner, opts Options) Report {
	home := opts.Home
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	version := opts.Version
	if version == "" {
		version = system.Version
	}
	id := opts.ID
	if opts.Last || id == "" {
		latest, err := latestID(home)
		if err != nil {
			return Report{SchemaVersion: 1, ToolVersion: version, Action: "restore", Status: "failed", Warnings: []string{err.Error()}}
		}
		id = latest
	}
	rep := Report{SchemaVersion: 1, ToolVersion: version, Action: "restore", ProfileID: id, ProfilePath: filepath.Join(system.UserLastGoodDir(home), filepath.Base(id))}
	m, err := Load(home, id)
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	if !opts.Yes {
		rep.Status = "dry_run"
		rep.SavedItems = restorableItems(m)
		rep.NextAction = "Run last-good restore --id " + m.ID + " --yes to restore saved config files."
		return rep
	}
	profileRoot := filepath.Join(system.UserLastGoodDir(home), filepath.Base(id))
	targets := originalPaths(home, profileRoot, m)
	rp, err := snapshot.NewRestorePointWithPolicy(system.UserRestorePointsDir(home), version, system.MutationOptions{RealUserHome: home, ExtraAllowedPaths: targets})
	if err != nil {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, err.Error())
		return rep
	}
	for _, item := range m.Items {
		if !item.Restorable || item.OriginalPath == "" || item.StoredPath == "" || !system.Exists(item.StoredPath) {
			continue
		}
		if err := validateRestoreItem(home, profileRoot, item); err != nil {
			rep.Warnings = append(rep.Warnings, "skipping "+item.ID+": "+safety.RedactSensitive(err.Error()))
			continue
		}
		if system.Exists(item.OriginalPath) {
			if _, err := rp.BackupPath(item.OriginalPath); err != nil {
				rep.Warnings = append(rep.Warnings, "backup failed for "+item.OriginalPath+": "+safety.RedactSensitive(err.Error()))
				continue
			}
		} else if _, err := rp.RecordCreatedPath(item.OriginalPath); err != nil {
			rep.Warnings = append(rep.Warnings, "created-path record failed for "+item.OriginalPath+": "+safety.RedactSensitive(err.Error()))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(item.OriginalPath), 0755); err != nil {
			rep.Warnings = append(rep.Warnings, err.Error())
			continue
		}
		data, err := os.ReadFile(item.StoredPath)
		if err != nil {
			rep.Warnings = append(rep.Warnings, err.Error())
			continue
		}
		mode := restoreMode(item)
		if err := os.WriteFile(item.OriginalPath, data, mode); err != nil {
			rep.Warnings = append(rep.Warnings, err.Error())
			continue
		}
		_ = os.Chmod(item.OriginalPath, mode)
		rep.SavedItems = append(rep.SavedItems, item)
	}
	if len(rep.SavedItems) == 0 {
		rep.Status = "failed"
		rep.Warnings = append(rep.Warnings, "no valid last-good entries restored")
		return rep
	}
	rep.Status = "restored"
	rep.SnapshotID = rp.Manifest.ID
	rep.SnapshotPath = rp.Path
	rep.RollbackAvailable = true
	rep.NextAction = "Rollback current restore with: agentlink restore last"
	_ = rp.WriteHumanReport("Last-good restore: " + m.ID + "\nRollback: agentlink restore last\n")
	return rep
}

func Load(home, id string) (Manifest, error) {
	var m Manifest
	data, err := os.ReadFile(filepath.Join(system.UserLastGoodDir(home), filepath.Base(id), "manifest.json"))
	if err != nil {
		return m, err
	}
	return m, json.Unmarshal(data, &m)
}

type itemSpec struct {
	id   string
	kind string
	path string
}

func itemSpecs(home string) []itemSpec {
	return []itemSpec{
		{"codex-config", "config", filepath.Join(home, ".codex", "config.toml")},
		{"claude-settings", "config", filepath.Join(home, ".claude", "settings.json")},
		{"claude-json", "config", filepath.Join(home, ".claude.json")},
		{"gemini-settings", "config", filepath.Join(home, ".gemini", "settings.json")},
		{"gemini-config", "config", filepath.Join(home, ".config", "gemini", "settings.json")},
	}
}

func saveItem(root string, spec itemSpec) SavedItem {
	item := SavedItem{ID: spec.id, Kind: spec.kind, OriginalPath: spec.path, Exists: system.Exists(spec.path), Restorable: true}
	if !item.Exists {
		return item
	}
	info, err := os.Lstat(spec.path)
	if err != nil || info.IsDir() || info.Mode().Type() != 0 {
		item.Restorable = false
		return item
	}
	item.Mode = fmt.Sprintf("%04o", uint32(info.Mode().Perm()))
	stored := filepath.Join(root, "files", strings.TrimPrefix(filepath.Clean(spec.path), string(os.PathSeparator)))
	if err := os.MkdirAll(filepath.Dir(stored), 0700); err != nil {
		item.Restorable = false
		return item
	}
	data, err := os.ReadFile(spec.path)
	if err != nil {
		item.Restorable = false
		return item
	}
	if err := os.WriteFile(stored, data, 0600); err != nil {
		item.Restorable = false
		return item
	}
	item.StoredPath = stored
	sum := sha256.Sum256(data)
	item.SHA256 = hex.EncodeToString(sum[:])
	return item
}

func latestID(home string) (string, error) {
	data, err := os.ReadFile(filepath.Join(system.UserLastGoodDir(home), "latest"))
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}
	list := List(home, system.Version)
	if len(list.Profiles) == 0 {
		return "", fmt.Errorf("no last-good profiles found")
	}
	return list.Profiles[len(list.Profiles)-1].ID, nil
}

func restorableItems(m Manifest) []SavedItem {
	var out []SavedItem
	for _, item := range m.Items {
		if item.Restorable && item.Exists {
			out = append(out, item)
		}
	}
	return out
}

func originalPaths(home, profileRoot string, m Manifest) []string {
	var out []string
	for _, item := range m.Items {
		if item.Restorable && item.OriginalPath != "" && validateRestoreItem(home, profileRoot, item) == nil {
			out = append(out, item.OriginalPath)
		}
	}
	return out
}

func validateRestoreItem(home, profileRoot string, item SavedItem) error {
	allowed, err := allowedConfigPaths(home)
	if err != nil {
		return err
	}
	original := filepath.Clean(item.OriginalPath)
	if original != item.OriginalPath || !filepath.IsAbs(original) {
		return fmt.Errorf("refusing ambiguous restore path: %s", item.OriginalPath)
	}
	if !allowed[original] {
		return fmt.Errorf("restore path outside AI config allowlist: %s", original)
	}
	stored := filepath.Clean(item.StoredPath)
	if stored != item.StoredPath || !filepath.IsAbs(stored) {
		return fmt.Errorf("refusing ambiguous stored path: %s", item.StoredPath)
	}
	root := filepath.Clean(profileRoot)
	rel, err := filepath.Rel(root, stored)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("stored path outside selected profile: %s", stored)
	}
	return nil
}

func allowedConfigPaths(home string) (map[string]bool, error) {
	if home == "" {
		return nil, fmt.Errorf("home is required")
	}
	cleanHome := filepath.Clean(home)
	if !filepath.IsAbs(cleanHome) {
		return nil, fmt.Errorf("home must be absolute")
	}
	out := map[string]bool{}
	for _, spec := range itemSpecs(cleanHome) {
		clean := filepath.Clean(spec.path)
		rel, err := filepath.Rel(cleanHome, clean)
		if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("configured path outside home: %s", clean)
		}
		out[clean] = true
	}
	return out, nil
}

func restoreMode(item SavedItem) os.FileMode {
	if item.Mode == "" {
		return 0600
	}
	var mode uint32
	if _, err := fmt.Sscanf(item.Mode, "%o", &mode); err != nil {
		return 0600
	}
	if mode == 0 {
		return 0600
	}
	return os.FileMode(mode) & 0777
}
