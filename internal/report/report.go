package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/system"
)

func WriteDiagnostic(r *diagnose.DiagnosticReport, home string) (string, error) {
	return WriteDiagnosticForUser(r, system.UserInfo{Home: home})
}

func WriteDiagnosticForUser(r *diagnose.DiagnosticReport, u system.UserInfo) (string, error) {
	home := u.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	dir := system.UserReportDir(home)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	id := strings.ReplaceAll(strings.ReplaceAll(r.CreatedAt, ":", ""), "-", "")
	id = strings.ReplaceAll(id, "+", "-")
	id = strings.ReplaceAll(id, "T", "-")
	path := filepath.Join(dir, "diagnostic-"+id+".json")
	r.ReportPath = path
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	latest := filepath.Join(dir, "latest.json")
	_ = os.WriteFile(latest, data, 0644)
	warningCount := len(r.Warnings)
	chownDiagnosticOutputs(r, u, dir, path, latest)
	if len(r.Warnings) != warningCount {
		if updated, err := json.MarshalIndent(r, "", "  "); err == nil {
			_ = os.WriteFile(path, updated, 0644)
			_ = os.WriteFile(latest, updated, 0644)
			chownDiagnosticOutputs(r, u, path, latest)
		}
	}
	return path, nil
}

func chownDiagnosticOutputs(r *diagnose.DiagnosticReport, u system.UserInfo, paths ...string) {
	if os.Geteuid() != 0 || u.UID == "" || u.GID == "" {
		return
	}
	uid, uidErr := strconv.Atoi(u.UID)
	gid, gidErr := strconv.Atoi(u.GID)
	if uidErr != nil || gidErr != nil {
		r.Warnings = append(r.Warnings, "could not chown report files: invalid uid/gid for real user")
		return
	}
	for _, path := range paths {
		if err := os.Chown(path, uid, gid); err != nil {
			r.Warnings = append(r.Warnings, "could not chown report path "+path+": "+err.Error())
		}
	}
}

func LatestDiagnosticPath(home string) (string, error) {
	dir := system.UserReportDir(home)
	latest := filepath.Join(dir, "latest.json")
	if _, err := os.Stat(latest); err == nil {
		return latest, nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, "diagnostic-*.json"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", os.ErrNotExist
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func LoadDiagnostic(path string) (diagnose.DiagnosticReport, error) {
	var r diagnose.DiagnosticReport
	data, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(data, &r)
	return r, err
}

func HumanDiagnostic(r diagnose.DiagnosticReport) string {
	var b strings.Builder
	status := "needs attention"
	if len(r.Classifications) == 1 && r.Classifications[0] == "OK" {
		status = "OK"
	}
	fmt.Fprintf(&b, "Cactus AgentLink Rescue %s\n\n", r.ToolVersion)
	fmt.Fprintf(&b, "Current status: %s\n", status)
	if len(r.Classifications) > 0 {
		fmt.Fprintf(&b, "Failure classes: %s\n", strings.Join(r.Classifications, ", "))
	}
	if r.RecommendedRepairLevel != "" {
		fmt.Fprintf(&b, "Recommended repair level: %s\n", r.RecommendedRepairLevel)
	}
	fmt.Fprintf(&b, "System proxy: %s\n", cleanDirty(r.Network.ProxySummary.Dirty))
	fmt.Fprintf(&b, "Default route: %s\n", okMissing(r.Network.DefaultRoute.Present))
	fmt.Fprintf(&b, "DNS: %s\n", probeMapSummary(r.Reachability.DNSNames))
	fmt.Fprintf(&b, "HTTPS: %s\n", probeMapSummary(r.Reachability.HTTPSTargets))
	fmt.Fprintf(&b, "Agent endpoints: %s\n", probeMapSummary(r.Reachability.AgentTargets))
	if countResidues(r.Residues) > 0 {
		fmt.Fprintf(&b, "Known residue: %d item(s) found\n", countResidues(r.Residues))
	}
	if r.ReportPath != "" {
		fmt.Fprintf(&b, "\nReport path: %s\n", r.ReportPath)
	}
	return b.String()
}

func cleanDirty(dirty bool) string {
	if dirty {
		return "dirty"
	}
	return "clean"
}

func okMissing(ok bool) string {
	if ok {
		return "OK"
	}
	return "missing"
}

func probeMapSummary(m map[string]diagnose.ProbeResult) string {
	if len(m) == 0 {
		return "not checked"
	}
	ok := 0
	for _, p := range m {
		if p.OK {
			ok++
		}
	}
	if ok == len(m) {
		return "OK"
	}
	if ok == 0 {
		return "failed"
	}
	return fmt.Sprintf("partial (%d/%d)", ok, len(m))
}

func countResidues(r diagnose.ResidueInfo) int {
	return len(r.LaunchDaemons) + len(r.LaunchAgents) + len(r.PrivilegedHelpers) + len(r.SystemExtensions) + len(r.GroupContainers) + len(r.AppSupport) + len(r.Caches) + len(r.Preferences) + len(r.Profiles)
}
