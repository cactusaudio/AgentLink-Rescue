import Foundation

struct CommandResult: Identifiable, Codable {
    var id = UUID()
    var exitCode: Int32
    var stdout: String
    var stderr: String
    var duration: TimeInterval
    var commandDisplay: String
    var startedAt: Date
    var endedAt: Date
    var timedOut: Bool

    var succeeded: Bool { exitCode == 0 && !timedOut }
    var combinedOutput: String {
        [stdout, stderr].filter { !$0.isEmpty }.joined(separator: "\n")
    }
    var stderrOrFallback: String {
        if !stderr.isEmpty { return stderr }
        if !combinedOutput.isEmpty { return combinedOutput }
        return "command failed"
    }
}

struct DoctorReport: Codable {
    var toolVersion: String?
    var classifications: [String]?
    var recommendedRepairLevel: String?
    var warnings: [String]?
    var reportPath: String?

    enum CodingKeys: String, CodingKey {
        case toolVersion
        case classifications
        case likelyFailureClasses
        case recommendedRepairLevel
        case warnings
        case reportPath
        case network
    }

    init(toolVersion: String? = nil, classifications: [String]? = nil, recommendedRepairLevel: String? = nil, warnings: [String]? = nil, reportPath: String? = nil) {
        self.toolVersion = toolVersion
        self.classifications = classifications
        self.recommendedRepairLevel = recommendedRepairLevel
        self.warnings = warnings
        self.reportPath = reportPath
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        toolVersion = try container.decodeIfPresent(String.self, forKey: .toolVersion)
        reportPath = try container.decodeIfPresent(String.self, forKey: .reportPath)
        let topWarnings = try container.decodeIfPresent([String].self, forKey: .warnings)
        let directClasses = try container.decodeIfPresent([String].self, forKey: .classifications)
        let likelyClasses = try container.decodeIfPresent([String].self, forKey: .likelyFailureClasses)
        let directRecommendation = try container.decodeIfPresent(String.self, forKey: .recommendedRepairLevel)

        if let nested = try container.decodeIfPresent(NetworkDiagnostic.self, forKey: .network) {
            classifications = directClasses ?? likelyClasses ?? nested.classifications
            recommendedRepairLevel = directRecommendation ?? nested.recommendedRepairLevel
            var mergedWarnings = topWarnings ?? []
            if let nestedWarnings = nested.warnings {
                mergedWarnings.append(contentsOf: nestedWarnings)
            }
            warnings = mergedWarnings.isEmpty ? nil : mergedWarnings
        } else {
            classifications = directClasses ?? likelyClasses
            recommendedRepairLevel = directRecommendation
            warnings = topWarnings
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encodeIfPresent(toolVersion, forKey: .toolVersion)
        try container.encodeIfPresent(classifications, forKey: .classifications)
        try container.encodeIfPresent(recommendedRepairLevel, forKey: .recommendedRepairLevel)
        try container.encodeIfPresent(warnings, forKey: .warnings)
        try container.encodeIfPresent(reportPath, forKey: .reportPath)
    }
}

private struct NetworkDiagnostic: Codable {
    var classifications: [String]?
    var recommendedRepairLevel: String?
    var warnings: [String]?
}

struct BrainDoctorReport: Codable {
    var backend: String?
    var brainPackAvailable: Bool?
    var packageRoot: String?
    var modelFamily: String?
    var modelID: String?
    var modelName: String?
    var modelPath: String?
    var modelExists: Bool?
    var modelSha256OK: Bool?
    var modelSizeBytes: Int64?
    var runtimePath: String?
    var serverPath: String?
    var runtimeExists: Bool?
    var runtimeExecutable: Bool?
    var serverExists: Bool?
    var serverExecutable: Bool?
    var runtimeArch: String?
    var packageLocalAssets: Bool?
    var userCacheAssets: Bool?
    var missingAssets: [String]?
    var fetchCommands: [String]?
    var warnings: [String]?
}

struct BrainSelftestReport: Codable {
    var ok: Bool?
    var doctor: BrainDoctorReport?
    var decision: PlannerDecisionViewModel?
    var error: String?
}

struct BrainChatReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var ok: Bool?
    var prompt: String?
    var response: String?
    var backend: String?
    var modelID: String?
    var modelPath: String?
    var durationMs: Int64?
    var error: String?
    var warnings: [String]?
}

