import SwiftUI

struct StatusDashboardView: View {
    @EnvironmentObject var state: AppState
    @State private var confirmSupportBundle = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                guidedHero
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 220), spacing: 12)], spacing: 12) {
                    summaryPill(title: "Package Health", value: state.packageHealthLabel, kind: packageHealthKind)
                    summaryPill(title: "AgentLink Core", value: state.selftestStatus, kind: state.selftestStatus == "OK" ? .ok : .warn)
                    summaryPill(title: "Brain Pack", value: state.brainDoctor?.brainPackAvailable == true ? "OK" : "Missing", kind: state.brainDoctor?.brainPackAvailable == true ? .ok : .warn)
                    summaryPill(title: "Network", value: state.doctor?.classifications?.contains("OK") == true ? "OK" : state.recommendationLabel, kind: state.recommendationKind)
                    summaryPill(title: "Last Session", value: state.repair?.status ?? state.dryRun?.status ?? "None", kind: state.dryRun?.status == "dry-run" || state.repair?.status == "success" ? .ok : .neutral)
                }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 320), spacing: 12)], spacing: 12) {
                    restoreCard
                    reportsCard
                }
                OutputCard(title: "Latest Command Output", text: state.latestResult?.combinedOutput ?? "", placeholder: "Run the main rescue flow to see command output.", collapsedByDefault: true)
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

    private var guidedHero: some View {
        HStack(alignment: .center, spacing: 18) {
            VStack(alignment: .leading, spacing: 8) {
                Text("Fix My Connection")
                    .font(.title2.weight(.semibold))
                    .onTapGesture(count: 3) {
                        state.developerModeEnabled = true
                    }
                Text(state.fieldMode?.mode == "macbook-network-rescue" ? "AgentLink diagnoses the connection, checks Clash/TUN as one possible cause, asks before writable repair, creates rollback checkpoints, and uses Terminal tickets when admin permission is needed." : "AgentLink diagnoses your network, explains the likely cause, asks before writable repair, creates rollback checkpoints, verifies after each step, and uses Terminal tickets when admin permission is needed.")
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                if state.safeModeActive {
                    Text("Package needs local repair or is missing required resources. Safe Mode limits the main action to diagnostics until package health is restored.")
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
                if state.incompleteJournalDetected {
                    Text("Previous repair did not finish. Review journal recovery before applying another repair.")
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
                HStack(spacing: 8) {
                    StatusBadge(text: "Package \(state.packageHealthLabel)", kind: packageHealthKind)
                    StatusBadge(text: state.packageType, kind: state.packageType == "Brain" ? .ok : .warn)
                    StatusBadge(text: state.recommendationLabel, kind: state.recommendationKind)
                    if state.incompleteJournalDetected {
                        StatusBadge(text: "Incomplete repair", kind: .warn)
                    }
                    if state.isRunning {
                        StatusBadge(text: "Running", kind: .warn)
                    }
                }
            }
            Spacer(minLength: 16)
            VStack(alignment: .trailing, spacing: 12) {
                Button {
                    if state.safeModeActive {
                        Task { await state.runDoctor() }
                    } else {
                        state.page = .guided
                        Task { await state.runGuidedRescue(allowRepair: false, target: state.fieldMode?.mode == "macbook-network-rescue" ? "network" : "auto") }
                    }
                } label: {
                    Label(state.safeModeActive ? "Diagnose Only" : "Fix My Connection", systemImage: state.safeModeActive ? "stethoscope" : "sparkles.rectangle.stack")
                        .font(.headline)
                        .frame(minWidth: 260)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isRunning)
                if state.isRunning {
                    Text("A rescue command is already running.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                if state.packageHealth?.status == "needs_repair" || state.packageHealth?.status == "broken" {
                    Button {
                        Task { await state.repairPackage() }
                    } label: {
                        Label("Fix Package", systemImage: "wrench.and.screwdriver")
                    }
                    .buttonStyle(.bordered)
                    .disabled(state.isRunning)
                }
                if state.incompleteJournalDetected {
                    Button {
                        state.page = .reports
                    } label: {
                        Label("Review Recovery", systemImage: "exclamationmark.arrow.triangle.2.circlepath")
                    }
                    .buttonStyle(.bordered)
                    .disabled(state.isRunning)
                }
                Button {
                    confirmSupportBundle = true
                } label: {
                    Label("Export Support Bundle", systemImage: "shippingbox")
                }
                .buttonStyle(.bordered)
                .disabled(state.isRunning)
                Button(state.developerModeEnabled ? "Developer Mode On" : "Developer Mode") {
                    state.page = .settings
                }
                .buttonStyle(.borderless)
            }
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 10))
    }

    private var restoreCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Connection Rescue").font(.headline)
            Text(state.versionText).foregroundStyle(.secondary)
            StatusBadge(text: state.selftestStatus, kind: state.selftestStatus == "OK" ? .ok : .warn)
            InfoRow(label: "Package health", value: state.packageHealthLabel)
            InfoRow(label: "Safe Mode", value: state.safeModeActive ? "On" : "Off")
            InfoRow(label: "Journal", value: state.journalRecovery?.status ?? "unknown")
            InfoRow(label: "Package", value: state.packageType)
            InfoRow(label: "Recommended", value: state.recommendationLabel)
            ForEach((state.doctor?.classifications ?? []).prefix(4), id: \.self) { item in
                StatusBadge(text: item, kind: item == "OK" ? .ok : .warn)
            }
            InfoRow(label: "Binary", value: state.client.binaryURL.path, monospaced: true)
        }
        .card()
    }

    private var reportsCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Reports & Rollback").font(.headline)
            InfoRow(label: "Dry-run", value: state.dryRun?.status ?? "None")
            InfoRow(label: "Repair", value: state.repair?.status ?? "None")
            InfoRow(label: "Session", value: state.repair?.sessionId ?? state.dryRun?.sessionId ?? "None", monospaced: true)
            Text("Rollback is available only after a snapshot-backed repair.")
                .font(.caption)
                .foregroundStyle(.secondary)
            Divider()
            let available = state.brainDoctor?.brainPackAvailable == true
            StatusBadge(text: available ? "Brain package available" : "Brain assets missing", kind: available ? .ok : .warn)
            InfoRow(label: "Model SHA", value: state.brainDoctor?.modelSha256OK == true ? "OK" : "Missing/failed")
            InfoRow(label: "Runtime", value: state.brainDoctor?.runtimeExecutable == true ? "Executable" : "Missing")
        }
        .card()
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
