package restartgate

import (
	"context"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/networkverify"
	"cactus-agentlink-rescue/internal/system"
)

type Report struct {
	SchemaVersion       int                  `json:"schemaVersion"`
	ToolVersion         string               `json:"toolVersion"`
	CreatedAt           string               `json:"createdAt"`
	RestartRequired     bool                 `json:"restartRequired"`
	Reason              string               `json:"reason,omitempty"`
	CompletedActions    []string             `json:"completedActions,omitempty"`
	StillFailingSignals []string             `json:"stillFailingSignals,omitempty"`
	DoBeforeRestart     []string             `json:"doBeforeRestart,omitempty"`
	PostRestartCommand  string               `json:"postRestartCommand,omitempty"`
	SupportBundlePath   string               `json:"supportBundlePath,omitempty"`
	RollbackAvailable   bool                 `json:"rollbackAvailable"`
	Verify              networkverify.Report `json:"verify"`
}

func Prepare(ctx context.Context, runner command.Runner, rulesDir string) Report {
	verify := networkverify.Run(ctx, runner, rulesDir, false)
	report := baseReport(verify)
	if verify.OK {
		report.RestartRequired = false
		report.Reason = "network verification already passes"
		return report
	}
	if verify.Tun.RecommendedRepair == "tun" || hasCriticalNetworkFailures(verify) {
		report.RestartRequired = true
		report.Reason = "macOS NetworkExtension/TUN runtime state may still be held after targeted repair"
		report.CompletedActions = []string{
			"Clash/Mihomo runtime stop attempted",
			"NetworkExtension daemons kickstart attempted",
			"stale utun down attempted",
			"Wi-Fi DHCP/DNS/default route rebuild attempted",
			"AWDL/sharingd refresh attempted",
		}
		report.DoBeforeRestart = []string{"Do not reopen Clash, ClashX, Clash Verge, or Mihomo before verification."}
		report.PostRestartCommand = "agentlink restart-gate verify --json"
	}
	return report
}

func Verify(ctx context.Context, runner command.Runner, rulesDir string) Report {
	verify := networkverify.Run(ctx, runner, rulesDir, false)
	report := baseReport(verify)
	report.RestartRequired = !verify.OK
	if verify.OK {
		report.Reason = "post-restart network verification passed"
	} else {
		report.Reason = "post-restart network verification still fails"
	}
	report.PostRestartCommand = "agentlink verify network --json"
	return report
}

func baseReport(verify networkverify.Report) Report {
	report := Report{
		SchemaVersion:     1,
		ToolVersion:       system.Version,
		CreatedAt:         time.Now().Format(time.RFC3339),
		Verify:            verify,
		RollbackAvailable: false,
	}
	for _, class := range verify.FailureClasses {
		if class != "OK" {
			report.StillFailingSignals = append(report.StillFailingSignals, class)
		}
	}
	return report
}

func hasCriticalNetworkFailures(v networkverify.Report) bool {
	return !v.DefaultRouteOK || !v.RawIPOK || !v.DNSOK || !v.HTTPSOK
}
