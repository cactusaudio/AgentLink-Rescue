package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/configfile"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/safety"
	"cactus-agentlink-rescue/internal/session"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/system"
	"cactus-agentlink-rescue/internal/verifier"
)

type RunOptions struct {
	Home        string
	RecipesDir  string
	DryRun      bool
	Yes         bool
	JSON        bool
	Params      map[string]string
	CommandLine []string
}

type RunResult struct {
	ToolVersion       string            `json:"toolVersion"`
	RecipeID          string            `json:"recipeId"`
	DryRun            bool              `json:"dryRun"`
	Status            string            `json:"status"`
	Warnings          []string          `json:"warnings,omitempty"`
	SnapshotID        string            `json:"snapshotId,omitempty"`
	SnapshotPath      string            `json:"snapshotPath,omitempty"`
	ChangedFiles      []string          `json:"changedFiles"`
	PlannedActions    []DryRunAction    `json:"plannedActions,omitempty"`
	VerifierResults   []verifier.Result `json:"verifierResults,omitempty"`
	SessionID         string            `json:"sessionId,omitempty"`
	HumanReportPath   string            `json:"humanReportPath,omitempty"`
	AgentDispatchPath string            `json:"agentDispatchPath,omitempty"`
	Error             string            `json:"error,omitempty"`
}

const (
	StatusSuccess                 = "success"
	StatusSuccessWithWarnings     = "success_with_warnings"
	StatusConfigTemplateGenerated = "config_template_generated"
	StatusNeedsUserSecret         = "needs_user_secret"
	StatusNeedsOnlineSmokeTest    = "needs_online_smoke_test"
	StatusVerifierFailed          = "verifier_failed"
	StatusRolledBack              = "rolled_back"
	StatusFail                    = "fail"
	StatusSkipped                 = "skipped"
	StatusDryRun                  = "dry-run"
)

func Run(ctx context.Context, runner command.Runner, reg Registry, id string, opts RunOptions) RunResult {
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	home := opts.Home
	if home == "" {
		home = HomeDir()
	}
	res := RunResult{ToolVersion: system.Version, RecipeID: id, DryRun: opts.DryRun, ChangedFiles: []string{}}
	r, ok := reg.Get(id)
	if !ok {
		res.Status = StatusFail
		res.Error = "unknown recipe"
		return res
	}
	if !SupportsOS(r, "darwin") {
		res.Status = StatusSkipped
		res.Error = "unsupported OS"
		return res
	}
	if r.Risk == RiskDestructiveAction {
		res.Status = StatusFail
		res.Error = "destructive_action is refused in v0.3"
		return res
	}
	if !opts.DryRun && r.Risk != RiskReadOnly && r.Risk != RiskSafePatch && !opts.Yes {
		res.Status = StatusFail
		res.Error = "recipe risk requires --yes"
		return res
	}
	if !opts.DryRun && r.RequiresRoot && !system.IsRoot() {
		res.Status = StatusFail
		res.Error = "recipe requires root"
		return res
	}
	params, err := ResolveParams(r, opts.Params)
	if err != nil {
		res.Status = StatusFail
		res.Error = err.Error()
		return res
	}
	res.Warnings = append(res.Warnings, recipeParamWarnings(r, params)...)
	sess := session.New(system.Version, opts.CommandLine)
	sess.SelectedRecipe = r.ID
	sess.RiskLevel = r.Risk
	sess.DryRun = opts.DryRun
	sess.InitialFacts = facts.Collect(ctx, runner, home, false)
	sess.RealUser = sess.InitialFacts.RealUser
	sess.HostSummary = sess.InitialFacts.OS + "/" + sess.InitialFacts.Arch

	sess.Warnings = append(sess.Warnings, res.Warnings...)

	if err := checkPreconditions(ctx, runner, home, r, params, opts.Yes); err != nil {
		res.Status = StatusFail
		res.Error = err.Error()
		sess.Finish(StatusFail)
		_ = session.NewStore(home).Save(&sess)
		return finishResult(res, sess)
	}
	holder := &snapshotHolder{home: home, recipe: r, params: params}
	for _, patch := range r.Patches {
		if !patchEnabled(patch, params) {
			continue
		}
		planned := DryRunAction{ID: patch.ID, Description: patch.Type, Path: Interpolate(patch.Path, home, params)}
		if opts.DryRun {
			res.PlannedActions = append(res.PlannedActions, planned)
			continue
		}
		changed, err := applyPatch(ctx, runner, holder, home, params, patch, &sess)
		if err != nil {
			res.Status = StatusFail
			res.Error = err.Error()
			sess.Finish(StatusFail)
			_ = session.NewStore(home).Save(&sess)
			return finishResult(res, sess)
		}
		if changed != "" {
			res.ChangedFiles = appendIfMissing(res.ChangedFiles, changed)
			sess.ChangedFiles = appendIfMissing(sess.ChangedFiles, changed)
		}
	}
	if opts.DryRun {
		res.Status = StatusDryRun
		sess.Finish(StatusDryRun)
		_ = session.NewStore(home).Save(&sess)
		return finishResult(res, sess)
	}
	if holder.rp != nil {
		res.SnapshotID = holder.rp.Manifest.ID
		res.SnapshotPath = holder.rp.Path
		sess.SnapshotID = holder.rp.Manifest.ID
		sess.SnapshotPath = holder.rp.Path
		sess.RollbackAvailable = true
	}
	vreg := verifier.NewRegistry()
	vctx := verifier.Context{Context: ctx, Runner: runner, Home: home, Params: params}
	failed := false
	warned := len(res.Warnings) > 0
	for _, ref := range r.Verify {
		if !verifierEnabled(ref, params) {
			continue
		}
		args := interpolateMap(ref.Args, home, params)
		vr := vreg.Run(vctx, ref.ID, args)
		res.VerifierResults = append(res.VerifierResults, vr)
		sess.VerifierResults = append(sess.VerifierResults, vr)
		if vr.Status == "fail" {
			failed = true
		}
		if vr.Status == "warn" {
			warned = true
		}
	}
	if failed {
		res.Status = StatusVerifierFailed
	} else {
		res.Status = finalRecipeStatus(r, res, warned)
	}
	if r.ID == "codex-deepseek-provider-config" && res.Status != StatusVerifierFailed {
		res.Warnings = appendIfMissing(res.Warnings, "static config template generated")
		res.Warnings = appendIfMissing(res.Warnings, "provider_smoke_test_required")
		res.Warnings = appendIfMissing(res.Warnings, "not proven operational until explicit online smoke test passes")
	}
	sess.Warnings = append([]string(nil), res.Warnings...)
	sess.Finish(res.Status)
	_ = session.NewStore(home).Save(&sess)
	return finishResult(res, sess)
}

