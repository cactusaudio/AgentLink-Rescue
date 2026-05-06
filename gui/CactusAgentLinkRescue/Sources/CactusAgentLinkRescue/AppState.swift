import Foundation
import SwiftUI

@MainActor
final class AppState: ObservableObject {
    @Published var developerModeEnabled = UserDefaults.standard.bool(forKey: "developerModeEnabled") {
        didSet { UserDefaults.standard.set(developerModeEnabled, forKey: "developerModeEnabled") }
    }
    @Published var page: RescuePage = .dashboard
    @Published var target: String = "path" {
        didSet {
            guard oldValue != target else { return }
            plan = nil
            planResult = nil
            dryRun = nil
            dryRunResult = nil
            guidedCanApply = false
        }
    }
    @Published var isRunning = false
    @Published var versionText = "unknown"
    @Published var selftestStatus = "not run"
    @Published var doctor: DoctorReport?
    @Published var brainDoctor: BrainDoctorReport?
    @Published var brainSelftest: BrainSelftestReport?
    @Published var plan: PlanReport?
    @Published var dryRun: RepairReport?
    @Published var repair: RepairReport?
    @Published var doctorResult: CommandResult?
    @Published var brainDoctorResult: CommandResult?
    @Published var brainSelftestResult: CommandResult?
    @Published var planResult: CommandResult?
    @Published var dryRunResult: CommandResult?
    @Published var repairResult: CommandResult?
    @Published var rollbackOutput: String = ""
    @Published var humanReport: String = ""
    @Published var codexDispatch: String = ""
    @Published var reportsLoadedAt: Date?
    @Published var reportLoadError: String?
    @Published var latestResult: CommandResult?
    @Published var logLines: [String] = []
    @Published var rollbackAvailable = false
    @Published var guidedStatus = "Not run"
    @Published var guidedSummary = "Run a guided rescue to detect local issues, choose bounded recipes, dry-run changes, and verify outcomes."
    @Published var guidedSteps: [GuidedStep] = []
    @Published var guidedCanApply = false
    @Published var guidedLastTarget: String?
    @Published var guidedLastRecipe: String?
    @Published var guidedResult: CommandResult?
    @Published var guidedReport: GuidedRescueReport?
    @Published var installerDoctor: InstallerDoctorReport?
    @Published var installerResult: CommandResult?
    @Published var installerActionReport: InstallerReport?
    @Published var readiness: ReadinessReport?
    @Published var readinessResult: CommandResult?
    @Published var devEssentials: DevEssentialsReport?
    @Published var devEssentialsResult: CommandResult?
    @Published var lastGood: LastGoodReport?
    @Published var lastGoodResult: CommandResult?
    @Published var supportBundle: SupportBundleReport?
    @Published var supportBundleResult: CommandResult?
    @Published var packageHealth: PackageHealthReport?
    @Published var packageHealthResult: CommandResult?
    @Published var journalRecovery: JournalRecoveryReport?
    @Published var journalRecoveryResult: CommandResult?
    @Published var brainSandboxPresented = false
    @Published var brainChatPrompt = ""
    @Published var brainChatReport: BrainChatReport?
    @Published var brainChatResult: CommandResult?
    @Published var fieldMode: FieldMode?
    @Published var fieldReport: FieldRescueReport?
    @Published var fieldResult: CommandResult?

    let client: AgentlinkClient

    init(client: AgentlinkClient = AgentlinkClient()) {
        self.client = client
        self.fieldMode = client.fieldMode()
        if let fieldMode = self.fieldMode {
            target = fieldMode.target ?? "network"
            if fieldMode.startupView == "guided-rescue" {
                page = .dashboard
                guidedStatus = "Field mode"
                guidedSummary = "MacBook Network Rescue mode is active. Start with the main Fix button; sudo repair uses a Terminal ticket."
            }
        }
    }

    var packageType: String {
        brainDoctor?.brainPackAvailable == true ? "Brain" : "Core"
    }

    var safeModeActive: Bool {
        packageHealth?.safeMode == true || packageHealth?.status == "broken"
    }

    var packageHealthLabel: String {
        packageHealth?.status ?? "unknown"
    }

