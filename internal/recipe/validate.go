package recipe

import (
	"fmt"
	"runtime"
)

var validRisks = map[string]int{
	RiskReadOnly:          0,
	RiskSafePatch:         1,
	RiskReversiblePatch:   2,
	RiskNetworkAction:     3,
	RiskPrivilegedAction:  4,
	RiskDestructiveAction: 5,
}

var validPreconditions = map[string]bool{
	"os_is": true, "shell_is": true, "file_exists": true, "file_exists_or_creatable": true,
	"command_exists": true, "command_missing": true, "env_present": true, "env_missing": true,
	"port_listening": true, "config_file_readable": true,
}

var validPatches = map[string]bool{
	"append_managed_block_if_missing":      true,
	"remove_managed_block":                 true,
	"set_env_managed_block":                true,
	"write_file_from_template_with_backup": true,
	"update_json_field_with_backup":        true,
	"update_toml_section_with_backup":      true,
	"set_git_config_key":                   true,
	"set_npm_config_key":                   true,
	"unset_git_config_key":                 true,
	"unset_npm_config_key":                 true,
}

var rejectedPatches = map[string]bool{
	"raw_shell":                     true,
	"arbitrary_delete":              true,
	"recursive_chmod":               true,
	"recursive_chown":               true,
	"curl_pipe_shell":               true,
	"overwrite_file_without_backup": true,
}

func ValidateRecipe(r Recipe) error {
	if r.SchemaVersion <= 0 || r.ID == "" || r.Title == "" {
		return fmt.Errorf("missing required recipe metadata")
	}
	if _, ok := validRisks[r.Risk]; !ok {
		return fmt.Errorf("invalid risk %q", r.Risk)
	}
	if r.Risk == RiskDestructiveAction {
		return fmt.Errorf("destructive_action is refused in v0.3")
	}
	for _, p := range r.Preconditions {
		if !validPreconditions[p.Type] {
			return fmt.Errorf("invalid precondition %q", p.Type)
		}
	}
	for _, p := range r.Patches {
		if rejectedPatches[p.Type] {
			return fmt.Errorf("rejected patch type %q", p.Type)
		}
		if !validPatches[p.Type] {
			return fmt.Errorf("invalid patch type %q", p.Type)
		}
	}
	if WritableRisk(r.Risk) && len(r.Patches) > 0 && len(r.Rollback) == 0 {
		return fmt.Errorf("writable recipe must define rollback")
	}
	return nil
}

func SupportsOS(r Recipe, osName string) bool {
	if len(r.SupportedOS) == 0 {
		return true
	}
	for _, item := range r.SupportedOS {
		if item == osName {
			return true
		}
	}
	return false
}

func RiskRank(risk string) int {
	return validRisks[risk]
}

func CurrentOSSupported(r Recipe) bool {
	return SupportsOS(r, runtime.GOOS)
}
