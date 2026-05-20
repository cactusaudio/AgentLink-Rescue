import SwiftUI

struct StatusDashboardView: View {
    @EnvironmentObject var state: AppState
    @State private var confirmSupportBundle = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                rescueHero
                if needsPackageAttention || state.incompleteJournalDetected {
                    attentionCard
                }
                betaReadinessCard
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 320), spacing: 12)], spacing: 12) {
                    statusCard
                    planCard
                }
                safetyCard
                if state.developerModeEnabled {
                    developerDetails
                    OutputCard(title: "Latest Command Output", text: state.latestResult?.combinedOutput ?? "", placeholder: "Run Check & Plan Rescue to see command output.", collapsedByDefault: true)
                }
            }
            .padding()
        }
        .alert("Export Support Bundle?", isPresented: $confirmSupportBundle) {
            Button("Cancel", role: .cancel) {}
            Button("Export") {
                Task { await state.createSupportBundle() }
            }
        } message: {
            Text("This support bundle contains redacted local diagnostics, including tool presence, local paths, network/proxy status, readiness reports, and latest session metadata. It does not include private keys, browser cookies, shell history, Wi-Fi passwords, or full API keys.")
        }
    }

    private var rescueHero: some View {
        HStack(alignment: .center, spacing: 18) {
            VStack(alignment: .leading, spacing: 8) {
                Text("AgentLink Rescue")
                    .font(.title2.weight(.semibold))
                    .onTapGesture(count: 3) {
                        state.developerModeEnabled = true
                    }
                Text("Check this Mac, prepare a safe rescue plan, and export a support bundle. The app does not run sudo in the GUI; admin repairs are handed off as Terminal tickets.")
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                HStack(spacing: 8) {
                    StatusBadge(text: "Local diagnostics", kind: .ok)
                    StatusBadge(text: "Dry-run first", kind: .ok)
                    StatusBadge(text: "Approval required", kind: .ok)
                    if state.incompleteJournalDetected {
                        StatusBadge(text: "Incomplete repair", kind: .warn)
                    }
                }
            }
            Spacer(minLength: 16)
            VStack(alignment: .trailing, spacing: 12) {
                Button {
                    if state.packageBlocksMainAction {
                        Task { await state.runDoctor() }
                    } else {
                        Task {
                            await state.runGuidedRescue(allowRepair: false, target: state.fieldMode?.mode == "macbook-network-rescue" ? "network" : "auto")
                            await state.loadReports()
                        }
                    }
                } label: {
                    Label(state.packageBlocksMainAction ? "Run Diagnosis" : "Check & Plan Rescue", systemImage: state.packageBlocksMainAction ? "stethoscope" : "checklist.checked")
                        .font(.headline)
                        .frame(minWidth: 250)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isRunning)
                Text(state.packageBlocksMainAction ? "Package health limits this to diagnostics." : "Creates a dry-run rescue plan only.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                if state.isRunning {
                    Text("AgentLink is checking this Mac.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Button {
                    Task { await state.runBetaReadinessCheck() }
                } label: {
                    Label("Run Field Beta Check", systemImage: "checkmark.seal")
                }
                .buttonStyle(.bordered)
                .disabled(state.isRunning)
                Button {
                    confirmSupportBundle = true
                } label: {
                    Label("Export Support Bundle", systemImage: "shippingbox")
                }
                .buttonStyle(.bordered)
                .disabled(state.isRunning)
                Button("View Reports") {
                    state.page = .reports
                }
                .buttonStyle(.borderless)
            }
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 10))
    }

    private var attentionCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label(state.packageBlocksMainAction ? "Needs attention" : "First-run setup", systemImage: state.packageBlocksMainAction ? "exclamationmark.triangle" : "folder.badge.plus")
                    .font(.headline)
                Spacer()
                StatusBadge(text: state.packageBlocksMainAction ? "Safe Mode" : "Setup", kind: .warn)
            }
            if needsPackageAttention {
                Text(state.packageBlocksMainAction ? "AgentLink package health is \(state.packageHealthLabel). The main action is limited to diagnosis until the local package is healthy." : "AgentLink is ready to diagnose. Snapshot-backed repairs need local support folders; package repair can create them before you apply a repair.")
                    .foregroundStyle(.secondary)
                if state.packageHealth?.status == "needs_repair" || state.packageHealth?.status == "broken" {
                    Button {
                        Task { await state.repairPackage() }
                    } label: {
                        Label(state.packageBlocksMainAction ? "Repair AgentLink Package" : "Prepare Local Support Folders", systemImage: "wrench.and.screwdriver")
                    }
                    .buttonStyle(.bordered)
                    .disabled(state.isRunning)
                }
            }
            if state.incompleteJournalDetected {
                Text("A previous repair did not finish. Review the recovery report before running another repair.")
                    .foregroundStyle(.secondary)
                Button {
                    state.page = .reports
                } label: {
                    Label("Review Recovery Report", systemImage: "doc.text.magnifyingglass")
                }
                .buttonStyle(.bordered)
            }
        }
        .card()
    }

    private var betaReadinessCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .center) {
                Label("Field Beta Readiness", systemImage: "checkmark.shield")
                    .font(.headline)
                Spacer()
                StatusBadge(text: state.betaStatus, kind: betaStatusKind)
            }
            Text(state.betaSummary)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
            if !state.betaChecks.isEmpty {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 220), spacing: 10)], spacing: 10) {
                    ForEach(state.betaChecks) { check in
                        VStack(alignment: .leading, spacing: 5) {
                            HStack {
                                Text(check.title)
                                    .font(.caption.weight(.semibold))
                                Spacer()
                                StatusBadge(text: check.status, kind: betaCheckKind(check.status))
                            }
                            Text(check.detail)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .lineLimit(3)
                                .truncationMode(.middle)
                        }
                        .padding(10)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(Color(nsColor: .textBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
                    }
                }
            } else {
                Text("This check is stricter than the main rescue button: it verifies package health, journal state, CLI selftest, read-only diagnosis, guided dry-run, readiness, and support-bundle export.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            if let checkedAt = state.betaCheckedAt {
                Text("Last checked \(checkedAt.formatted(date: .abbreviated, time: .standard))")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .card()
    }

    private var statusCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Current Status").font(.headline)
            InfoRow(label: "AgentLink", value: state.selftestStatus == "OK" ? "Ready" : "Needs attention")
            InfoRow(label: "Package", value: state.packageBlocksMainAction ? "Needs local repair" : "Ready")
            InfoRow(label: "Field Beta", value: state.betaStatus)
            InfoRow(label: "Network", value: state.doctor?.classifications?.contains("OK") == true ? "Looks OK" : state.recommendationLabel)
            if let classes = state.doctor?.classifications, !classes.isEmpty {
                HStack(spacing: 6) {
                    ForEach(classes.prefix(3), id: \.self) { item in
                        StatusBadge(text: item, kind: item == "OK" ? .ok : .warn)
                    }
                }
            } else {
                Text("Run Check & Plan Rescue to refresh the network assessment.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .card()
    }

    private var planCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Latest Plan").font(.headline)
            if let guided = state.guidedReport {
                StatusBadge(text: guided.status ?? state.guidedStatus, kind: guided.status == "failed" ? .fail : .ok)
                InfoRow(label: "Target", value: guided.target ?? state.guidedLastTarget ?? "auto")
                InfoRow(label: "Recipe", value: guided.selectedRecipe ?? state.guidedLastRecipe ?? "none", monospaced: true)
                if guided.requiresAdmin == true {
                    StatusBadge(text: "Terminal ticket required", kind: .warn)
                }
                Text(guided.humanSummary ?? guided.finalSummary ?? state.guidedSummary)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                if let ticket = guided.terminalTicketPath, !ticket.isEmpty {
                    InfoRow(label: "Ticket", value: ticket, monospaced: true)
                }
            } else if let dryRun = state.dryRun {
                    StatusBadge(text: dryRun.status ?? "dry-run", kind: .ok)
                    InfoRow(label: "Target", value: dryRun.target ?? "auto")
                    InfoRow(label: "Recipe", value: dryRun.plannerDecision?.selectedRecipe?.id ?? dryRun.recipeResult?.recipeId ?? "none", monospaced: true)
                    InfoRow(label: "Risk", value: dryRun.plannerDecision?.risk ?? "unknown")
                    if let explanation = dryRun.plannerDecision?.explanationForUser, !explanation.isEmpty {
                        Text(explanation)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .fixedSize(horizontal: false, vertical: true)
                    }
            } else if state.guidedStatus == "Running" {
                StatusBadge(text: "Checking", kind: .warn)
                Text(state.guidedSummary)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            } else if let result = state.guidedResult, !result.succeeded {
                StatusBadge(text: "Failed", kind: .fail)
                Text(result.stderrOrFallback)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            } else {
                StatusBadge(text: "No plan yet", kind: .neutral)
                Text("Start with Check & Plan Rescue. AgentLink will produce a dry-run plan and support-bundle evidence before any repair.")
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            if let session = state.repair?.sessionId ?? state.dryRun?.sessionId {
                InfoRow(label: "Session", value: session, monospaced: true)
            }
        }
        .card()
    }

    private var safetyCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Safety Boundary").font(.headline)
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 220), spacing: 10)], spacing: 10) {
                safetyItem("No sudo in the GUI", "Admin work is prepared as a Terminal ticket.")
                safetyItem("Dry-run before repair", "The main button produces a plan, not a system mutation.")
                safetyItem("Reports first", "Support bundles are redacted and exportable for review.")
                safetyItem("Protected topologies preserved", "Audio, video, storage, lab, and policy-owned interfaces become report-only boundaries.")
            }
        }
        .card()
    }

    private var developerDetails: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("Developer Details").font(.headline)
                Spacer()
                StatusBadge(text: "Developer Mode", kind: .warn)
            }
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 220), spacing: 12)], spacing: 12) {
                summaryPill(title: "Package Health", value: state.packageHealthLabel, kind: packageHealthKind)
                summaryPill(title: "AgentLink Core", value: state.selftestStatus, kind: state.selftestStatus == "OK" ? .ok : .warn)
                summaryPill(title: "Brain Pack", value: state.brainDoctor?.brainPackAvailable == true ? "OK" : "Missing", kind: state.brainDoctor?.brainPackAvailable == true ? .ok : .warn)
                summaryPill(title: "Last Session", value: state.repair?.status ?? state.dryRun?.status ?? "None", kind: state.dryRun?.status == "dry-run" || state.repair?.status == "success" ? .ok : .neutral)
            }
            InfoRow(label: "Version", value: state.versionText)
            InfoRow(label: "Binary", value: state.client.binaryURL.path, monospaced: true)
            InfoRow(label: "Journal", value: state.journalRecovery?.status ?? "unknown")
            Text("Rollback is available only after a snapshot-backed repair.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .card()
    }

    private func safetyItem(_ title: String, _ detail: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.subheadline.weight(.semibold))
            Text(detail)
                .font(.caption)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .textBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
    }

    private func summaryPill(title: String, value: String, kind: StatusBadge.Kind) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title).font(.caption).foregroundStyle(.secondary)
            StatusBadge(text: value, kind: kind)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
    }

    private var packageHealthKind: StatusBadge.Kind {
        switch state.packageHealth?.status {
        case "ok", "warning", "repaired":
            return .ok
        case "needs_repair", "dry_run", "partial_repair":
            return .warn
        case "broken", "failed":
            return .fail
        default:
            return .neutral
        }
    }

    private var needsPackageAttention: Bool {
        guard let status = state.packageHealth?.status else { return false }
        return ["needs_repair", "broken", "partial_repair", "failed"].contains(status)
    }

    private var betaStatusKind: StatusBadge.Kind {
        switch state.betaStatus {
        case "Ready for field beta": return .ok
        case "Checking": return .warn
        case "Needs attention": return .fail
        default: return .neutral
        }
    }

    private func betaCheckKind(_ status: String) -> StatusBadge.Kind {
        switch status {
        case "ok": return .ok
        case "running": return .warn
        case "failed": return .fail
        default: return .neutral
        }
    }

    private func export(_ text: String, name: String) {
        let url = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Desktop").appendingPathComponent(name)
        try? text.write(to: url, atomically: true, encoding: .utf8)
    }
}

extension View {
    func card() -> some View {
        self.padding()
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
    }
}