type snapshotHolder struct {
	home   string
	recipe Recipe
	params map[string]string
	rp     *snapshot.RestorePoint
}

func (h *snapshotHolder) ensure() error {
	if h.rp != nil {
		return nil
	}
	rp, err := snapshot.NewRestorePointWithPolicy(system.UserRestorePointsDir(h.home), system.Version, system.MutationOptions{RealUserHome: h.home, ExtraAllowedPaths: recipeMutationPaths(h.recipe, h.home, h.params)})
	if err != nil {
		return err
	}
	h.rp = &rp
	return nil
}

func recipeMutationPaths(r Recipe, home string, params map[string]string) []string {
	var out []string
	for _, p := range r.Patches {
		path := Interpolate(p.Path, home, params)
		if path != "" {
			out = append(out, path)
		}
	}
	return out
}

func finishResult(res RunResult, sess session.Session) RunResult {
	res.SessionID = sess.SessionID
	res.HumanReportPath = sess.HumanReportPath
	res.AgentDispatchPath = sess.AgentDispatchPath
	if res.SnapshotID == "" {
		res.SnapshotID = sess.SnapshotID
		res.SnapshotPath = sess.SnapshotPath
	}
	if len(res.ChangedFiles) == 0 {
		res.ChangedFiles = append([]string(nil), sess.ChangedFiles...)
	}
	return res
}

func checkPreconditions(ctx context.Context, runner command.Runner, home string, r Recipe, params map[string]string, yes bool) error {
	for _, p := range r.Preconditions {
		ok := false
		switch p.Type {
		case "os_is":
			ok = p.Value == "" || p.Value == "darwin"
		case "shell_is":
			ok = strings.Contains(os.Getenv("SHELL"), p.Value)
		case "file_exists":
			ok = system.Exists(Interpolate(p.Path, home, params))
		case "file_exists_or_creatable":
			path := Interpolate(p.Path, home, params)
			ok = fileExistsOrCreatable(path)
		case "command_exists":
			_, ok = findCommand(p.Command)
		case "command_missing":
			_, exists := findCommand(p.Command)
			ok = !exists
		case "env_present":
			ok = os.Getenv(p.Name) != ""
		case "env_missing":
			ok = os.Getenv(p.Name) == ""
		case "port_listening":
			ok = portListening(p.Port)
		case "config_file_readable":
			path := Interpolate(p.Path, home, params)
			f, err := os.Open(path)
			if err == nil {
				_ = f.Close()
				ok = true
			}
		}
		if !ok && !(yes && p.OptionalWithYes) {
			return fmt.Errorf("precondition failed: %s", p.Type)
		}
		_ = ctx
		_ = runner
	}
	return nil
}

