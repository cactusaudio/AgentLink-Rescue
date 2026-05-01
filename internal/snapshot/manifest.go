package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/system"
)

type Manifest struct {
	SchemaVersion           int           `json:"schemaVersion"`
	ToolVersion             string        `json:"toolVersion"`
	ID                      string        `json:"id"`
	CreatedAt               string        `json:"createdAt"`
	RootPath                string        `json:"rootPath"`
	PreflightPath           string        `json:"preflightPath,omitempty"`
	PostflightPath          string        `json:"postflightPath,omitempty"`
	CommandsLogPath         string        `json:"commandsLogPath,omitempty"`
	HumanReportPath         string        `json:"humanReportPath,omitempty"`
	PreviousNetworkLocation string        `json:"previousNetworkLocation,omitempty"`
	Entries                 []FileEntry   `json:"entries"`
	ConfigEntries           []ConfigEntry `json:"configEntries,omitempty"`
	Actions                 []ActionLog   `json:"actions"`
	RollbackStatus          string        `json:"rollbackStatus"`
	TouchedSystemPaths      bool          `json:"touchedSystemPaths"`
}

type FileEntry struct {
	OriginalPath      string   `json:"originalPath"`
	BackupPath        string   `json:"backupPath,omitempty"`
	QuarantinePath    string   `json:"quarantinePath,omitempty"`
	OwnerUID          int      `json:"ownerUid"`
	OwnerGID          int      `json:"ownerGid"`
	Mode              string   `json:"mode"`
	FileType          string   `json:"fileType"`
	SHA256            string   `json:"sha256,omitempty"`
	Xattrs            []string `json:"xattrs,omitempty"`
	Action            string   `json:"action"`
	Timestamp         string   `json:"timestamp"`
	RollbackAvailable bool     `json:"rollbackAvailable"`
}

type ConfigEntry struct {
	Tool              string `json:"tool"`
	Scope             string `json:"scope"`
	Key               string `json:"key"`
	Existed           bool   `json:"existed"`
	OldValue          string `json:"oldValue,omitempty"`
	Action            string `json:"action"`
	Timestamp         string `json:"timestamp"`
	RollbackAvailable bool   `json:"rollbackAvailable"`
}

type ActionLog struct {
	Timestamp         string `json:"timestamp"`
	ActionID          string `json:"actionId"`
	Command           string `json:"command"`
	ExitCode          int    `json:"exitCode"`
	RedactedStdout    string `json:"redactedStdout,omitempty"`
	RedactedStderr    string `json:"redactedStderr,omitempty"`
	RollbackAvailable bool   `json:"rollbackAvailable"`
}

type RestorePoint struct {
	Path     string
	Manifest Manifest
	Policy   system.MutationOptions
}

func NewRestorePoint(baseDir string, version string) (RestorePoint, error) {
	return NewRestorePointWithPolicy(baseDir, version, system.MutationOptions{})
}

func NewRestorePointWithPolicy(baseDir string, version string, policy system.MutationOptions) (RestorePoint, error) {
	id := GenerateID(time.Now())
	root := filepath.Join(baseDir, id)
	for i := 1; system.Exists(root); i++ {
		root = filepath.Join(baseDir, fmt.Sprintf("%s-%02d", id, i))
	}
	if err := os.MkdirAll(filepath.Join(root, "files"), 0755); err != nil {
		return RestorePoint{}, err
	}
	if err := os.MkdirAll(filepath.Join(root, "quarantine"), 0755); err != nil {
		return RestorePoint{}, err
	}
	rp := RestorePoint{
		Path:   root,
		Policy: policy,
		Manifest: Manifest{
			SchemaVersion:   1,
			ToolVersion:     version,
			ID:              filepath.Base(root),
			CreatedAt:       time.Now().Format(time.RFC3339),
			RootPath:        root,
			CommandsLogPath: filepath.Join(root, "commands.log"),
			HumanReportPath: filepath.Join(root, "human-report.txt"),
			RollbackStatus:  "not_rolled_back",
		},
	}
	return rp, rp.Save()
}

func GenerateID(t time.Time) string {
	return t.Format("20060102-150405")
}