    var incompleteJournalDetected: Bool {
        journalRecovery?.status == "incomplete_transactions_found"
    }

    var executeEnabled: Bool {
        guard dryRun?.status == "dry-run",
              dryRun?.target == target,
              riskAllowedInGUI(dryRun?.plannerDecision?.risk),
              dryRun?.validationErrors?.isEmpty != false
        else {
            return false
        }
        return true
    }

    var recommendationLabel: String {
        recommendationDisplay(for: doctor?.recommendedRepairLevel, classifications: doctor?.classifications).label
    }

    var recommendationKind: StatusBadge.Kind {
        recommendationDisplay(for: doctor?.recommendedRepairLevel, classifications: doctor?.classifications).kind
    }

    func bootstrap() async {
        await runPackageDoctor()
        await runJournalRecoverCheck()
        await runVersion()
        await runSelftest()
        await runDoctor()
        await runBrainDoctor()
        await runInstallerDoctor()
        if fieldMode != nil {
            await runFieldNetworkSummary()
        }
        await runReadiness()
        await loadReports()
    }

    func runVersion() async {
        let result = await client.run(["version"], timeout: 10)
        latestResult = result
        versionText = result.stdout.trimmingCharacters(in: .whitespacesAndNewlines)
        appendLog(result)
    }

    func runSelftest() async {
        let result = await client.run(["selftest"], timeout: 20)
        latestResult = result
        selftestStatus = result.succeeded ? "OK" : "failed"
        appendLog(result)
    }