struct PlannerDecisionViewModel: Codable {
    var schemaVersion: Int?
    var intent: String?
    var failureClass: String?
    var confidence: Double?
    var selectedRecipe: SelectedRecipe?
    var risk: String?
    var requiresUserApproval: Bool?
    var expectedVerifiers: [String]?
    var fallbackRecipes: [String]?
    var explanationForUser: String?
    var evidence: [String]?
    var stopReason: String?
}

struct SelectedRecipe: Codable {
    var id: String?
    var params: [String: String]?
}

struct PlanReport: Codable {
    var decision: PlannerDecisionViewModel?
    var validationErrors: [String]?
    var rawOutput: String?
}

struct RepairReport: Codable {
    var toolVersion: String?
    var brainEnabled: Bool?
    var status: String?
    var target: String?
    var dryRun: Bool?
    var plannerDecision: PlannerDecisionViewModel?
    var validationErrors: [String]?
    var sessionId: String?
    var humanReportPath: String?
    var agentDispatchPath: String?
    var stopReason: String?
    var error: String?
    var recipeResult: RecipeRunResult?
}

struct RecipeRunResult: Codable {
    var recipeId: String?
    var status: String?
    var dryRun: Bool?
    var snapshotId: String?
    var snapshotPath: String?
    var changedFiles: [String]?
    var plannedActions: [PlannedAction]?
    var verifierResults: [VerifierResult]?
    var warnings: [String]?
    var error: String?
}

struct GuidedRescueReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var target: String?
    var mode: String?
    var status: String?
    var cycles: [GuidedCycle]?
    var finalSummary: String?
    var humanSummary: String?
    var selectedRecipe: String?
    var requiresAdmin: Bool?
    var terminalTicketPath: String?
    var brainAvailable: Bool?
    var gemmaCalled: Bool?
    var supervisorMode: String?
    var gemmaCallCount: Int?
    var plannerUsed: Bool?
    var brainModel: String?
    var snapshotID: String?
    var rollbackAvailable: Bool?
    var rollbackCommand: String?
    var humanReportPath: String?
    var codexDispatchPath: String?
    var warnings: [String]?
    var nextSafeCommand: String?
}

struct GuidedCycle: Codable {
    var index: Int?
    var state: String?
    var stateTransitions: [String]?
    var failureClasses: [String]?
    var candidateRecipes: [String]?
    var dryRun: RecipeRunResult?
    var execution: RecipeRunResult?
    var verifiers: [VerifierResult]?
    var result: String?
}

struct FieldMode: Codable {
    var schemaVersion: Int?
    var mode: String?
    var startupView: String?
    var target: String?
    var headline: String?
    var showClashTunShortcut: Bool?
    var showNetworkRescueCommands: Bool?
    var showInstallerCenter: Bool?
    var showBrainStatus: Bool?
}

struct FieldRescueReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var fieldMode: String?
    var diagnosis: FieldDiagnosis?
    var recommendedAction: String?
    var commands: [String: String]?
    var warnings: [String]?
}

struct FieldDiagnosis: Codable {
    var classes: [String]?
    var proxyDirty: Bool?
    var clashResidueDetected: Bool?
    var networkExtensionSuspected: Bool?
    var clashTunDetected: Bool?
    var defaultRouteOK: Bool?
    var dnsOK: Bool?
}

struct VerifierResult: Codable, Identifiable {
    var id: String
    var type: String?
    var status: String?
    var evidence: String?
    var durationMs: Int?
    var error: String?
}

struct PlannedAction: Codable, Identifiable {
    var id: String?
    var description: String?
    var path: String?
}

struct RecipeSummary: Codable, Identifiable {
    var id: String
    var risk: String?
    var title: String?
}

struct SessionSummary: Codable {
    var sessionID: String?
    var selectedRecipe: String?
    var finalState: String?
    var snapshotID: String?
    var rollbackAvailable: Bool?
    var humanReportPath: String?
    var agentDispatchPath: String?
}

struct InstallerDoctorReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var installers: [InstallerReport]?
    var warnings: [String]?
}