func Load(path string) (RestorePoint, error) {
	manifestPath := filepath.Join(path, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return RestorePoint{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return RestorePoint{}, err
	}
	return RestorePoint{Path: path, Manifest: m}, nil
}

func (rp *RestorePoint) SetMutationPolicy(policy system.MutationOptions) {
	rp.Policy = policy
}

func Latest(baseDir string) (RestorePoint, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return RestorePoint{}, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(baseDir, e.Name()))
		}
	}
	sort.Strings(dirs)
	for i := len(dirs) - 1; i >= 0; i-- {
		rp, err := Load(dirs[i])
		if err == nil && rp.Manifest.RollbackStatus != "rolled_back" {
			return rp, nil
		}
	}
	return RestorePoint{}, os.ErrNotExist
}

func (rp *RestorePoint) Save() error {
	data, err := json.MarshalIndent(rp.Manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rp.Path, "manifest.json"), data, 0644)
}

func (rp *RestorePoint) WriteJSON(name string, v any) (string, error) {
	path := filepath.Join(rp.Path, name)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	switch name {
	case "preflight.json":
		rp.Manifest.PreflightPath = path
	case "postflight.json":
		rp.Manifest.PostflightPath = path
	}
	return path, rp.Save()
}

func (rp *RestorePoint) WriteHumanReport(text string) error {
	if err := os.WriteFile(rp.Manifest.HumanReportPath, []byte(text), 0644); err != nil {
		return err
	}
	return rp.Save()
}

func (rp *RestorePoint) LogCommand(actionID string, res command.Result, rollbackAvailable bool) error {
	entry := ActionLog{
		Timestamp:         time.Now().Format(time.RFC3339),
		ActionID:          actionID,
		Command:           safety.RedactSensitive(res.Command),
		ExitCode:          res.ExitCode,
		RedactedStdout:    oneLine(safety.RedactSensitive(res.Stdout)),
		RedactedStderr:    oneLine(safety.RedactSensitive(res.Stderr)),
		RollbackAvailable: rollbackAvailable,
	}
	rp.Manifest.Actions = append(rp.Manifest.Actions, entry)
	line := fmt.Sprintf("%s | %s | %s | %d | stdout=%s stderr=%s\n", entry.Timestamp, entry.ActionID, entry.Command, entry.ExitCode, entry.RedactedStdout, entry.RedactedStderr)
	if err := appendFile(rp.Manifest.CommandsLogPath, line); err != nil {
		return err
	}
	return rp.Save()
}

func (rp *RestorePoint) BackupPath(path string) (FileEntry, error) {
	if err := rp.validateMutationPath(path); err != nil {
		return FileEntry{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return FileEntry{}, err
	}
	entry := fileEntry(path, info, "backup")
	rel := strings.TrimPrefix(filepath.Clean(path), string(os.PathSeparator))
	target := filepath.Join(rp.Path, "files", rel)
	entry.BackupPath = target
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return FileEntry{}, err
	}
	if info.IsDir() {
		err = copyDir(path, target)
	} else {
		err = copyFileOrSymlink(path, target, info)
	}
	if err != nil {
		return FileEntry{}, err
	}
	entry.RollbackAvailable = true
	rp.Manifest.Entries = append(rp.Manifest.Entries, entry)
	if strings.HasPrefix(filepath.Clean(path), "/Library/") {
		rp.Manifest.TouchedSystemPaths = true
	}
	return entry, rp.Save()
}

