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
    var runtimeExists: Bool?
    var runtimeExecutable: Bool?
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
    var plannedActions: [PlannedAction]?
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

enum RescuePage: String, CaseIterable, Identifiable {
    case dashboard = "Dashboard"
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
