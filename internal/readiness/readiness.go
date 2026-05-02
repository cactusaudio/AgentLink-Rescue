package readiness

import (
	"context"
	"time"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/devdoctor"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/installer"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion int                     `json:"schemaVersion"`
	ToolVersion   string                  `json:"toolVersion"`
	CreatedAt     string                  `json:"createdAt"`
	Status        string                  `json:"status"`
	Checks        []Check                 `json:"checks"`
	Facts         facts.Facts             `json:"facts"`
	Brain         brain.BrainAvailability `json:"brain"`
	Installers    installer.DoctorReport  `json:"installers"`
	DevEssentials devdoctor.Report        `json:"devEssentials"`
	Warnings      []string                `json:"warnings,omitempty"`
	NextActions   []string                `json:"nextActions,omitempty"`
}

type Check struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
	Action   string `json:"action,omitempty"`
}

func Run(ctx context.Context, runner command.Runner, home string, catalog installer.Catalog) Report {
	if runner == nil {
		runner = command.NewExecRunner()
	}
	f := facts.Collect(ctx, runner, home, true)
	b := brain.Doctor(ctx, runner, home)
	inst := installer.Doctor(ctx, runner, catalog, system.Version)
	dev := devdoctor.Run(ctx, runner, system.Version)
	report := Report{
		SchemaVersion: 1,
		ToolVersion:   system.Version,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Status:        "ready",
		Facts:         f,
		Brain:         b,
		Installers:    inst,
		DevEssentials: dev,
	}
	report.addNetworkCheck(f)
	report.addProxyCheck(f)
	report.addBrainCheck(b)
	report.addInstallerCheck(inst)
	report.addSecretsCheck(f)
	report.addDevCheck(dev)
	report.finalize()
	return report
}

func (r *Report) addNetworkCheck(f facts.Facts) {
	status := "ok"
	evidence := "network classifications: OK"
	if len(f.LikelyFailures) > 0 && !(len(f.LikelyFailures) == 1 && f.LikelyFailures[0] == "OK") {
		status = "warn"
		evidence = "likely failures: " + join(f.LikelyFailures)
	}
	r.Checks = append(r.Checks, Check{ID: "network", Title: "Network / endpoint path", Status: status, Evidence: evidence, Action: "Run guided rescue or network diagnose before escalating."})
}

func (r *Report) addProxyCheck(f facts.Facts) {
	status := "ok"
	evidence := "no proxy env vars detected"
	if len(f.ProxyEnv) > 0 {
		status = "warn"
		evidence = "proxy env vars present"
	}
	r.Checks = append(r.Checks, Check{ID: "proxy", Title: "Proxy environment", Status: status, Evidence: evidence, Action: "Review proxy detect or use proxy cleanup recipe if stale."})
}

func (r *Report) addBrainCheck(b brain.BrainAvailability) {
	status := "ok"
	evidence := "Gemma Brain assets available"
	action := "Brain planner can run offline."
	if !b.BrainPackAvailable || !b.ModelExists || !b.RuntimeExecutable || !b.ModelSHA256OK {
		status = "warn"
		evidence = "Brain model/runtime missing or not verified"
		action = "Use Brain package offline or run scripts/fetch_brain_assets.sh while online."
	}
	r.Checks = append(r.Checks, Check{ID: "brain", Title: "Local Brain readiness", Status: status, Evidence: evidence, Action: action})
}

func (r *Report) addInstallerCheck(inst installer.DoctorReport) {
	missing := 0
	ready := 0
	for _, item := range inst.Reports {
		switch item.Status {
		case "installed", "installed_verified", "available":
			ready++
		default:
			missing++
		}
	}
	status := "ok"
	if missing > 0 {
		status = "warn"
	}
	r.Checks = append(r.Checks, Check{ID: "installers", Title: "Recovery installers", Status: status, Evidence: joinCounts(ready, missing), Action: "Open Installer Center for missing tools."})
}

func (r *Report) addSecretsCheck(f facts.Facts) {
	present := 0
	for _, key := range f.APIKeys {
		if key.Present {
			present++
		}
	}
	status := "ok"
	evidence := "API key env presence detected"
	if present == 0 {
		status = "warn"
		evidence = "no common AI API key env vars present"
	}
	r.Checks = append(r.Checks, Check{ID: "secrets", Title: "API key readiness", Status: status, Evidence: evidence, Action: "Set only needed vendor keys; AgentLink reports redacted presence only."})
}

func (r *Report) addDevCheck(dev devdoctor.Report) {
	status := "ok"
	if dev.Status != "ok" {
		status = "warn"
	}
	r.Checks = append(r.Checks, Check{ID: "dev-essentials", Title: "Dev essentials for AI CLI installers", Status: status, Evidence: dev.Status, Action: "Install only the missing base tool needed by an AI CLI installer."})
}

func (r *Report) finalize() {
	for _, c := range r.Checks {
		if c.Status == "fail" {
			r.Status = "not_ready"
			r.NextActions = append(r.NextActions, c.Action)
		}
		if c.Status == "warn" && r.Status == "ready" {
			r.Status = "ready_with_warnings"
			r.NextActions = append(r.NextActions, c.Action)
		}
	}
	if r.Status == "ready" {
		r.NextActions = append(r.NextActions, "No rescue action required. Export a support bundle before risky changes.")
	}
}

func join(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	out := values[0]
	for _, v := range values[1:] {
		out += ", " + v
	}
	return out
}

func joinCounts(ready, missing int) string {
	return "ready/available: " + itoa(ready) + ", attention: " + itoa(missing)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	return string(digits[i:])
}