func (rp *RestorePoint) QuarantinePath(path string) (FileEntry, error) {
	if err := rp.validateMutationPath(path); err != nil {
		return FileEntry{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return FileEntry{}, err
	}
	entry := fileEntry(path, info, "quarantine")
	rel := strings.TrimPrefix(filepath.Clean(path), string(os.PathSeparator))
	backupTarget := filepath.Join(rp.Path, "files", rel)
	entry.BackupPath = backupTarget
	target := uniquePath(filepath.Join(rp.Path, "quarantine", filepath.Base(path)))
	entry.QuarantinePath = target
	if err := os.MkdirAll(filepath.Dir(backupTarget), 0755); err != nil {
		return FileEntry{}, err
	}
	if info.IsDir() {
		err = copyDir(path, backupTarget)
	} else {
		err = copyFileOrSymlink(path, backupTarget, info)
	}
	if err != nil {
		return FileEntry{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return FileEntry{}, err
	}
	if err := os.Rename(path, target); err != nil {
		return FileEntry{}, err
	}
	entry.RollbackAvailable = true
	rp.Manifest.Entries = append(rp.Manifest.Entries, entry)
	if strings.HasPrefix(filepath.Clean(path), "/Library/") {
		rp.Manifest.TouchedSystemPaths = true
	}
	return entry, rp.Save()
}

func (rp *RestorePoint) RecordCreatedPath(path string) (FileEntry, error) {
	if err := rp.validateMutationPath(path); err != nil {
		return FileEntry{}, err
	}
	entry := FileEntry{
		OriginalPath:      path,
		FileType:          "created",
		Action:            "create",
		Timestamp:         time.Now().Format(time.RFC3339),
		RollbackAvailable: true,
	}
	rp.Manifest.Entries = append(rp.Manifest.Entries, entry)
	if strings.HasPrefix(filepath.Clean(path), "/Library/") {
		rp.Manifest.TouchedSystemPaths = true
	}
	return entry, rp.Save()
}

func (rp *RestorePoint) RestoreAll() error {
	if err := rp.restoreFileEntries(); err != nil {
		return err
	}
	rp.Manifest.RollbackStatus = "rolled_back"
	return rp.Save()
}

func (rp *RestorePoint) RestoreAllWithRunner(ctx context.Context, runner command.Runner) error {
	var errs []string
	if err := rp.restoreFileEntries(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := rp.RestoreConfigEntries(ctx, runner); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	rp.Manifest.RollbackStatus = "rolled_back"
	return rp.Save()
}

func (rp *RestorePoint) restoreFileEntries() error {
	var errs []string
	for i := len(rp.Manifest.Entries) - 1; i >= 0; i-- {
		entry := rp.Manifest.Entries[i]
		if entry.QuarantinePath != "" {
			if err := rp.restoreQuarantine(entry); err != nil {
				errs = append(errs, err.Error())
			}
			continue
		}
		if entry.Action == "create" && entry.BackupPath == "" {
			if err := rp.removeCreatedPath(entry); err != nil {
				errs = append(errs, err.Error())
			}
			continue
		}
		if entry.BackupPath != "" {
			if err := rp.restoreBackup(entry); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (rp *RestorePoint) RecordConfigValue(tool, scope, key string, existed bool, oldValue string) error {
	entry := ConfigEntry{
		Tool:              tool,
		Scope:             scope,
		Key:               key,
		Existed:           existed,
		OldValue:          oldValue,
		Action:            "record_config_value",
		Timestamp:         time.Now().Format(time.RFC3339),
		RollbackAvailable: true,
	}
	rp.Manifest.ConfigEntries = append(rp.Manifest.ConfigEntries, entry)
	return rp.Save()
}

func (rp *RestorePoint) RestoreConfigEntries(ctx context.Context, runner command.Runner) error {
	if len(rp.Manifest.ConfigEntries) == 0 {
		return nil
	}
	if runner == nil {
		runner = command.NewExecRunner()
	}
	var errs []string
	for i := len(rp.Manifest.ConfigEntries) - 1; i >= 0; i-- {
		entry := rp.Manifest.ConfigEntries[i]
		path, args, err := configRestoreCommand(entry)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		res := runner.Run(ctx, path, args...)
		_ = rp.LogCommand("restore_config_"+entry.Tool+"_"+entry.Key, res, false)
		if res.ExitCode != 0 && entry.Existed {
			errs = append(errs, safety.RedactSensitive(res.Stderr+res.Error))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return rp.Save()
}

func configRestoreCommand(entry ConfigEntry) (string, []string, error) {
	switch entry.Tool {
	case "git":
		if entry.Existed {
			return restoreConfigToolPath("git"), []string{"config", "--global", entry.Key, entry.OldValue}, nil
		}
		return restoreConfigToolPath("git"), []string{"config", "--global", "--unset", entry.Key}, nil
	case "npm":
		if entry.Existed {
			return restoreConfigToolPath("npm"), []string{"config", "set", entry.Key, entry.OldValue}, nil
		}
		return restoreConfigToolPath("npm"), []string{"config", "delete", entry.Key}, nil
	default:
		return "", nil, fmt.Errorf("unsupported config restore tool: %s", entry.Tool)
	}
}

func restoreConfigToolPath(tool string) string {
	switch tool {
	case "git":
		if path, ok := system.FindFirstExisting("/usr/bin/git", "/opt/homebrew/bin/git", "/usr/local/bin/git"); ok {
			return path
		}
		return "/usr/bin/git"
	case "npm":
		if path, ok := system.FindFirstExisting("/usr/bin/npm", "/opt/homebrew/bin/npm", "/usr/local/bin/npm"); ok {
			return path
		}
		return "/usr/bin/npm"
	default:
		return tool
	}
}

func fileEntry(path string, info fs.FileInfo, action string) FileEntry {
	entry := FileEntry{
		OriginalPath:      path,
		Mode:              info.Mode().String(),
		FileType:          fileType(info),
		Action:            action,
		Timestamp:         time.Now().Format(time.RFC3339),
		RollbackAvailable: false,
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		entry.OwnerUID = int(st.Uid)
		entry.OwnerGID = int(st.Gid)
	}
	if !info.IsDir() && info.Mode().Type() == 0 {
		entry.SHA256 = sha256File(path)
	}
	return entry
}

func fileType(info fs.FileInfo) string {
	switch {
	case info.IsDir():
		return "directory"
	case info.Mode()&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "file"
	}
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

func copyFileOrSymlink(src, dst string, info fs.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	_ = os.Chtimes(dst, time.Now(), info.ModTime())
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return copyFileOrSymlink(path, target, info)
	})
}

func (rp *RestorePoint) restoreBackup(entry FileEntry) error {
	if entry.BackupPath == "" || entry.OriginalPath == "" {
		return nil
	}
	if err := rp.validateMutationPath(entry.OriginalPath); err != nil {
		return err
	}
	info, err := os.Lstat(entry.BackupPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(entry.OriginalPath), 0755); err != nil {
		return err
	}
	if info.IsDir() {
		_ = os.RemoveAll(entry.OriginalPath)
		if err := copyDir(entry.BackupPath, entry.OriginalPath); err != nil {
			return err
		}
	} else {
		if err := copyFileOrSymlink(entry.BackupPath, entry.OriginalPath, info); err != nil {
			return err
		}
	}
	_ = os.Chown(entry.OriginalPath, entry.OwnerUID, entry.OwnerGID)
	return nil
}

func (rp *RestorePoint) restoreQuarantine(entry FileEntry) error {
	if entry.QuarantinePath == "" || entry.OriginalPath == "" {
		return nil
	}
	if err := rp.validateMutationPath(entry.OriginalPath); err != nil {
		return err
	}
	if _, err := os.Lstat(entry.QuarantinePath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(entry.OriginalPath), 0755); err != nil {
		return err
	}
	if system.Exists(entry.OriginalPath) {
		return fmt.Errorf("refusing to overwrite existing path during rollback: %s", entry.OriginalPath)
	}
	if err := os.Rename(entry.QuarantinePath, entry.OriginalPath); err != nil {
		return err
	}
	_ = os.Chown(entry.OriginalPath, entry.OwnerUID, entry.OwnerGID)
	return nil
}

func (rp *RestorePoint) removeCreatedPath(entry FileEntry) error {
	if entry.OriginalPath == "" {
		return nil
	}
	if err := rp.validateMutationPath(entry.OriginalPath); err != nil {
		return err
	}
	if !system.Exists(entry.OriginalPath) {
		return nil
	}
	info, err := os.Lstat(entry.OriginalPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("refusing to remove created directory during rollback: %s", entry.OriginalPath)
	}
	return os.Remove(entry.OriginalPath)
}

func (rp *RestorePoint) validateMutationPath(path string) error {
	return system.IsAllowedMutationPath(path, rp.Policy)
}

func appendFile(path, line string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

func uniquePath(path string) string {
	if !system.Exists(path) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%02d%s", base, i, ext)
		if !system.Exists(candidate) {
			return candidate
		}
	}
}