struct InstallerReport: Codable, Identifiable {
    var schemaVersion: Int?
    var toolVersion: String?
    var id: String
    var displayName: String?
    var status: String?
    var method: String?
    var requiresNetwork: Bool?
    var requiresAdmin: Bool?
    var commands: [InstallerCommandPlan]?
    var methodStatuses: [InstallerMethodStatus]?
    var verification: [InstallerVerifyResult]?
    var warnings: [String]?
    var assetPath: String?
    var officialURL: String?
    var nextAction: String?
}

struct InstallerMethodStatus: Codable, Identifiable {
    var id: String
    var type: String?
    var available: Bool?
    var missingDependencies: [String]?
    var command: InstallerCommandPlan?
}

struct InstallerCommandPlan: Codable, Identifiable {
    var id: String { display }
    var display: String
    var path: String?
    var args: [String]?
    var mutates: Bool?
}

struct InstallerVerifyResult: Codable, Identifiable {
    var id: String
    var type: String?
    var status: String?
    var evidence: String?
    var error: String?
}

struct ReadinessReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var status: String?
    var checks: [ReadinessCheck]?
    var warnings: [String]?
    var nextActions: [String]?
}

struct ReadinessCheck: Codable, Identifiable {
    var id: String
    var title: String?
    var status: String?
    var evidence: String?
    var action: String?
}

struct DevEssentialsReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var status: String?
    var tools: [DevToolCheck]?
    var warnings: [String]?
    var nextActions: [String]?
}

struct DevToolCheck: Codable, Identifiable {
    var id: String
    var name: String?
    var required: Bool?
    var installed: Bool?
    var path: String?
    var version: String?
    var status: String?
}

struct LastGoodReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var action: String?
    var status: String?
    var profileID: String?
    var profilePath: String?
    var savedItems: [LastGoodItem]?
    var profiles: [LastGoodProfile]?
    var snapshotID: String?
    var snapshotPath: String?
    var rollbackAvailable: Bool?
    var warnings: [String]?
    var nextAction: String?
}

struct LastGoodProfile: Codable, Identifiable {
    var id: String { profileID }
    var profileID: String
    var name: String?
    var createdAt: String?
    var home: String?
    var items: [LastGoodItem]?

    enum CodingKeys: String, CodingKey {
        case profileID = "id"
        case name
        case createdAt
        case home
        case items
    }
}

struct LastGoodItem: Codable, Identifiable {
    var id: String
    var kind: String?
    var originalPath: String?
    var storedPath: String?
    var exists: Bool?
    var sha256: String?
    var restorable: Bool?
}

struct SupportBundleReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var status: String?
    var bundlePath: String?
    var files: [String]?
    var categories: [String]?
    var warnings: [String]?
}

struct PackageHealthReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var createdAt: String?
    var packageRoot: String?
    var appBundleRoot: String?
    var status: String?
    var safeMode: Bool?
    var checks: [PackageHealthCheck]?
    var actions: [String]?
    var warnings: [String]?
    var error: String?
}

struct PackageHealthCheck: Codable, Identifiable {
    var id: String
    var status: String?
    var path: String?
    var required: Bool?
    var message: String?
}

struct JournalRecoveryReport: Codable {
    var schemaVersion: Int?
    var toolVersion: String?
    var status: String?
    var transactions: [JournalTransaction]?
    var warnings: [String]?
    var nextAction: String?
}

struct JournalTransaction: Codable, Identifiable {
    var id: String { transactionID }
    var transactionID: String
    var startedAt: String?
    var endedAt: String?
    var state: String?
    var target: String?
    var recipe: String?
    var restorePoint: String?
    var packageRoot: String?
    var requiresAdmin: Bool?
    var rollbackAvailable: Bool?
}

struct BetaReadinessCheck: Identifiable {
    let id: String
    var title: String
    var status: String
    var detail: String
}

enum RescuePage: String, CaseIterable, Identifiable {
    case dashboard = "Dashboard"
    case guided = "Guided Rescue"
    case installer = "Installer Center"
    case readiness = "Readiness Center"
    case doctor = "Doctor"
    case brain = "Brain"
    case plan = "Plan"
    case dryRun = "Dry Run"
    case rescue = "Rescue"
    case rollback = "Rollback"
    case reports = "Reports"
    case settings = "Settings"

    var id: String { rawValue }
}

struct GuidedStep: Identifiable {
    let id = UUID()
    var title: String
    var detail: String
    var status: String
}