func applyPatch(ctx context.Context, runner command.Runner, holder *snapshotHolder, home string, params map[string]string, patch Patch, sess *session.Session) (string, error) {
	path := Interpolate(patch.Path, home, params)
	switch patch.Type {
	case "append_managed_block_if_missing", "set_env_managed_block", "update_toml_section_with_backup":
		if configfile.CountManagedBlocks(path, patch.Marker) > 1 {
			return "", fmt.Errorf("duplicate managed block detected: %s", patch.Marker)
		}
		if !configfile.HasManagedBlock(path, patch.Marker) {
			if err := snapshotFileBeforePatch(holder, path); err != nil {
				return "", err
			}
		}
		pr := configfile.AppendManagedBlockIfMissing(path, patch.Marker, Interpolate(patch.Content, home, params))
		if pr.Message != "" && !pr.Changed && strings.Contains(pr.Message, "error") {
			return "", fmt.Errorf("%s", pr.Message)
		}
		if pr.Changed {
			return path, nil
		}
		return "", nil
	case "remove_managed_block":
		if configfile.HasManagedBlock(path, patch.Marker) {
			if err := snapshotFileBeforePatch(holder, path); err != nil {
				return "", err
			}
		}
		pr := configfile.RemoveManagedBlock(path, patch.Marker)
		if pr.Changed {
			return path, nil
		}
		return "", nil
	case "write_file_from_template_with_backup":
		if patch.OnlyIfTomlMalformed && system.Exists(path) && configfile.TOMLBasicSyntaxOK(path) == nil {
			return "", nil
		}
		if err := snapshotFileBeforePatch(holder, path); err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(Interpolate(patch.Template, home, params)), 0644); err != nil {
			return "", err
		}
		return path, nil
	case "update_json_field_with_backup":
		return updateJSONField(holder, path, patch.Key, Interpolate(patch.Value, home, params))
	case "unset_git_config_key":
		return configPatch(ctx, runner, holder, "git", patch.Key, "", "unset", sess)
	case "unset_npm_config_key":
		return configPatch(ctx, runner, holder, "npm", patch.Key, "", "unset", sess)
	case "set_git_config_key":
		return configPatch(ctx, runner, holder, "git", patch.Key, Interpolate(patch.Value, home, params), "set", sess)
	case "set_npm_config_key":
		return configPatch(ctx, runner, holder, "npm", patch.Key, Interpolate(patch.Value, home, params), "set", sess)
	default:
		return "", fmt.Errorf("unsupported patch type: %s", patch.Type)
	}
}

func snapshotFileBeforePatch(holder *snapshotHolder, path string) error {
	if holder == nil {
		return fmt.Errorf("snapshot holder missing before writable patch")
	}
	if err := holder.ensure(); err != nil {
		return err
	}
	if system.Exists(path) {
		_, err := holder.rp.BackupPath(path)
		return err
	}
	_, err := holder.rp.RecordCreatedPath(path)
	return err
}

func updateJSONField(holder *snapshotHolder, path, key, value string) (string, error) {
	if err := snapshotFileBeforePatch(holder, path); err != nil {
		return "", err
	}
	data, _ := os.ReadFile(path)
	var m map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return "", err
		}
	} else {
		m = map[string]any{}
	}
	m[key] = value
	out, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func configPatch(ctx context.Context, runner command.Runner, holder *snapshotHolder, tool string, key string, value string, action string, sess *session.Session) (string, error) {
	if key == "" {
		return "", fmt.Errorf("config key missing")
	}
	if holder != nil {
		if err := holder.ensure(); err != nil {
			return "", err
		}
	}
	if runner == nil {
		r := command.NewExecRunner()
		runner = r
	}
	bin, getArgs, setArgs, unsetArgs, scope, err := configPatchCommands(tool, key, value)
	if err != nil {
		return "", err
	}
	getRes := runner.Run(ctx, bin, getArgs...)
	logCommand(sess, "record_original_"+tool+"_"+key, getRes)
	if holder != nil && holder.rp != nil {
		_ = holder.rp.LogCommand("record_original_"+tool+"_"+key, getRes, true)
	}
	oldValue := strings.TrimSpace(getRes.Stdout)
	existed := getRes.ExitCode == 0 && oldValue != "" && oldValue != "null" && oldValue != "undefined"
	if tool == "npm" && !existed {
		if val, ok := readNPMRCValue(key); ok {
			oldValue = val
			existed = true
		}
	}
	if holder != nil && holder.rp != nil {
		if err := holder.rp.RecordConfigValue(tool, scope, key, existed, oldValue); err != nil {
			return "", err
		}
	}
	args := unsetArgs
	if action == "set" {
		args = setArgs
	}
	mutateRes := runner.Run(ctx, bin, args...)
	logCommand(sess, action+"_"+tool+"_"+key, mutateRes)
	if holder != nil && holder.rp != nil {
		_ = holder.rp.LogCommand(action+"_"+tool+"_"+key, mutateRes, true)
	}
	if mutateRes.ExitCode != 0 && (action == "set" || existed) {
		return "", fmt.Errorf("%s", safety.RedactSensitive(mutateRes.Stderr+mutateRes.Error))
	}
	return tool + ":" + scope + ":" + key, nil
}

