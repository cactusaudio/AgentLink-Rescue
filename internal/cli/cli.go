package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/brain"
	"cactus-agentlink-rescue/internal/chaos"
	"cactus-agentlink-rescue/internal/classify"
	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/devdoctor"
	"cactus-agentlink-rescue/internal/diagnose"
	"cactus-agentlink-rescue/internal/facts"
	"cactus-agentlink-rescue/internal/field"
	"cactus-agentlink-rescue/internal/guided"
	"cactus-agentlink-rescue/internal/installer"
	"cactus-agentlink-rescue/internal/journal"
	"cactus-agentlink-rescue/internal/lastgood"
	"cactus-agentlink-rescue/internal/networkverify"
	"cactus-agentlink-rescue/internal/opencode"
	"cactus-agentlink-rescue/internal/orchestrator"
	"cactus-agentlink-rescue/internal/packagehealth"
	"cactus-agentlink-rescue/internal/planner"
	"cactus-agentlink-rescue/internal/proxyapp"
	"cactus-agentlink-rescue/internal/readiness"
	"cactus-agentlink-rescue/internal/recipe"
	"cactus-agentlink-rescue/internal/repair"
	"cactus-agentlink-rescue/internal/report"
	"cactus-agentlink-rescue/internal/restartgate"
	"cactus-agentlink-rescue/internal/rollback"
	"cactus-agentlink-rescue/internal/session"
	"cactus-agentlink-rescue/internal/snapshot"
	"cactus-agentlink-rescue/internal/supportbundle"
	"cactus-agentlink-rescue/internal/system"
	"cactus-agentlink-rescue/internal/ticket"
	"cactus-agentlink-rescue/internal/verifier"
	"cactus-agentlink-rescue/internal/verify"
)

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 50
	}
	safeMode := false
	if args[0] == "--safe-mode" {
		safeMode = true
		args = args[1:]
		if len(args) == 0 {
			fmt.Fprintln(stderr, "--safe-mode requires a command")
			return 50
		}
	}
	cmd := args[0]
	if safeMode && !safeModeAllowed(cmd, args[1:]) {
		fmt.Fprintf(stderr, "safe mode allows only doctor, support bundle, package doctor, and journal list/recover; refused: %s\n", cmd)
		return 50
	}
	if cmd != "version" && cmd != "selftest" && !system.IsDarwin() {
		fmt.Fprintln(stderr, "unsupported platform: agentlink supports macOS only")
		return 60
	}
	ctx := context.Background()
	runner := command.NewExecRunner()
	rulesDir := findRulesDir()
	switch cmd {
	case "readiness":
		return runReadiness(ctx, runner, args[1:], stdout, stderr)
	case "last-good":
		return runLastGood(ctx, runner, args[1:], stdout, stderr)
	case "support":
		return runSupport(ctx, runner, args[1:], stdout, stderr)
	case "package":
		return runPackage(ctx, runner, args[1:], stdout, stderr)
	case "journal":
		return runJournal(ctx, runner, args[1:], stdout, stderr)
	case "chaos":
		return runChaos(args[1:], stdout, stderr)
	case "dev":
		return runDev(ctx, runner, args[1:], stdout, stderr)
	case "guided":
		return runGuided(ctx, runner, args[1:], stdout, stderr)
	case "orchestrator":
		return runOrchestrator(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "field":
		return runField(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "verify":
		return runVerify(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "restart-gate":
		return runRestartGate(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "ticket":
		return runTicket(ctx, runner, args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(ctx, runner, args[1:], stdout, stderr)
	case "snapshot":
		return runSnapshot(ctx, runner, args[1:], stdout, stderr)
	case "diff":
		return runDiff(ctx, runner, args[1:], stdout, stderr)
	case "restore":
		return runRestore(ctx, runner, args[1:], stdout, stderr)
	case "recipe":
		return runRecipe(ctx, runner, args[1:], stdout, stderr)
	case "repair":
		return runTargetRepair(ctx, runner, args[1:], stdout, stderr)
	case "config":
		return runConfig(ctx, runner, args[1:], stdout, stderr)
	case "proxy":
		return runProxy(ctx, runner, args[1:], stdout, stderr)
	case "keys":
		return runKeys(ctx, runner, args[1:], stdout, stderr)
	case "planner":
		return runPlanner(ctx, runner, args[1:], stdout, stderr)
	case "brain":
		return runBrain(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "installer":
		return runInstaller(ctx, runner, args[1:], stdout, stderr)
	case "opencode":
		return runOpenCode(ctx, runner, args[1:], stdout, stderr)
	case "proxyapp":
		return runProxyApp(ctx, runner, args[1:], stdout, stderr)
	case "diagnose":
		return runDiagnose(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "classify":
		return runClassify(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "rescue":
		return runRescue(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "rollback":
		return runRollback(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "report":
		return runReport(ctx, runner, args[1:], stdout, stderr)
	case "selftest":
		return runSelftest(stdout, stderr)
	case "manifest":
		return runManifest(args[1:], stdout, stderr)
	case "diagnose-graph":
		return runDiagnoseGraph(ctx, runner, rulesDir, args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "agentlink %s\n", system.Version)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		usage(stderr)
		return 50
	}
}

func safeModeAllowed(cmd string, args []string) bool {
	switch cmd {
	case "version", "doctor", "selftest":
		return true
	case "support":
		return len(args) > 0 && args[0] == "bundle"
	case "package":
		return len(args) > 0 && args[0] == "doctor"
	case "journal":
		return len(args) > 0 && (args[0] == "list" || args[0] == "recover" || args[0] == "inspect")
	default:
		return false
	}
}

func runDoctor(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	home := currentHome(ctx, runner)
	f := facts.Collect(ctx, runner, home, true)
	if *jsonOut {
		data, _ := json.MarshalIndent(f, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s doctor\n\n", system.Version)
	fmt.Fprintf(stdout, "OS: %s/%s\nShell: %s\n", f.OS, f.Arch, f.Shell)
	fmt.Fprintf(stdout, "Likely failures: %s\n", strings.Join(f.LikelyFailures, ", "))
	fmt.Fprintf(stdout, "Codex config: %s\n", existsText(f.ConfigPaths["codex"].Exists))
	fmt.Fprintf(stdout, "Proxy env vars: %d\n", len(f.ProxyEnv))
	return 0
}

func runReadiness(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "doctor" {
		fmt.Fprintln(stderr, "readiness requires doctor")
		return 50
	}
	fs := flag.NewFlagSet("readiness doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	catalog, err := installer.LoadCatalog(installer.FindCatalogPath())
	if err != nil {
		catalog = installer.Catalog{SchemaVersion: 1}
		fmt.Fprintln(stderr, "installer catalog unavailable; readiness will continue in degraded mode:", err)
	}
	rep := readiness.Run(ctx, runner, currentHome(ctx, runner), catalog)
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s Offline Readiness\n\n", system.Version)
	fmt.Fprintf(stdout, "Status: %s\n", rep.Status)
	for _, c := range rep.Checks {
		fmt.Fprintf(stdout, "- %s: %s (%s)\n", c.Title, c.Status, c.Evidence)
	}
	if len(rep.NextActions) > 0 {
		fmt.Fprintln(stdout, "\nNext actions:")
		for _, a := range rep.NextActions {
			fmt.Fprintf(stdout, "- %s\n", a)
		}
	}
	return 0
}

func runDev(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "doctor" {
		fmt.Fprintln(stderr, "dev requires doctor")
		return 50
	}
	fs := flag.NewFlagSet("dev doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	rep := devdoctor.Run(ctx, runner, system.Version)
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s Dev Essentials Doctor\n\n", system.Version)
	fmt.Fprintf(stdout, "Status: %s\n", rep.Status)
	for _, tool := range rep.Tools {
		fmt.Fprintf(stdout, "- %s: %s\n", tool.Name, tool.Status)
	}
	return 0
}

func runLastGood(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "last-good requires save, list, inspect, or restore")
		return 50
	}
	home := currentHome(ctx, runner)
	switch args[0] {
	case "save":
		fs := flag.NewFlagSet("last-good save", flag.ContinueOnError)
		fs.SetOutput(stderr)
		name := fs.String("name", "", "profile name")
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := lastgood.Save(ctx, runner, lastgood.Options{Home: home, Version: system.Version, Name: *name})
		return printLastGood(rep, *jsonOut, stdout, stderr)
	case "list":
		fs := flag.NewFlagSet("last-good list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := lastgood.List(home, system.Version)
		return printLastGood(rep, *jsonOut, stdout, stderr)
	case "inspect":
		fs := flag.NewFlagSet("last-good inspect", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		parseArgs := args[1:]
		id := ""
		if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
			id = parseArgs[0]
			parseArgs = parseArgs[1:]
		}
		if err := fs.Parse(parseArgs); err != nil {
			return 50
		}
		if id == "" && fs.NArg() == 1 {
			id = fs.Arg(0)
		}
		if id == "" {
			fmt.Fprintln(stderr, "last-good inspect requires id")
			return 50
		}
		rep := lastgood.Inspect(home, id, system.Version)
		return printLastGood(rep, *jsonOut, stdout, stderr)
	case "restore":
		fs := flag.NewFlagSet("last-good restore", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		yes := fs.Bool("yes", false, "restore saved config files")
		last := fs.Bool("last", false, "restore latest profile")
		idFlag := fs.String("id", "", "profile id")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := lastgood.Restore(ctx, runner, lastgood.Options{Home: home, Version: system.Version, ID: *idFlag, Last: *last, Yes: *yes})
		return printLastGood(rep, *jsonOut, stdout, stderr)
	default:
		fmt.Fprintln(stderr, "unknown last-good subcommand")
		return 50
	}
}

func printLastGood(rep lastgood.Report, jsonOut bool, stdout, stderr io.Writer) int {
	if jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Last-good %s: %s\n", rep.Action, rep.Status)
		if rep.ProfileID != "" {
			fmt.Fprintf(stdout, "Profile: %s\n", rep.ProfileID)
		}
		if rep.ProfilePath != "" {
			fmt.Fprintf(stdout, "Path: %s\n", rep.ProfilePath)
		}
		for _, item := range rep.SavedItems {
			if item.OriginalPath != "" {
				fmt.Fprintf(stdout, "- %s %s\n", item.ID, item.OriginalPath)
			}
		}
		if rep.SnapshotID != "" {
			fmt.Fprintf(stdout, "Rollback: agentlink restore last\n")
		}
		if rep.NextAction != "" {
			fmt.Fprintf(stdout, "Next: %s\n", rep.NextAction)
		}
	}
	if rep.Status == "failed" {
		for _, w := range rep.Warnings {
			fmt.Fprintln(stderr, w)
		}
		return 30
	}
	return 0
}

func runSupport(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "bundle" {
		fmt.Fprintln(stderr, "support requires bundle")
		return 50
	}
	fs := flag.NewFlagSet("support bundle", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	output := fs.String("output", "", "bundle zip path")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	catalog, err := installer.LoadCatalog(installer.FindCatalogPath())
	if err != nil {
		catalog = installer.Catalog{SchemaVersion: 1}
		fmt.Fprintln(stderr, "installer catalog unavailable; support bundle will continue in degraded mode:", err)
	}
	rep := supportbundle.Create(ctx, runner, currentHome(ctx, runner), *output, catalog)
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Support bundle: %s\nStatus: %s\n", rep.BundlePath, rep.Status)
		for _, f := range rep.Files {
			fmt.Fprintf(stdout, "- %s\n", f)
		}
	}
	if rep.Status == "failed" {
		return 30
	}
	return 0
}

func runPackage(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "package requires doctor or repair")
		return 50
	}
	fs := flag.NewFlagSet("package "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	packageRoot := fs.String("package-root", "", "explicit AgentLink package root")
	dryRun := fs.Bool("dry-run", false, "show package repair actions without changing files")
	yes := fs.Bool("yes", false, "apply package-local repairs")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	opts := packagehealth.Options{PackageRoot: *packageRoot, DryRun: *dryRun, Yes: *yes}
	var rep packagehealth.Report
	switch args[0] {
	case "doctor":
		rep = packagehealth.Doctor(ctx, runner, opts)
	case "repair":
		if !*dryRun && !*yes {
			*dryRun = true
			opts.DryRun = true
		}
		rep = packagehealth.Repair(ctx, runner, opts)
	default:
		fmt.Fprintln(stderr, "package requires doctor or repair")
		return 50
	}
	if *jsonOut {
		fmt.Fprintln(stdout, packagehealth.Marshal(rep))
	} else {
		fmt.Fprintf(stdout, "Package %s: %s\nRoot: %s\n", args[0], rep.Status, rep.PackageRoot)
		for _, check := range rep.Checks {
			line := fmt.Sprintf("- %s: %s", check.ID, check.Status)
			if check.Path != "" {
				line += " " + check.Path
			}
			if check.Message != "" {
				line += " (" + check.Message + ")"
			}
			fmt.Fprintln(stdout, line)
		}
		if len(rep.Actions) > 0 {
			fmt.Fprintln(stdout, "\nActions:")
			for _, action := range rep.Actions {
				fmt.Fprintf(stdout, "- %s\n", action)
			}
		}
	}
	switch rep.Status {
	case "ok", "warning", "dry_run", "repaired":
		return 0
	case "needs_repair", "partial_repair":
		return 20
	default:
		if rep.Error != "" {
			fmt.Fprintln(stderr, rep.Error)
		}
		return 30
	}
}

func runJournal(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "journal requires list, inspect, or recover")
		return 50
	}
	home := currentHome(ctx, runner)
	mgr := journal.New(home)
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("journal list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		list, err := mgr.List()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(list, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			for _, tx := range list {
				fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", tx.TransactionID, tx.State, tx.Target, tx.RestorePoint)
			}
		}
		return 0
	case "inspect":
		fs := flag.NewFlagSet("journal inspect", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		parseArgs := args[1:]
		id := ""
		if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
			id = parseArgs[0]
			parseArgs = parseArgs[1:]
		}
		if err := fs.Parse(parseArgs); err != nil {
			return 50
		}
		if id == "" && fs.NArg() == 1 {
			id = fs.Arg(0)
		}
		if id == "" {
			fmt.Fprintln(stderr, "journal inspect requires id")
			return 50
		}
		tx, err := mgr.Load(id)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(tx, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Transaction: %s\nState: %s\nTarget: %s\nRestore point: %s\n", tx.TransactionID, tx.State, tx.Target, tx.RestorePoint)
		}
		return 0
	case "recover":
		fs := flag.NewFlagSet("journal recover", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		yes := fs.Bool("yes", false, "mark no-mutation transactions abandoned and create rollback tickets for mutated transactions")
		packageRoot := fs.String("package-root", "", "explicit AgentLink package root for generated tickets")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := mgr.Recover(!*yes)
		if *yes {
			root := *packageRoot
			if root == "" {
				root = packagehealth.FindPackageRoot()
			}
			for _, tx := range rep.Transactions {
				if mgr.HasMutations(tx.TransactionID) && tx.RollbackAvailable && tx.RestorePoint != "" {
					t := ticket.Create(ctx, runner, ticket.Options{Home: home, Type: "rollback", ID: tx.RestorePoint, Version: system.Version, PackageRoot: root})
					if t.Status == "created" {
						rep.Warnings = append(rep.Warnings, "rollback terminal ticket created: "+t.Directory)
						rep.NextAction = "run rollback terminal ticket: " + t.Directory
					} else {
						rep.Status = "failed"
						rep.Warnings = append(rep.Warnings, "rollback ticket failed: "+t.Error)
					}
				}
			}
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Journal recover: %s\n", rep.Status)
			for _, tx := range rep.Transactions {
				fmt.Fprintf(stdout, "- %s %s %s\n", tx.TransactionID, tx.State, tx.RestorePoint)
			}
			for _, warning := range rep.Warnings {
				fmt.Fprintf(stdout, "Warning: %s\n", warning)
			}
			if rep.NextAction != "" {
				fmt.Fprintf(stdout, "Next: %s\n", rep.NextAction)
			}
		}
		if rep.Status == "failed" {
			return 30
		}
		if rep.Status == "incomplete_transactions_found" {
			return 20
		}
		return 0
	default:
		fmt.Fprintln(stderr, "journal requires list, inspect, or recover")
		return 50
	}
}

func runChaos(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "chaos requires list or run")
		return 50
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("chaos list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		root := fs.String("root", filepath.Join("testdata", "chaos"), "fixture root")
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		list, err := chaos.List(*root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(list, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			for _, path := range list {
				fmt.Fprintln(stdout, path)
			}
		}
		return 0
	case "run":
		fs := flag.NewFlagSet("chaos run", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fixture := fs.String("fixture", "", "fixture JSON path")
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		if *fixture == "" {
			fmt.Fprintln(stderr, "chaos run requires --fixture")
			return 50
		}
		rep := chaos.Run(*fixture)
		if *jsonOut {
			data, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Chaos %s: %s\n", rep.ID, rep.Status)
			for _, warning := range rep.Warnings {
				fmt.Fprintf(stdout, "- %s\n", warning)
			}
		}
		if rep.Status != "passed" {
			return 30
		}
		return 0
	default:
		fmt.Fprintln(stderr, "chaos requires list or run")
		return 50
	}
}

func runSnapshot(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	_ = args
	home := currentHome(ctx, runner)
	rp, err := snapshot.NewRestorePointWithPolicy(system.UserRestorePointsDir(home), system.Version, system.MutationOptions{RealUserHome: home})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	fmt.Fprintf(stdout, "Created snapshot: %s\n", rp.Path)
	return 0
}

func runDiff(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("snapshot", "", "snapshot id")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	home := currentHome(ctx, runner)
	rp, err := loadUserSnapshot(home, *id)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	fmt.Fprintf(stdout, "Snapshot: %s\n", rp.Manifest.ID)
	for _, entry := range rp.Manifest.Entries {
		fmt.Fprintf(stdout, "- %s %s\n", entry.Action, entry.OriginalPath)
	}
	return 0
}

func runRestore(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "last" {
		fmt.Fprintln(stderr, "restore requires: restore last")
		return 50
	}
	fs := flag.NewFlagSet("restore last", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	home := currentHome(ctx, runner)
	rp, err := snapshot.Latest(system.UserRestorePointsDir(home))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	rp.SetMutationPolicy(system.MutationOptions{RealUserHome: home, IncludeNetworkExtensionPlists: true, ExtraAllowedPaths: manifestPaths(rp.Manifest)})
	if err := rp.RestoreAllWithRunner(ctx, runner); err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if *jsonOut {
		out := map[string]any{"status": "restored", "snapshotID": rp.Manifest.ID, "snapshotPath": rp.Path}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Restored snapshot: %s\n", rp.Manifest.ID)
	return 0
}

func runRecipe(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "recipe requires list, inspect, or run")
		return 50
	}
	reg, err := recipe.LoadRegistry(recipe.FindRecipesDir())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("recipe list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		list := reg.List()
		if *jsonOut {
			data, _ := json.MarshalIndent(list, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			for _, r := range list {
				fmt.Fprintf(stdout, "%s\t%s\t%s\n", r.ID, r.Risk, r.Title)
			}
		}
		return 0
	case "inspect":
		fs := flag.NewFlagSet("recipe inspect", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		parseArgs := args[1:]
		id := ""
		if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
			id = parseArgs[0]
			parseArgs = parseArgs[1:]
		}
		if err := fs.Parse(parseArgs); err != nil {
			return 50
		}
		if id == "" && fs.NArg() == 1 {
			id = fs.Arg(0)
		}
		if id == "" {
			fmt.Fprintln(stderr, "recipe inspect requires id")
			return 50
		}
		r, ok := reg.Get(id)
		if !ok {
			fmt.Fprintln(stderr, "unknown recipe")
			return 30
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(r, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "%s\n%s\nRisk: %s\n", r.Title, r.Description, r.Risk)
		}
		return 0
	case "run":
		fs := flag.NewFlagSet("recipe run", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dryRun := fs.Bool("dry-run", false, "dry run")
		yes := fs.Bool("yes", false, "approve")
		jsonOut := fs.Bool("json", false, "JSON")
		params := paramFlags{}
		fs.Var(&params, "param", "key=value param")
		parseArgs := args[1:]
		id := ""
		if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
			id = parseArgs[0]
			parseArgs = parseArgs[1:]
		}
		if err := fs.Parse(parseArgs); err != nil {
			return 50
		}
		if id == "" && fs.NArg() == 1 {
			id = fs.Arg(0)
		}
		if id == "" {
			fmt.Fprintln(stderr, "recipe run requires id")
			return 50
		}
		return executeRecipe(ctx, runner, reg, id, recipe.RunOptions{Home: currentHome(ctx, runner), DryRun: *dryRun, Yes: *yes, JSON: *jsonOut, Params: params.values, CommandLine: append([]string{"recipe", "run", id}, args[1:]...)}, *jsonOut, stdout, stderr)
	default:
		fmt.Fprintln(stderr, "unknown recipe subcommand")
		return 50
	}
}

func runTargetRepair(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("repair", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "path, proxy, codex, keys, or network")
	dryRun := fs.Bool("dry-run", false, "dry run")
	yes := fs.Bool("yes", false, "approve")
	jsonOut := fs.Bool("json", false, "JSON")
	auto := fs.Bool("auto", false, "auto-select repair")
	brainFlag := fs.Bool("brain", false, "use local brain planner")
	online := fs.Bool("online", false, "allow network_action recipes")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *auto || *brainFlag {
		if !*auto || !*brainFlag {
			fmt.Fprintln(stderr, "brain auto repair requires --auto --brain")
			return 50
		}
		if *target == "" {
			fmt.Fprintln(stderr, "repair --auto --brain requires --target")
			return 50
		}
		reg, err := recipe.LoadRegistry(recipe.FindRecipesDir())
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		res := brain.RunLoop(ctx, runner, reg, brain.LoopOptions{Home: currentHome(ctx, runner), Target: *target, DryRun: *dryRun, Yes: *yes, Online: *online, JSON: *jsonOut, CommandLine: append([]string{"repair"}, args...)})
		if *jsonOut {
			fmt.Fprintln(stdout, brain.MarshalLoopResult(res))
		} else {
			fmt.Fprintf(stdout, "Brain auto repair\nTarget: %s\nStatus: %s\n", res.Target, res.Status)
			if res.StopReason != "" {
				fmt.Fprintf(stdout, "Stop reason: %s\n", res.StopReason)
			}
			if res.RecipeResult.RecipeID != "" {
				fmt.Fprintf(stdout, "Recipe: %s\n", res.RecipeResult.RecipeID)
			}
			if res.RecipeResult.SnapshotID != "" {
				fmt.Fprintln(stdout, "Rollback: agentlink restore last")
			}
			if res.HumanReportPath != "" {
				fmt.Fprintf(stdout, "Report path: %s\n", res.HumanReportPath)
			}
		}
		if res.Error != "" {
			fmt.Fprintln(stderr, res.Error)
			return 30
		}
		return 0
	}
	id := map[string]string{"path": "macos-zsh-path-repair", "proxy": "proxy-clean-stale-env", "codex": "codex-config-parse-repair", "keys": "api-key-detection-redaction"}[*target]
	if id == "" {
		fmt.Fprintln(stderr, "repair --target must be path, proxy, codex, or keys")
		return 50
	}
	reg, err := recipe.LoadRegistry(recipe.FindRecipesDir())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	return executeRecipe(ctx, runner, reg, id, recipe.RunOptions{Home: currentHome(ctx, runner), DryRun: *dryRun, Yes: *yes, JSON: *jsonOut, CommandLine: append([]string{"repair"}, args...)}, *jsonOut, stdout, stderr)
}

func runGuided(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "rescue" {
		fmt.Fprintln(stderr, "guided requires: guided rescue")
		return 50
	}
	fs := flag.NewFlagSet("guided rescue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "auto", "auto, path, proxy, codex, keys, or network")
	dryRun := fs.Bool("dry-run", false, "analyze and dry-run only")
	yes := fs.Bool("yes", false, "allow reversible recipe execution")
	jsonOut := fs.Bool("json", false, "print JSON")
	maxCycles := fs.Int("max-cycles", 3, "maximum guided cycles")
	timeoutSeconds := fs.Int("timeout-seconds", 600, "maximum runtime in seconds")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	if !validGuidedTarget(*target) {
		fmt.Fprintln(stderr, "guided rescue --target must be auto, path, proxy, codex, keys, or network")
		return 50
	}
	if *yes && *dryRun {
		fmt.Fprintln(stderr, "guided rescue cannot combine --yes and --dry-run")
		return 50
	}
	effectiveDryRun := *dryRun || !*yes
	reg, err := recipe.LoadRegistry(recipe.FindRecipesDir())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	rep := guided.Run(ctx, runner, reg, guided.Options{
		Home:           currentHome(ctx, runner),
		Target:         *target,
		DryRun:         effectiveDryRun,
		Yes:            *yes,
		MaxCycles:      *maxCycles,
		TimeoutSeconds: *timeoutSeconds,
		CommandLine:    append([]string{"guided", "rescue"}, args[1:]...),
	})
	if *jsonOut {
		fmt.Fprintln(stdout, guided.MarshalReport(rep))
	} else {
		fmt.Fprint(stdout, guided.HumanReport(rep))
	}
	if rep.Status == guided.StatusFailed {
		return 30
	}
	return 0
}

func runOrchestrator(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "rescue" {
		fmt.Fprintln(stderr, "orchestrator requires: orchestrator rescue")
		return 50
	}
	fs := flag.NewFlagSet("orchestrator rescue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "auto", "auto, network, clash-tun, proxy, codex, keys, readiness")
	dryRun := fs.Bool("dry-run", false, "dry run")
	yes := fs.Bool("yes", false, "execute if allowed")
	jsonOut := fs.Bool("json", false, "JSON")
	maxCycles := fs.Int("max-cycles", 3, "maximum cycles")
	timeoutSeconds := fs.Int("timeout-seconds", 600, "timeout seconds")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	rep := orchestrator.Run(ctx, runner, orchestrator.Options{Target: *target, DryRun: *dryRun || !*yes, Yes: *yes, MaxCycles: *maxCycles, TimeoutSeconds: *timeoutSeconds, RulesDir: rulesDir, Home: currentHome(ctx, runner), PackageRoot: ticket.FindPackageRoot()})
	if *jsonOut {
		fmt.Fprintln(stdout, orchestrator.MarshalReport(rep))
	} else {
		fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s Orchestrator\n\nStatus: %s\n%s\n", system.Version, rep.Status, rep.HumanSummary)
		if rep.TerminalTicketPath != "" {
			fmt.Fprintf(stdout, "\nTerminal ticket: %s\n", rep.TerminalTicketPath)
		}
		if rep.NextAction != "" {
			fmt.Fprintf(stdout, "Next: %s\n", rep.NextAction)
		}
	}
	if rep.Status == "failed" {
		return 30
	}
	return 0
}

func runField(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "macbook-network-rescue" {
		fmt.Fprintln(stderr, "field requires: field macbook-network-rescue")
		return 50
	}
	fs := flag.NewFlagSet("field macbook-network-rescue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	fixTun := fs.Bool("fix-tun", false, "run targeted Clash/TUN repair")
	yes := fs.Bool("yes", false, "approve targeted repair")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	if *fixTun {
		if *yes && !system.IsRoot() {
			return sudoReexec(append([]string{"field"}, args...), stdout, stderr)
		}
		res := repair.Run(ctx, runner, repair.Options{Level: repair.LevelTun, Yes: *yes, DryRun: !*yes, JSON: *jsonOut, RulesDir: rulesDir, Stdin: os.Stdin, Stdout: stdout})
		if *jsonOut {
			fmt.Fprintln(stdout, repair.JSON(res))
		} else {
			printRescueSummary(stdout, res)
		}
		return res.ExitCode
	}
	diag := diagnose.NewEngine(runner, diagnose.Options{RulesDir: rulesDir}).Run(ctx)
	classify.Apply(&diag)
	rep := field.MacBookNetworkRescue(system.Version, diag)
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s MacBook Network Rescue\n\n", system.Version)
	fmt.Fprintf(stdout, "Recommended action: %s\n", rep.RecommendedAction)
	fmt.Fprintf(stdout, "Classes: %s\n", strings.Join(rep.Diagnosis.Classes, ", "))
	fmt.Fprintf(stdout, "System proxy dirty: %v\n", rep.Diagnosis.ProxyDirty)
	fmt.Fprintf(stdout, "Clash/Verge/Mihomo residue: %v\n", rep.Diagnosis.ClashResidueDetected)
	fmt.Fprintf(stdout, "Clash/Mihomo stale TUN: %v\n", rep.Diagnosis.ClashTunDetected)
	fmt.Fprintf(stdout, "Network Extension/TUN suspected: %v\n", rep.Diagnosis.NetworkExtensionSuspected)
	fmt.Fprintf(stdout, "Default route OK: %v\n", rep.Diagnosis.DefaultRouteOK)
	fmt.Fprintf(stdout, "DNS OK: %v\n\n", rep.Diagnosis.DNSOK)
	fmt.Fprintln(stdout, "Copyable commands:")
	for _, key := range []string{"tun", "safe", "standard", "standardSystemReset", "deep", "verifyNetwork", "ticket", "supportBundle"} {
		if cmd := rep.Commands[key]; cmd != "" {
			fmt.Fprintf(stdout, "- %s: %s\n", key, cmd)
		}
	}
	for _, warning := range rep.Warnings {
		fmt.Fprintf(stdout, "Warning: %s\n", warning)
	}
	return 0
}

func validGuidedTarget(target string) bool {
	switch target {
	case "auto", "path", "proxy", "codex", "keys", "network":
		return true
	default:
		return false
	}
}

func executeRecipe(ctx context.Context, runner command.Runner, reg recipe.Registry, id string, opts recipe.RunOptions, jsonOut bool, stdout, stderr io.Writer) int {
	res := recipe.Run(ctx, runner, reg, id, opts)
	if jsonOut {
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Recipe: %s\nStatus: %s\n", res.RecipeID, res.Status)
		if res.DryRun {
			fmt.Fprintln(stdout, "Dry run: no changes made")
		}
		for _, a := range res.PlannedActions {
			fmt.Fprintf(stdout, "- %s %s\n", a.Description, a.Path)
		}
		for _, v := range res.VerifierResults {
			fmt.Fprintf(stdout, "- verifier %s: %s\n", v.ID, v.Status)
		}
		for _, w := range res.Warnings {
			fmt.Fprintf(stdout, "- warning: %s\n", w)
		}
		if res.SnapshotID != "" {
			fmt.Fprintf(stdout, "Rollback: agentlink restore last\n")
		}
		if res.HumanReportPath != "" {
			fmt.Fprintf(stdout, "Report path: %s\n", res.HumanReportPath)
		}
	}
	if res.Error != "" {
		fmt.Fprintln(stderr, res.Error)
		return 30
	}
	if res.Status == "fail" || res.Status == recipe.StatusVerifierFailed {
		return 30
	}
	return 0
}

func runConfig(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "doctor" {
		fmt.Fprintln(stderr, "config requires doctor")
		return 50
	}
	f := facts.Collect(ctx, runner, currentHome(ctx, runner), false)
	out := map[string]any{"codexConfig": f.ConfigPaths["codex"]}
	jsonOut := contains(args[1:], "--json")
	if jsonOut {
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Codex config: %s\n", f.ConfigPaths["codex"].Path)
	}
	return 0
}

func runProxy(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "detect" {
		fmt.Fprintln(stderr, "proxy requires detect")
		return 50
	}
	f := facts.Collect(ctx, runner, currentHome(ctx, runner), false)
	out := map[string]any{"proxyEnv": f.ProxyEnv, "ports": f.Ports}
	if contains(args[1:], "--json") {
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Proxy env vars: %d\n", len(f.ProxyEnv))
		for _, p := range f.Ports {
			fmt.Fprintf(stdout, "Port %s listening: %v\n", p.Port, p.Listening)
		}
	}
	return 0
}

func runKeys(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "doctor" {
		fmt.Fprintln(stderr, "keys requires doctor")
		return 50
	}
	f := facts.Collect(ctx, runner, currentHome(ctx, runner), false)
	if contains(args[1:], "--json") {
		data, _ := json.MarshalIndent(f.APIKeys, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		for _, k := range f.APIKeys {
			fmt.Fprintf(stdout, "%s present: %v\n", k.Name, k.Present)
		}
	}
	return 0
}

func runPlanner(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	_ = ctx
	_ = runner
	if len(args) != 2 || args[0] != "validate" {
		fmt.Fprintln(stderr, "planner requires validate <decision.json>")
		return 50
	}
	reg, err := recipe.LoadRegistry(recipe.FindRecipesDir())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	decision, err := planner.LoadDecision(args[1])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if err := planner.ValidateDecision(decision, reg, verifier.NewRegistry()); err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	fmt.Fprintln(stdout, "planner decision valid")
	return 0
}

func runBrain(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "brain requires doctor, fetch, selftest, prompt, chat, plan, rescue-plan, server, or explain")
		return 50
	}
	home := currentHome(ctx, runner)
	reg, err := recipe.LoadRegistry(recipe.FindRecipesDir())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	switch args[0] {
	case "doctor":
		fs := flag.NewFlagSet("brain doctor", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		doc := brain.Doctor(ctx, runner, home)
		if *jsonOut {
			data, _ := json.MarshalIndent(doc, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Cactus AgentLink Brain %s doctor\n", system.Version)
			fmt.Fprintf(stdout, "Backend: %s\nPackage root: %s\nRuntime arch: %s\nModel exists: %v\nModel SHA256 OK: %v\nRuntime executable: %v\n", doc.Backend, doc.PackageRoot, doc.RuntimeArch, doc.ModelExists, doc.ModelSHA256OK, doc.RuntimeExecutable)
			if len(doc.MissingAssets) > 0 {
				fmt.Fprintf(stdout, "Missing assets: %s\n", strings.Join(doc.MissingAssets, ", "))
				fmt.Fprintln(stdout, "Fetch:")
				for _, c := range doc.FetchCommands {
					fmt.Fprintf(stdout, "  %s\n", c)
				}
			}
		}
		return 0
	case "fetch":
		return runBrainFetch(ctx, runner, args[1:], stdout, stderr)
	case "selftest":
		fs := flag.NewFlagSet("brain selftest", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		res := brain.Selftest(ctx, runner, home, reg)
		if *jsonOut {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else if res.OK {
			fmt.Fprintln(stdout, "agentlink brain selftest OK")
			fmt.Fprintf(stdout, "Decision: %s confidence %.2f\n", res.Decision.Intent, res.Decision.Confidence)
		} else {
			fmt.Fprintf(stderr, "brain selftest failed: %s\n", res.Error)
			return 30
		}
		if !res.OK {
			return 30
		}
		return 0
	case "prompt":
		fs := flag.NewFlagSet("brain prompt", flag.ContinueOnError)
		fs.SetOutput(stderr)
		text := fs.String("text", "", "prompt text")
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		if *text == "" {
			fmt.Fprintln(stderr, "brain prompt requires --text")
			return 50
		}
		resp, err := brain.NewLlamaCLIBackend(runner, home).Generate(ctx, brain.BrainRequest{SystemPrompt: "Return a short response.", UserPrompt: *text, MaxTokens: 256, Temperature: 0, ContextSize: 2048, Timeout: 2 * time.Minute})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprint(stdout, resp.RawText)
		}
		return 0
	case "chat":
		fs := flag.NewFlagSet("brain chat", flag.ContinueOnError)
		fs.SetOutput(stderr)
		prompt := fs.String("prompt", "", "chat prompt")
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		if *prompt == "" {
			fmt.Fprintln(stderr, "brain chat requires --prompt")
			return 50
		}
		rep := brain.Chat(ctx, runner, home, *prompt)
		if *jsonOut {
			data, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else if rep.OK {
			fmt.Fprintln(stdout, rep.Response)
		} else {
			fmt.Fprintf(stderr, "brain chat failed: %s\n", rep.Error)
		}
		if !rep.OK {
			return 30
		}
		return 0
	case "plan":
		fs := flag.NewFlagSet("brain plan", flag.ContinueOnError)
		fs.SetOutput(stderr)
		target := fs.String("target", "", "path, proxy, codex, keys, or network")
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		if *target == "" {
			fmt.Fprintln(stderr, "brain plan requires --target")
			return 50
		}
		f := facts.Collect(ctx, runner, home, *target == "network")
		pl := brain.NewPlanner(brain.NewLlamaCLIBackend(runner, home), reg)
		plan, err := pl.Plan(ctx, brain.PlanInput{Target: *target, Home: home, Facts: f})
		if *jsonOut {
			data, _ := json.MarshalIndent(plan, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else if err == nil {
			fmt.Fprintf(stdout, "Intent: %s\nRecipe: %s\nConfidence: %.2f\n", plan.Decision.Intent, plan.Decision.SelectedRecipe.ID, plan.Decision.Confidence)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		return 0
	case "rescue-plan":
		fs := flag.NewFlagSet("brain rescue-plan", flag.ContinueOnError)
		fs.SetOutput(stderr)
		target := fs.String("target", "network", "network or clash-tun")
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		decision := orchestrator.RescuePlan(ctx, runner, orchestrator.Options{Target: *target, DryRun: true, Yes: false, RulesDir: rulesDir, Home: home})
		if *jsonOut {
			data, _ := json.MarshalIndent(decision, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Intent: %s\nFailure: %s\nRecipe: %s\nConfidence: %.2f\n%s\n", decision.Intent, decision.FailureClass, decision.SelectedRecipe, decision.Confidence, decision.ExplanationForUser)
		}
		return 0
	case "server":
		return runBrainServer(ctx, runner, home, args[1:], stdout, stderr)
	case "explain":
		if len(args) != 2 || args[1] != "--latest" {
			fmt.Fprintln(stderr, "brain explain requires --latest")
			return 50
		}
		store := session.NewStore(home)
		sess, err := store.Latest()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		fmt.Fprint(stdout, session.HumanReport(sess))
		return 0
	default:
		fmt.Fprintln(stderr, "unknown brain subcommand")
		return 50
	}
}

func runBrainServer(ctx context.Context, runner command.Runner, home string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "brain server requires start, stop, status, or verify")
		return 50
	}
	fs := flag.NewFlagSet("brain server "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 8080, "local port")
	jsonOut := fs.Bool("json", false, "JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	var rep brain.ServerReport
	switch args[0] {
	case "start":
		rep = brain.StartServer(ctx, runner, home, *port)
	case "stop":
		rep = brain.StopServer(home, *port)
	case "status", "verify":
		rep = brain.ServerStatus(home, *port)
	default:
		fmt.Fprintln(stderr, "brain server requires start, stop, status, or verify")
		return 50
	}
	if *jsonOut {
		fmt.Fprintln(stdout, brain.MarshalServer(rep))
	} else {
		fmt.Fprintf(stdout, "Brain server: %s\nURL: %s\n", rep.Status, rep.URL)
		if rep.Error != "" {
			fmt.Fprintf(stderr, "%s\n", rep.Error)
		}
	}
	if rep.Status == "failed" {
		return 30
	}
	if args[0] == "verify" && rep.Status != "running" {
		return 20
	}
	return 0
}

func runBrainFetch(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("brain fetch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	_ = fs.String("model", brain.DefaultModelID, "model id")
	_ = fs.String("runtime", "llama.cpp", "runtime")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	script := findScript("fetch_brain_assets.sh")
	if script == "" {
		fmt.Fprintln(stderr, "fetch script not found; run scripts/fetch_brain_assets.sh from the source checkout")
		return 30
	}
	fetchRunner := command.ExecRunner{Timeout: 4 * time.Hour}
	res := fetchRunner.Run(ctx, "/bin/bash", script)
	fmt.Fprint(stdout, res.Stdout)
	if res.ExitCode != 0 {
		fmt.Fprint(stderr, res.Stderr)
		return 30
	}
	return 0
}

func runInstaller(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "installer requires list, doctor, inspect, dry-run, install, verify, or open")
		return 50
	}
	catalog, err := installer.LoadCatalog(installer.FindCatalogPath())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("installer list", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		if *jsonOut {
			data, _ := json.MarshalIndent(catalog.Installers, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			for _, in := range catalog.Installers {
				fmt.Fprintf(stdout, "%s\t%s\t%s\n", in.ID, in.Risk, in.DisplayName)
			}
		}
		return 0
	case "doctor":
		fs := flag.NewFlagSet("installer doctor", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := installer.Doctor(ctx, runner, catalog, system.Version)
		if *jsonOut {
			data, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Cactus AgentLink Installer Center %s\n", system.Version)
			for _, r := range rep.Reports {
				fmt.Fprintf(stdout, "%s\t%s\t%s\n", r.ID, r.Status, r.NextAction)
			}
		}
		return 0
	case "inspect":
		id, jsonOut, ok := parseInstallerIDFlag("installer inspect", args[1:], stderr)
		if !ok {
			return 50
		}
		in, found := catalog.Find(id)
		if !found {
			fmt.Fprintf(stderr, "unknown installer: %s\n", id)
			return 50
		}
		if jsonOut {
			data, _ := json.MarshalIndent(in, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "%s\nRisk: %s\nDefault method: %s\n", in.DisplayName, in.Risk, in.DefaultMethod)
			for _, w := range in.Warnings {
				fmt.Fprintf(stdout, "Warning: %s\n", w)
			}
		}
		return 0
	case "dry-run", "install", "verify", "open":
		return runInstallerAction(ctx, runner, catalog, args[0], args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "unknown installer subcommand")
		return 50
	}
}

func parseInstallerIDFlag(name string, args []string, stderr io.Writer) (string, bool, bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "JSON")
	parseArgs := args
	id := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		id = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return "", false, false
	}
	if id == "" && fs.NArg() == 1 {
		id = fs.Arg(0)
	}
	if id == "" {
		fmt.Fprintf(stderr, "%s requires installer id\n", name)
		return "", *jsonOut, false
	}
	return id, *jsonOut, true
}

func runInstallerAction(ctx context.Context, runner command.Runner, catalog installer.Catalog, action string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("installer "+action, flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "JSON")
	yes := fs.Bool("yes", false, "execute install")
	method := fs.String("method", "", "installer method")
	parseArgs := args
	id := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		id = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return 50
	}
	if id == "" && fs.NArg() == 1 {
		id = fs.Arg(0)
	}
	if id == "" {
		fmt.Fprintf(stderr, "installer %s requires installer id\n", action)
		return 50
	}
	rep, err := installer.Run(ctx, runner, catalog, installer.Options{
		Action:  installer.Action(action),
		ID:      id,
		Method:  *method,
		Yes:     *yes,
		Version: system.Version,
	})
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "%s: %s\n", rep.ID, rep.Status)
		if rep.NextAction != "" {
			fmt.Fprintf(stdout, "Next: %s\n", rep.NextAction)
		}
		for _, cmd := range rep.Commands {
			fmt.Fprintf(stdout, "Command: %s\n", cmd.Display)
		}
		for _, w := range rep.Warnings {
			fmt.Fprintf(stdout, "Warning: %s\n", w)
		}
	}
	if err != nil {
		if !*jsonOut {
			fmt.Fprintln(stderr, err)
		}
		return 30
	}
	return 0
}

func runOpenCode(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "opencode requires doctor, install, configure-local-gemma, install-plugin, verify, or open")
		return 50
	}
	home := currentHome(ctx, runner)
	switch args[0] {
	case "doctor":
		return printOpenCode(opencode.Doctor(ctx, runner, home), contains(args[1:], "--json"), stdout)
	case "install":
		fs := flag.NewFlagSet("opencode install", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dryRun := fs.Bool("dry-run", false, "dry run")
		yes := fs.Bool("yes", false, "install")
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		if *dryRun || !*yes {
			return printOpenCode(opencode.InstallDryRun(), *jsonOut, stdout)
		}
		catalog, err := installer.LoadCatalog(installer.FindCatalogPath())
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		rep, err := installer.Run(ctx, runner, catalog, installer.Options{Action: installer.ActionInstall, ID: "opencode-cli", Yes: true, Version: system.Version})
		if *jsonOut {
			data, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "%s: %s\n", rep.ID, rep.Status)
		}
		if err != nil {
			return 30
		}
		return 0
	case "configure-local-gemma":
		fs := flag.NewFlagSet("opencode configure-local-gemma", flag.ContinueOnError)
		fs.SetOutput(stderr)
		dryRun := fs.Bool("dry-run", false, "dry run")
		yes := fs.Bool("yes", false, "write config if safe")
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		return printOpenCode(opencode.ConfigureLocalGemma(home, *yes && !*dryRun), *jsonOut, stdout)
	case "install-plugin":
		return printOpenCode(opencode.InstallPluginDryRun(), contains(args[1:], "--json"), stdout)
	case "verify":
		return printOpenCode(opencode.Verify(ctx, runner, home), contains(args[1:], "--json"), stdout)
	case "open":
		path := "/usr/bin/open"
		res := runner.Run(ctx, path, "-a", "Terminal")
		if res.ExitCode != 0 {
			fmt.Fprintln(stderr, res.Stderr+res.Error)
			return 30
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unknown opencode subcommand")
		return 50
	}
}

func printOpenCode(rep opencode.Report, jsonOut bool, stdout io.Writer) int {
	if jsonOut {
		fmt.Fprintln(stdout, opencode.Marshal(rep))
	} else {
		fmt.Fprint(stdout, opencode.Human(rep))
	}
	if rep.Status == "failed" {
		return 30
	}
	return 0
}

func runProxyApp(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "clean-reinstall" {
		fmt.Fprintln(stderr, "proxyapp requires clean-reinstall <id>")
		return 50
	}
	parseArgs := args[1:]
	appID := ""
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		appID = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	fs := flag.NewFlagSet("proxyapp clean-reinstall", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "dry run")
	yes := fs.Bool("yes", false, "quarantine final-resort app residue")
	jsonOut := fs.Bool("json", false, "JSON")
	if err := fs.Parse(parseArgs); err != nil {
		return 50
	}
	if appID == "" && fs.NArg() == 1 {
		appID = fs.Arg(0)
	}
	if appID == "" {
		fmt.Fprintln(stderr, "proxyapp clean-reinstall requires app id")
		return 50
	}
	runYes := *yes && !*dryRun
	rep := proxyapp.CleanReinstall(ctx, runner, currentHome(ctx, runner), appID, runYes)
	if *jsonOut {
		fmt.Fprintln(stdout, proxyapp.Marshal(rep))
	} else {
		fmt.Fprintf(stdout, "Proxy app %s clean reinstall: %s\n", appID, rep.Status)
		for _, action := range rep.Actions {
			fmt.Fprintf(stdout, "- %s\n", action)
		}
		for _, warning := range rep.Warnings {
			fmt.Fprintf(stdout, "Warning: %s\n", warning)
		}
		if rep.NextAction != "" {
			fmt.Fprintf(stdout, "Next: %s\n", rep.NextAction)
		}
	}
	if rep.Status == "failed" {
		if rep.Error != "" {
			fmt.Fprintln(stderr, rep.Error)
		}
		return 30
	}
	if rep.Status == "manual_action_required" {
		return 20
	}
	return 0
}

func runDiagnose(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "tun" {
		fs := flag.NewFlagSet("diagnose tun", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: rulesDir, Verbose: true})
		r := engine.Run(ctx)
		classify.Apply(&r)
		tun := diagnose.DiagnoseTun(r)
		if *jsonOut {
			data, _ := json.MarshalIndent(tun, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Suspicious TUN signatures: %d\nRecommended repair: %s\n", len(tun.SuspiciousSignatures), tun.RecommendedRepair)
			for _, sig := range tun.SuspiciousSignatures {
				fmt.Fprintf(stdout, "- %s\n", sig)
			}
		}
		return 0
	}
	fs := flag.NewFlagSet("diagnose", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	verbose := fs.Bool("verbose", false, "include raw command output")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: rulesDir, Verbose: *verbose})
	r := engine.Run(ctx)
	classify.Apply(&r)
	if path, err := report.WriteDiagnosticForUser(&r, system.UserInfo{Name: r.Host.RealUser, Home: r.Host.RealUserHome, UID: r.Host.RealUserUID, GID: r.Host.RealUserGID}); err == nil {
		r.ReportPath = path
	} else {
		fmt.Fprintf(stderr, "warning: could not write diagnostic report: %v\n", err)
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(r, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprint(stdout, report.HumanDiagnostic(r))
	}
	return 0
}

func runClassify(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	engine := diagnose.NewEngine(runner, diagnose.Options{RulesDir: rulesDir})
	r := engine.Run(ctx)
	classify.Apply(&r)
	if path, err := report.WriteDiagnosticForUser(&r, system.UserInfo{Name: r.Host.RealUser, Home: r.Host.RealUserHome, UID: r.Host.RealUserUID, GID: r.Host.RealUserGID}); err == nil {
		r.ReportPath = path
	}
	if *jsonOut {
		out := map[string]any{"classifications": r.Classifications, "recommendedRepairLevel": r.RecommendedRepairLevel, "reportPath": r.ReportPath}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Classifications: %s\nRecommended repair level: %s\nReport path: %s\n", strings.Join(r.Classifications, ", "), r.RecommendedRepairLevel, r.ReportPath)
	}
	return 0
}

func runVerify(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "verify requires network or airdrop")
		return 50
	}
	switch args[0] {
	case "network":
		fs := flag.NewFlagSet("verify network", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := networkverify.Run(ctx, runner, rulesDir, false)
		if *jsonOut {
			data, _ := json.MarshalIndent(rep, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "Network OK: %v\nRecommendation: %s\nClasses: %s\n", rep.OK, rep.NextRecommendation, strings.Join(rep.FailureClasses, ", "))
		}
		if !rep.OK {
			return 20
		}
		return 0
	case "airdrop":
		fs := flag.NewFlagSet("verify airdrop", flag.ContinueOnError)
		fs.SetOutput(stderr)
		jsonOut := fs.Bool("json", false, "JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 50
		}
		rep := networkverify.Run(ctx, runner, rulesDir, false)
		out := map[string]any{"toolVersion": system.Version, "awdl0Up": rep.AWDLUp, "firewallBlockAll": rep.FirewallBlockAll, "ok": rep.AWDLUp}
		if *jsonOut {
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stdout, "AWDL up: %v\n", rep.AWDLUp)
		}
		if !rep.AWDLUp {
			return 20
		}
		return 0
	default:
		fmt.Fprintln(stderr, "verify requires network or airdrop")
		return 50
	}
}

func runRestartGate(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "restart-gate requires prepare or verify")
		return 50
	}
	fs := flag.NewFlagSet("restart-gate "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "JSON")
	afterRestorePoint := fs.String("after-restore-point", "", "restore point ID after targeted repair")
	incident := fs.String("incident", "", "incident ID/path after targeted repair")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	var rep restartgate.Report
	switch args[0] {
	case "prepare":
		rep = restartgate.Prepare(ctx, runner, rulesDir, restartgate.Context{AfterRestorePoint: *afterRestorePoint, Incident: *incident})
	case "verify":
		rep = restartgate.Verify(ctx, runner, rulesDir)
	default:
		fmt.Fprintln(stderr, "restart-gate requires prepare or verify")
		return 50
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "Restart required: %v\nReason: %s\nPost-restart: %s\n", rep.RestartRequired, rep.Reason, rep.PostRestartCommand)
	}
	if rep.RestartRequired {
		return 20
	}
	return 0
}

func runTicket(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "create" {
		fmt.Fprintln(stderr, "ticket requires create")
		return 50
	}
	fs := flag.NewFlagSet("ticket create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ticketType := fs.String("type", "", "clash-tun-fix, rollback, or verify-network")
	id := fs.String("id", "", "restore point id")
	packageRoot := fs.String("package-root", "", "explicit AgentLink package root")
	jsonOut := fs.Bool("json", false, "JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 50
	}
	rep := ticket.Create(ctx, runner, ticket.Options{Home: currentHome(ctx, runner), Type: *ticketType, ID: *id, Version: system.Version, PackageRoot: *packageRoot})
	if *jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprint(stdout, ticket.Human(rep))
	}
	if rep.Status == "failed" {
		return 30
	}
	return 0
}

func runRescue(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rescue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	level := fs.String("level", repair.LevelSafe, "safe, tun, standard, clean-baseline, standard-system-reset, or deep")
	yes := fs.Bool("yes", false, "confirm destructive steps")
	dryRun := fs.Bool("dry-run", false, "show actions without changing system")
	jsonOut := fs.Bool("json", false, "print JSON")
	reboot := fs.Bool("reboot", false, "reboot after deep rescue")
	includeNE := fs.Bool("include-networkextension-plists", false, "include NetworkExtension plists during deep rescue")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *level == repair.LevelDeep && !*yes {
		fmt.Fprintln(stderr, "deep rescue requires --yes")
		return 50
	}
	if *level == repair.LevelStandardSystemReset && !*yes {
		fmt.Fprintln(stderr, "standard-system-reset rescue requires --yes")
		return 50
	}
	if *level == repair.LevelCleanBaseline && !*dryRun && !*yes {
		fmt.Fprintln(stderr, "clean-baseline rescue requires --yes")
		return 50
	}
	if *level == repair.LevelTun && !*dryRun && !*yes {
		fmt.Fprintln(stderr, "tun rescue requires --yes")
		return 50
	}
	if !*dryRun && !system.IsRoot() {
		return sudoReexec(append([]string{"rescue"}, args...), stdout, stderr)
	}
	res := repair.Run(ctx, runner, repair.Options{
		Level:                         *level,
		Yes:                           *yes,
		DryRun:                        *dryRun,
		JSON:                          *jsonOut,
		Reboot:                        *reboot,
		IncludeNetworkExtensionPlists: *includeNE,
		RulesDir:                      rulesDir,
		Stdin:                         os.Stdin,
		Stdout:                        stdout,
	})
	if *jsonOut {
		fmt.Fprintln(stdout, repair.JSON(res))
	} else {
		if res.DryRun {
			printDryRun(stdout, res)
		} else {
			if res.ReportPath != "" {
				if data, err := os.ReadFile(res.ReportPath); err == nil {
					fmt.Fprint(stdout, string(data))
				} else {
					printRescueSummary(stdout, res)
				}
			} else {
				printRescueSummary(stdout, res)
			}
		}
	}
	if res.Error != "" {
		fmt.Fprintln(stderr, res.Error)
	}
	return res.ExitCode
}

func printRescueSummary(stdout io.Writer, res repair.Result) {
	fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s\n\n", res.ToolVersion)
	fmt.Fprintf(stdout, "Status: %s\n", res.Status)
	if res.RestorePointPath != "" {
		fmt.Fprintf(stdout, "Restore point: %s\n", res.RestorePointPath)
	}
	if res.ReportPath != "" {
		fmt.Fprintf(stdout, "Report path: %s\n", res.ReportPath)
	}
	if res.RollbackCommand != "" {
		fmt.Fprintf(stdout, "Rollback: %s\n", res.RollbackCommand)
	}
}

func runRollback(ctx context.Context, runner command.Runner, rulesDir string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	fs.SetOutput(stderr)
	last := fs.Bool("last", false, "use latest restore point")
	id := fs.String("id", "", "restore point id")
	dryRun := fs.Bool("dry-run", false, "show actions without changing system")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if !*dryRun && !system.IsRoot() {
		return sudoReexec(append([]string{"rollback"}, args...), stdout, stderr)
	}
	res := rollback.Run(ctx, runner, rollback.Options{ID: *id, Last: *last, DryRun: *dryRun, JSON: *jsonOut, RulesDir: rulesDir})
	if *jsonOut {
		fmt.Fprintln(stdout, rollback.JSON(res))
	} else {
		fmt.Fprint(stdout, rollback.Human(res))
	}
	if res.Error != "" {
		fmt.Fprintln(stderr, res.Error)
	}
	return res.ExitCode
}

func runReport(ctx context.Context, runner command.Runner, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	latest := fs.Bool("latest", false, "show latest report")
	id := fs.String("id", "", "restore point/report id")
	jsonOut := fs.Bool("json", false, "print JSON")
	forHuman := fs.Bool("for-human", false, "print latest human session report")
	forCodex := fs.Bool("for-codex", false, "print latest agent dispatch report")
	if err := fs.Parse(args); err != nil {
		return 50
	}
	if *forHuman || *forCodex {
		if !*latest {
			fmt.Fprintln(stderr, "report --for-human/--for-codex requires --latest")
			return 50
		}
		store := session.NewStore(currentHome(ctx, runner))
		sess, err := store.Latest()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		path := sess.HumanReportPath
		if *forCodex {
			path = sess.AgentDispatchPath
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 30
		}
		fmt.Fprint(stdout, string(data))
		return 0
	}
	if !*latest && *id == "" {
		fmt.Fprintln(stderr, "report requires --latest or --id")
		return 50
	}
	if *id != "" {
		return printRestoreReport(*id, *jsonOut, stdout, stderr)
	}
	if rp, err := snapshot.Latest(system.RestorePointsDir()); err == nil {
		if *jsonOut {
			data, _ := json.MarshalIndent(rp.Manifest, "", "  ")
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		if data, err := os.ReadFile(rp.Manifest.HumanReportPath); err == nil {
			fmt.Fprint(stdout, string(data))
			return 0
		}
	}
	u := system.RealConsoleUser(context.Background(), command.NewExecRunner())
	path, err := report.LatestDiagnosticPath(u.Home)
	if err != nil {
		fmt.Fprintln(stderr, "no report found")
		return 30
	}
	if *jsonOut {
		data, _ := os.ReadFile(path)
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	r, err := report.LoadDiagnostic(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	fmt.Fprint(stdout, report.HumanDiagnostic(r))
	return 0
}

type paramFlags struct {
	values map[string]string
}

func (p *paramFlags) String() string {
	return fmt.Sprint(p.values)
}

func (p *paramFlags) Set(value string) error {
	if p.values == nil {
		p.values = map[string]string{}
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || parts[0] == "" {
		return fmt.Errorf("expected key=value")
	}
	p.values[parts[0]] = parts[1]
	return nil
}

func currentHome(ctx context.Context, runner command.Runner) string {
	if !system.IsRoot() {
		if home := os.Getenv("HOME"); home != "" {
			return home
		}
	}
	u := system.RealConsoleUser(ctx, runner)
	if u.Home != "" {
		return u.Home
	}
	home, _ := os.UserHomeDir()
	return home
}

func existsText(ok bool) string {
	if ok {
		return "exists"
	}
	return "missing"
}

func loadUserSnapshot(home, id string) (snapshot.RestorePoint, error) {
	if id == "" {
		return snapshot.Latest(system.UserRestorePointsDir(home))
	}
	return snapshot.Load(filepath.Join(system.UserRestorePointsDir(home), filepath.Base(id)))
}

func manifestPaths(m snapshot.Manifest) []string {
	var out []string
	for _, entry := range m.Entries {
		if entry.OriginalPath != "" {
			out = append(out, entry.OriginalPath)
		}
	}
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func printRestoreReport(id string, jsonOut bool, stdout, stderr io.Writer) int {
	rp, err := snapshot.Load(filepath.Join(system.RestorePointsDir(), filepath.Base(id)))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 30
	}
	if jsonOut {
		data, _ := json.MarshalIndent(rp.Manifest, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	if data, err := os.ReadFile(rp.Manifest.HumanReportPath); err == nil {
		fmt.Fprint(stdout, string(data))
		return 0
	}
	fmt.Fprintf(stdout, "Restore point: %s\nStatus: %s\n", rp.Path, rp.Manifest.RollbackStatus)
	return 0
}

func runSelftest(stdout, stderr io.Writer) int {
	errs := verify.Selftest()
	if len(errs) == 0 {
		fmt.Fprintf(stdout, "agentlink selftest OK\n")
		return 0
	}
	for _, err := range errs {
		fmt.Fprintln(stderr, err)
	}
	return 30
}

func printDryRun(stdout io.Writer, res repair.Result) {
	fmt.Fprintf(stdout, "Cactus AgentLink Rescue %s\n\n", res.ToolVersion)
	fmt.Fprintf(stdout, "Dry run: no changes made.\n")
	fmt.Fprintf(stdout, "Level: %s\n\n", res.Level)
	fmt.Fprintf(stdout, "Preflight classes: %s\n\n", strings.Join(res.Preflight, ", "))
	fmt.Fprintln(stdout, "Planned actions:")
	for _, a := range res.Actions {
		if len(a.Command) > 0 {
			fmt.Fprintf(stdout, "- %s: %s\n", a.Description, strings.Join(a.Command, " "))
		} else {
			fmt.Fprintf(stdout, "- %s\n", a.Description)
		}
	}
}

func sudoReexec(args []string, stdout, stderr io.Writer) int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 40
	}
	sudo := "/usr/bin/sudo"
	cmd := exec.Command(sudo, append([]string{exe}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return 40
	}
	return 0
}

func findRulesDir() string {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "rules"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "rules"), filepath.Join(dir, "..", "rules"), filepath.Join(dir, "..", "..", "rules"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}

func findScript(name string) string {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "scripts", name))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "..", "scripts", name), filepath.Join(dir, "scripts", name))
	}
	for _, c := range candidates {
		if system.Exists(c) {
			return c
		}
	}
	return ""
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  agentlink doctor [--json]")
	fmt.Fprintln(w, "  agentlink readiness doctor [--json]")
	fmt.Fprintln(w, "  agentlink last-good save [--name NAME] [--json]")
	fmt.Fprintln(w, "  agentlink last-good list|inspect|restore [--id ID] [--last] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink support bundle [--output PATH] [--json]")
	fmt.Fprintln(w, "  agentlink package doctor|repair [--package-root PATH] [--dry-run] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink journal list|inspect|recover [--yes] [--package-root PATH] [--json]")
	fmt.Fprintln(w, "  agentlink chaos list|run [--root PATH] [--fixture PATH] [--json]")
	fmt.Fprintln(w, "  agentlink --safe-mode doctor|support bundle|package doctor|journal list|journal recover")
	fmt.Fprintln(w, "  agentlink dev doctor [--json]")
	fmt.Fprintln(w, "  agentlink snapshot")
	fmt.Fprintln(w, "  agentlink diff [--snapshot ID]")
	fmt.Fprintln(w, "  agentlink restore last")
	fmt.Fprintln(w, "  agentlink recipe list [--json]")
	fmt.Fprintln(w, "  agentlink recipe inspect <id> [--json]")
	fmt.Fprintln(w, "  agentlink recipe run <id> [--dry-run] [--yes] [--json] [--param key=value]")
	fmt.Fprintln(w, "  agentlink repair --target path|proxy|codex|keys [--dry-run] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink config doctor [--json]")
	fmt.Fprintln(w, "  agentlink proxy detect [--json]")
	fmt.Fprintln(w, "  agentlink keys doctor [--json]")
	fmt.Fprintln(w, "  agentlink planner validate <decision.json>")
	fmt.Fprintln(w, "  agentlink orchestrator rescue [--target auto|network|clash-tun|proxy|codex|keys|readiness] [--dry-run] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink guided rescue [--target auto|path|proxy|codex|keys|network] [--dry-run] [--yes] [--json]  # deprecated compatibility alias")
	fmt.Fprintln(w, "  agentlink field macbook-network-rescue [--fix-tun] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink diagnose tun [--json]")
	fmt.Fprintln(w, "  agentlink verify network|airdrop [--json]")
	fmt.Fprintln(w, "  agentlink restart-gate prepare|verify [--after-restore-point ID] [--incident ID] [--json]")
	fmt.Fprintln(w, "  agentlink ticket create --type clash-tun-fix|rollback|verify-network [--id ID] [--package-root PATH] [--json]")
	fmt.Fprintln(w, "  agentlink brain doctor [--json]")
	fmt.Fprintln(w, "  agentlink brain fetch [--model gemma-4-e4b-it-q4km] [--runtime llama.cpp]")
	fmt.Fprintln(w, "  agentlink brain selftest [--json]")
	fmt.Fprintln(w, "  agentlink brain prompt --text \"...\" [--json]")
	fmt.Fprintln(w, "  agentlink brain chat --prompt \"...\" [--json]")
	fmt.Fprintln(w, "  agentlink brain plan --target path|proxy|codex|keys|network [--json]")
	fmt.Fprintln(w, "  agentlink brain rescue-plan --target network|clash-tun [--json]")
	fmt.Fprintln(w, "  agentlink brain server start|stop|status|verify [--port 8080] [--json]")
	fmt.Fprintln(w, "  agentlink installer list [--json]")
	fmt.Fprintln(w, "  agentlink installer doctor [--json]")
	fmt.Fprintln(w, "  agentlink installer inspect <id> [--json]")
	fmt.Fprintln(w, "  agentlink installer dry-run|install|verify|open <id> [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink opencode doctor|install|configure-local-gemma|install-plugin|verify [--dry-run] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink proxyapp clean-reinstall clash-verge-rev [--dry-run] [--yes] [--json]")
	fmt.Fprintln(w, "  agentlink repair --auto --brain --target path|proxy|codex|keys|network [--dry-run] [--yes] [--online] [--json]")
	fmt.Fprintln(w, "  agentlink diagnose [--json] [--verbose]")
	fmt.Fprintln(w, "  agentlink classify [--json]")
	fmt.Fprintln(w, "  agentlink rescue [--level safe|tun|standard|clean-baseline|standard-system-reset|deep] [--yes] [--dry-run] [--json]")
	fmt.Fprintln(w, "  agentlink rollback [--last | --id RESTORE_POINT_ID] [--dry-run] [--json]")
	fmt.Fprintln(w, "  agentlink report [--latest | --id REPORT_ID] [--json]")
	fmt.Fprintln(w, "  agentlink selftest")
	fmt.Fprintln(w, "  agentlink version")
}
