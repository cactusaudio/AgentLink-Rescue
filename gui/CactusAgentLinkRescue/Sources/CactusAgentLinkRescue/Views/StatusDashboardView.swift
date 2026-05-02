import SwiftUI

struct StatusDashboardView: View {
    @EnvironmentObject var state: AppState
    @State private var allowReversibleRepairs = false
    @State private var confirmReversibleRepair = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                guidedHero
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 220), spacing: 12)], spacing: 12) {
                    summaryPill(title: "AgentLink Core", value: state.selftestStatus, kind: state.selftestStatus == "OK" ? .ok : .warn)
                    summaryPill(title: "Brain Pack", value: state.brainDoctor?.brainPackAvailable == true ? "OK" : "Missing", kind: state.brainDoctor?.brainPackAvailable == true ? .ok : .warn)
                    summaryPill(title: "Network", value: state.doctor?.classifications?.contains("OK") == true ? "OK" : state.recommendationLabel, kind: state.recommendationKind)
                    summaryPill(title: "Last Session", value: state.repair?.status ?? state.dryRun?.status ?? "None", kind: state.dryRun?.status == "dry-run" || state.repair?.status == "success" ? .ok : .neutral)
                }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 320), spacing: 12)], spacing: 12) {
                    restoreCard
                    reportsCard
                }
                OutputCard(title: "Advanced / Raw Diagnostics", text: state.latestResult?.combinedOutput ?? "", placeholder: "Run Doctor or Brain Plan to see raw output.", collapsedByDefault: true)
            }
            .padding()
        }
    }

    private var guidedHero: some View {
        HStack(alignment: .center, spacing: 18) {
            VStack(alignment: .leading, spacing: 8) {
                Text("Restore Agent Link")
                    .font(.title2.weight(.semibold))
                    .onTapGesture(count: 3) {
                        state.brainSandboxPresented = true
                    }
                Text("Detect -> plan -> dry-run -> verify. Optional reversible repair with confirmation.")
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
                HStack(spacing: 8) {
                    StatusBadge(text: state.packageType, kind: state.packageType == "Brain" ? .ok : .warn)
                    StatusBadge(text: state.recommendationLabel, kind: state.recommendationKind)
                    if state.isRunning {
                        StatusBadge(text: "Running", kind: .warn)
                    }
                }
            }
            Spacer(minLength: 16)
            VStack(alignment: .trailing, spacing: 10) {
                Toggle("Allow reversible repairs", isOn: $allowReversibleRepairs)
                    .toggleStyle(.checkbox)
                Button {
                    state.page = .guided
                    if allowReversibleRepairs {
                        confirmReversibleRepair = true
                    } else {
                        Task { await state.runGuidedRescue(allowRepair: false) }
                    }
                } label: {
                    Label("Restore Agent Link", systemImage: "sparkles.rectangle.stack")
                        .font(.headline)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 8)
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isRunning)
                Button("Analyze Only") {
                    state.page = .guided
                    Task { await state.runGuidedRescue(allowRepair: false) }
                }
                .buttonStyle(.borderless)
                Button("Open Expert Console") {
                    state.page = .doctor
                }
                .buttonStyle(.borderless)
                Button("Installer Center") {
                    state.page = .installer
                }
                .buttonStyle(.borderless)
            }
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 10))
        .alert("Allow reversible repairs?", isPresented: $confirmReversibleRepair) {
            Button("Cancel", role: .cancel) {}
            Button("Run Guided Repair") {
                state.page = .guided
                Task { await state.runGuidedRescue(allowRepair: true) }
            }
        } message: {
            Text("AgentLink may apply reversible recipe repairs only. It will create snapshots and verify results. It will not run sudo or network deep rescue.")
        }
    }

    private var restoreCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Restore Agent Link").font(.headline)
            Text(state.versionText).foregroundStyle(.secondary)
            StatusBadge(text: state.selftestStatus, kind: state.selftestStatus == "OK" ? .ok : .warn)
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