func configPatchCommands(tool, key, value string) (string, []string, []string, []string, string, error) {
	switch tool {
	case "git":
		return configToolPath("git"),
			[]string{"config", "--global", "--get", key},
			[]string{"config", "--global", key, value},
			[]string{"config", "--global", "--unset", key},
			"global",
			nil
	case "npm":
		return configToolPath("npm"),
			[]string{"config", "get", key},
			[]string{"config", "set", key, value},
			[]string{"config", "delete", key},
			"user",
			nil
	default:
		return "", nil, nil, nil, "", fmt.Errorf("unsupported config tool: %s", tool)
	}
}

func configToolPath(tool string) string {
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

func readNPMRCValue(key string) (string, bool) {
	path := os.Getenv("NPM_CONFIG_USERCONFIG")
	if path == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", false
		}
		path = filepath.Join(home, ".npmrc")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == key {
			return strings.Trim(strings.TrimSpace(parts[1]), `"'`), true
		}
	}
	return "", false
}

func logCommand(sess *session.Session, id string, res command.Result) {
	if sess == nil {
		return
	}
	sess.CommandsRun = append(sess.CommandsRun, snapshot.ActionLog{
		Timestamp:         time.Now().Format(time.RFC3339),
		ActionID:          id,
		Command:           safety.RedactSensitive(res.Command),
		ExitCode:          res.ExitCode,
		RedactedStdout:    safety.RedactSensitive(res.Stdout),
		RedactedStderr:    safety.RedactSensitive(res.Stderr),
		RollbackAvailable: true,
	})
}

func patchEnabled(p Patch, params map[string]string) bool {
	for k, v := range p.WhenParamEquals {
		if params[k] != v {
			return false
		}
	}
	return true
}

func verifierEnabled(v VerifierRef, params map[string]string) bool {
	for k, want := range v.WhenParamEquals {
		if params[k] != want {
			return false
		}
	}
	return true
}

func interpolateMap(in map[string]string, home string, params map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = Interpolate(v, home, params)
	}
	return out
}

func findCommand(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		path := filepath.Join(dir, name)
		if system.CommandExists(path) {
			return path, true
		}
	}
	return "", false
}

func fileExistsOrCreatable(path string) bool {
	if system.Exists(path) {
		return true
	}
	dir := filepath.Dir(path)
	for dir != "" && dir != "." && dir != string(os.PathSeparator) {
		if system.Exists(dir) {
			info, err := os.Stat(dir)
			return err == nil && info.IsDir()
		}
		dir = filepath.Dir(dir)
	}
	return system.Exists(dir)
}

func portListening(port string) bool {
	if port == "" {
		return false
	}
	conn, err := netDialTimeout("tcp", "127.0.0.1:"+port, 250*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

var netDialTimeout = func(network, address string, timeout time.Duration) (interface{ Close() error }, error) {
	return (&net.Dialer{Timeout: timeout}).Dial(network, address)
}

func appendIfMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func recipeParamWarnings(r Recipe, params map[string]string) []string {
	if r.ID != "codex-deepseek-provider-config" {
		return nil
	}
	switch params["model"] {
	case "deepseek-chat", "deepseek-reasoner":
		return []string{"legacy DeepSeek compatibility model; scheduled deprecation per DeepSeek docs"}
	default:
		return nil
	}
}

func finalRecipeStatus(r Recipe, res RunResult, warned bool) string {
	switch r.ID {
	case "codex-deepseek-provider-config":
		for _, vr := range res.VerifierResults {
			if vr.ID == "codex_provider_key_env_present_redacted" && vr.Status == "warn" {
				return StatusNeedsUserSecret
			}
		}
		return StatusNeedsOnlineSmokeTest
	case "codex-config-parse-repair":
		if len(res.ChangedFiles) > 0 {
			return StatusConfigTemplateGenerated
		}
	}
	if warned {
		return StatusSuccessWithWarnings
	}
	return StatusSuccess
}
