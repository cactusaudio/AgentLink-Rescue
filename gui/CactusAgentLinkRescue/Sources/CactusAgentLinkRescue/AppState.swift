import Foundation
import SwiftUI

@MainActor
final class AppState: ObservableObject {
    @Published var page: RescuePage = .dashboard
    @Published var target: String = "path" {
        didSet {
            guard oldValue != target else { return }
            plan = nil
            planResult = nil
            dryRun = nil
            dryRunResult = nil
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

    let client = AgentlinkClient()

    var packageType: String {
        brainDoctor?.brainPackAvailable == true ? "Brain" : "Core"
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
        await runVersion()
        await runSelftest()
        await runDoctor()
        await runBrainDoctor()
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

    func rollbackLast() async {
        await runGuarded(mutating: true) {
            let result = await client.run(["restore", "last", "--json"], timeout: 120)
            latestResult = result
            rollbackOutput = result.combinedOutput
            appendLog(result)
            await loadReports()
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
            reportLoadError = nil
        } else if reportLoadError == nil {
            codexDispatch = ""
            reportLoadError = codex.combinedOutput.isEmpty ? "No latest Codex dispatch found." : codex.combinedOutput
        }
        reportsLoadedAt = Date()
    }

    func runGuarded(mutating: Bool, operation: () async -> Void) async {
        if isRunning { return }
        isRunning = true
        defer { isRunning = false }
        await operation()
    }

    func appendLog(_ result: CommandResult) {
        let line = "\(result.commandDisplay) -> \(result.exitCode) in \(String(format: "%.2f", result.duration))s"
        logLines.insert(line, at: 0)
        if logLines.count > 100 {
            logLines.removeLast(logLines.count - 100)
        }
    }

    func sudoRescueCommand(level: String) -> String {
        client.copyableTerminalCommand(["rescue", "--level", level], sudo: true)
    }

    func sudoDeepCommand() -> String {
        client.copyableTerminalCommand(["rescue", "--level", "deep", "--yes"], sudo: true)
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