    func runDoctor() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(DoctorReport.self, args: ["doctor", "--json"], timeout: 45)
            latestResult = result
            doctorResult = result
            doctor = decoded
            appendLog(result)
        }
    }

    func runPackageDoctor() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(PackageHealthReport.self, args: ["package", "doctor", "--package-root", client.packageRoot.path, "--json"], timeout: 20)
            latestResult = result
            packageHealthResult = result
            packageHealth = decoded
            appendLog(result)
        }
    }

    func repairPackage() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(PackageHealthReport.self, args: ["package", "repair", "--package-root", client.packageRoot.path, "--yes", "--json"], timeout: 60)
            latestResult = result
            packageHealthResult = result
            packageHealth = decoded
            appendLog(result)
        }
    }

    func runJournalRecoverCheck() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(JournalRecoveryReport.self, args: ["journal", "recover", "--json"], timeout: 20)
            latestResult = result
            journalRecoveryResult = result
            journalRecovery = decoded
            appendLog(result)
        }
    }

    func runBrainDoctor() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(BrainDoctorReport.self, args: ["brain", "doctor", "--json"], timeout: 30)
            latestResult = result
            brainDoctorResult = result
            brainDoctor = decoded
            appendLog(result)
        }
    }

    func runBrainSelftest() async {
        await runGuarded(mutating: false) {
            brainSelftest = nil
            brainSelftestResult = nil
            let (result, decoded) = await client.runJSON(BrainSelftestReport.self, args: ["brain", "selftest", "--json"], timeout: 180)
            latestResult = result
            brainSelftestResult = result
            brainSelftest = decoded
            appendLog(result)
        }
    }

    func runPlan() async {
        await runGuarded(mutating: false) {
            plan = nil
            planResult = nil
            let (result, decoded) = await client.runJSON(PlanReport.self, args: ["brain", "plan", "--target", target, "--json"], timeout: 180)
            latestResult = result
            planResult = result
            plan = decoded
            appendLog(result)
        }
    }

    func runDryRun() async {
        await runGuarded(mutating: false) {
            dryRun = nil
            dryRunResult = nil
            let (result, decoded) = await client.runJSON(RepairReport.self, args: ["repair", "--auto", "--brain", "--target", target, "--dry-run", "--json"], timeout: 180)
            latestResult = result
            dryRunResult = result
            dryRun = decoded
            appendLog(result)
        }
    }

    func executeRepair() async {
        guard executeEnabled else { return }
        await runGuarded(mutating: true) {
            let (result, decoded) = await client.runJSON(RepairReport.self, args: ["repair", "--auto", "--brain", "--target", target, "--yes", "--json"], timeout: 240)
            latestResult = result
            repairResult = result
            repair = decoded
            appendLog(result)
            await loadReports()
        }
    }

    func runGuidedRescue(allowRepair: Bool, target overrideTarget: String? = nil) async {
        await runGuarded(mutating: allowRepair) {
            let rescueTarget = overrideTarget ?? "auto"
            guidedStatus = "Running"
            guidedSummary = allowRepair ? "AgentLink CLI is running the Rescue Orchestrator. Privileged repairs are converted into Terminal tickets; the GUI will not ask for a password." : "AgentLink CLI is analyzing and dry-running only. No files will be changed."
            guidedSteps = [GuidedStep(title: "Start", detail: allowRepair ? "Mode: orchestrated repair. Sudo work uses a Terminal repair ticket." : "Mode: analyze-only. No mutation is allowed.", status: "running")]
            guidedCanApply = false
            guidedLastTarget = nil
            guidedLastRecipe = nil
            guidedResult = nil
            guidedReport = nil

            var args = ["orchestrator", "rescue", "--target", rescueTarget == "network" ? "clash-tun" : rescueTarget, "--json"]
            if allowRepair {
                args.append("--yes")
            } else {
                args.append("--dry-run")
            }
            let (result, decoded) = await client.runJSON(GuidedRescueReport.self, args: args, timeout: allowRepair ? 600 : 240)
            latestResult = result
            guidedResult = result
            guidedReport = decoded
            appendLog(result)

            if let decoded {
                applyGuidedReport(decoded)
            } else {
                guidedStatus = result.succeeded ? "Unknown" : "Failed"
                guidedSummary = result.stderrOrFallback
                guidedSteps = [GuidedStep(title: "Result", detail: result.stderrOrFallback, status: result.succeeded ? "warn" : "failed")]
            }
            await loadReports()
        }
    }

    func runFieldNetworkSummary() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(FieldRescueReport.self, args: ["field", "macbook-network-rescue", "--json"], timeout: 90)
            latestResult = result
            fieldResult = result
            fieldReport = decoded
            appendLog(result)
        }
    }

    func analyzeFieldNetwork() async {
        await runFieldNetworkSummary()
        await runGuidedRescue(allowRepair: false, target: "network")
    }

    func rollbackLast() async {
        await runGuarded(mutating: true) {
            let result = await client.run(["restore", "last", "--json"], timeout: 120)
            latestResult = result
            rollbackOutput = result.combinedOutput
            appendLog(result)
            await loadReports()
        }
    }

    func runInstallerDoctor() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(InstallerDoctorReport.self, args: ["installer", "doctor", "--json"], timeout: 45)
            latestResult = result
            installerResult = result
            installerDoctor = decoded
            appendLog(result)
        }
    }

    func runReadiness() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(ReadinessReport.self, args: ["readiness", "doctor", "--json"], timeout: 90)
            latestResult = result
            readinessResult = result
            readiness = decoded
            appendLog(result)
        }
    }

    func runDevEssentials() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(DevEssentialsReport.self, args: ["dev", "doctor", "--json"], timeout: 30)
            latestResult = result
            devEssentialsResult = result
            devEssentials = decoded
            appendLog(result)
        }
    }

    func saveLastGood() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(LastGoodReport.self, args: ["last-good", "save", "--json"], timeout: 30)
            latestResult = result
            lastGoodResult = result
            lastGood = decoded
            appendLog(result)
        }
    }

    func listLastGood() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(LastGoodReport.self, args: ["last-good", "list", "--json"], timeout: 20)
            latestResult = result
            lastGoodResult = result
            lastGood = decoded
            appendLog(result)
        }
    }

    func restoreLastGood() async {
        await runGuarded(mutating: true) {
            let (result, decoded) = await client.runJSON(LastGoodReport.self, args: ["last-good", "restore", "--last", "--yes", "--json"], timeout: 90)
            latestResult = result
            lastGoodResult = result
            lastGood = decoded
            rollbackAvailable = decoded?.rollbackAvailable == true
            appendLog(result)
            await loadReports()
        }
    }

    func createSupportBundle() async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(SupportBundleReport.self, args: ["support", "bundle", "--json"], timeout: 120)
            latestResult = result
            supportBundleResult = result
            supportBundle = decoded
            appendLog(result)
        }
    }

    func runInstallerDryRun(_ id: String) async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(InstallerReport.self, args: ["installer", "dry-run", id, "--json"], timeout: 45)
            latestResult = result
            installerResult = result
            installerActionReport = decoded
            appendLog(result)
        }
    }

    func runInstallerVerify(_ id: String) async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(InstallerReport.self, args: ["installer", "verify", id, "--json"], timeout: 45)
            latestResult = result
            installerResult = result
            installerActionReport = decoded
            appendLog(result)
        }
    }

    func runInstallerOpen(_ id: String) async {
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(InstallerReport.self, args: ["installer", "open", id, "--json"], timeout: 45)
            latestResult = result
            installerResult = result
            installerActionReport = decoded
            appendLog(result)
        }
    }

    func runInstallerInstall(_ id: String) async {
        await runGuarded(mutating: true) {
            let (result, decoded) = await client.runJSON(InstallerReport.self, args: ["installer", "install", id, "--yes", "--json"], timeout: 300)
            latestResult = result
            installerResult = result
            installerActionReport = decoded
            appendLog(result)
            let (doctorResult, doctorDecoded) = await client.runJSON(InstallerDoctorReport.self, args: ["installer", "doctor", "--json"], timeout: 45)
            installerDoctor = doctorDecoded
            appendLog(doctorResult)
        }
    }

    func runBrainChat() async {
        let prompt = brainChatPrompt.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !prompt.isEmpty else { return }
        await runGuarded(mutating: false) {
            let (result, decoded) = await client.runJSON(BrainChatReport.self, args: ["brain", "chat", "--prompt", prompt, "--json"], timeout: 240)
            latestResult = result
            brainChatResult = result
            brainChatReport = decoded
            appendLog(result)
        }
    }

    func loadReports() async {
        let human = await client.run(["report", "--for-human", "--latest"], timeout: 20)
        if human.succeeded {
            humanReport = human.stdout
        } else {
            humanReport = ""
            reportLoadError = human.combinedOutput.isEmpty ? "No latest human report found." : human.combinedOutput
        }
        let codex = await client.run(["report", "--for-codex", "--latest"], timeout: 20)
        if codex.succeeded {
            codexDispatch = codex.stdout
            rollbackAvailable = parseRollbackAvailable(codex.stdout)
            reportLoadError = nil
        } else if reportLoadError == nil {
            codexDispatch = ""
            rollbackAvailable = false
            reportLoadError = codex.combinedOutput.isEmpty ? "No latest Codex dispatch found." : codex.combinedOutput
        }
        reportsLoadedAt = Date()
    }

    func runGuarded(mutating: Bool, operation: () async -> Void) async {
        if mutating {
            if isRunning { return }
            isRunning = true
            defer { isRunning = false }
            await operation()
            return
        }
        await operation()
    }

    func appendLog(_ result: CommandResult) {
        let line = "\(result.commandDisplay) -> \(result.exitCode) in \(String(format: "%.2f", result.duration))s"
        logLines.insert(line, at: 0)
        if logLines.count > 100 {
            logLines.removeLast(logLines.count - 100)
        }
    }

    private func addGuidedStep(_ title: String, _ detail: String, _ status: String) {
        guidedSteps.append(GuidedStep(title: title, detail: detail, status: status))
    }

    private func applyGuidedReport(_ report: GuidedRescueReport) {
        guidedStatus = guidedStatusLabel(report.status)
        guidedSummary = report.humanSummary ?? report.finalSummary ?? "Guided rescue completed."
        guidedLastTarget = report.target
        guidedLastRecipe = report.selectedRecipe
        rollbackAvailable = report.rollbackAvailable == true
        let status = report.status ?? ""
        guidedCanApply = (status == "planned" || status == "dry_run_complete") && report.mode == "dry-run" && report.selectedRecipe != nil
        guidedSteps = []
        for cycle in report.cycles ?? [] {
            let result = cycle.result ?? "completed"
            let transitions = cycle.stateTransitions ?? []
            let title = "Cycle \(cycle.index ?? guidedSteps.count + 1)"
            var detailParts: [String] = []
            if !transitions.isEmpty {
                detailParts.append(transitions.joined(separator: " -> "))
            }
            if let recipe = report.selectedRecipe ?? cycle.candidateRecipes?.first {
                detailParts.append("Recipe: \(recipe)")
            }
            if let state = cycle.state {
                detailParts.append("State: \(state)")
            }
            if let classes = cycle.failureClasses, !classes.isEmpty {
                detailParts.append("Classes: \(classes.joined(separator: ", "))")
            }
            guidedSteps.append(GuidedStep(title: title, detail: detailParts.joined(separator: "\n"), status: guidedStepStatus(result)))
        }
        if guidedSteps.isEmpty {
            guidedSteps = [GuidedStep(title: "Result", detail: guidedSummary, status: guidedStepStatus(report.status ?? ""))]
        }
    }

    private func guidedStatusLabel(_ status: String?) -> String {
        switch status {
        case "healthy": return "Healthy"
        case "planned": return "Plan ready"
        case "dry_run_complete": return "Dry-run complete"
        case "repaired": return "Repaired"
        case "no_safe_action": return "No safe action"
        case "manual_action_required": return "Manual action required"
        case "rolled_back": return "Rolled back"
        case "ticket_created": return "Terminal ticket ready"
        case "restart_required": return "Restart required"
        case "rolled_back_after_worsening": return "Rolled back after worsening"
        case "failed": return "Failed"
        default: return status ?? "Unknown"
        }
    }

    private func guidedStepStatus(_ result: String) -> String {
        switch result {
        case "healthy", "repaired", "dry_run_complete", "planned":
            return "ok"
        case "manual_action_required", "no_safe_action", "verifier_failed", "ticket_created", "restart_required", "rolled_back_after_worsening":
            return "warn"
        case "failed", "rolled_back":
            return result == "rolled_back" ? "warn" : "failed"
        default:
            return "ok"
        }
    }

    private func parseRollbackAvailable(_ text: String) -> Bool {
        guard let data = text.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else {
            return false
        }
        return object["rollbackAvailable"] as? Bool ?? false
    }

    private func doctorResultAppend(_ result: CommandResult) {
        doctorResult = result
        appendLog(result)
    }

    private func doctorLooksHealthy(_ report: DoctorReport?) -> Bool {
        guard let report else { return false }
        let rec = recommendationDisplay(for: report.recommendedRepairLevel, classifications: report.classifications).label
        return rec == "None" && (report.classifications?.contains("OK") == true || report.classifications?.isEmpty != false)
    }

    func sudoRescueCommand(level: String) -> String {
        var args = ["rescue", "--level", level]
        if level == "tun" || level == "clean-baseline" || level == "standard-system-reset" {
            args.append("--yes")
        }
        return client.copyableTerminalCommand(args, sudo: true)
    }

    func sudoDeepCommand() -> String {
        client.copyableTerminalCommand(["rescue", "--level", "deep", "--yes"], sudo: true)
    }

    func fieldRescueCommand(level: String) -> String {
        var args = ["rescue", "--level", level]
        if level == "tun" {
            args.append("--yes")
            return client.copyableTerminalCommand(args, sudo: true)
        }
        if level == "standard" || level == "clean-baseline" || level == "standard-system-reset" || level == "deep" {
            args.append("--yes")
        }
        return client.copyableTerminalCommand(args, sudo: true)
    }

    private func riskAllowedInGUI(_ risk: String?) -> Bool {
        guard let risk else { return false }
        return ["read_only", "safe_patch", "reversible_patch"].contains(risk)
    }

    private func recommendationDisplay(for raw: String?, classifications: [String]?) -> (label: String, kind: StatusBadge.Kind) {
        let normalized = (raw ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch normalized {
        case "none":
            return ("None", .ok)
        case "safe":
            return ("Safe Rescue", .warn)
        case "standard":
            return ("Standard Rescue", .warn)
        case "deep":
            return ("Deep Rescue", .fail)
        case "":
            if classifications?.contains("OK") == true {
                return ("None", .ok)
            }
            return ("Unknown", .neutral)
        default:
            return (raw ?? "Unknown", .neutral)
        }
    }
}
