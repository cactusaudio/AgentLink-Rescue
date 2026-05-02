import SwiftUI

struct StatusDashboardView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    ActionButton(title: "Run Doctor", systemImage: "stethoscope", disabled: state.isRunning) { Task { await state.runDoctor() } }
                    ActionButton(title: "Brain Doctor", systemImage: "brain.head.profile", disabled: state.isRunning) { Task { await state.runBrainDoctor() } }
                    ActionButton(title: "Auto Repair Dry-Run", systemImage: "play.rectangle", disabled: state.isRunning) { Task { await state.runDryRun() } }
                    Button("Export Report") { export(state.humanReport, name: "AgentLink-Human-Report.txt") }
                }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 260), spacing: 12)], spacing: 12) {
                    coreCard
                    networkCard
                    brainCard
                    sessionCard
                }
                OutputCard(title: "Latest Command", text: state.latestResult?.combinedOutput ?? "")
            }
            .padding()
        }
    }

    private var coreCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Core Status").font(.headline)
            Text(state.versionText)
            StatusBadge(text: state.selftestStatus, kind: state.selftestStatus == "OK" ? .ok : .warn)
            Text("Package: \(state.packageType)")
            Text(state.client.binaryURL.path).font(.caption).foregroundStyle(.secondary).lineLimit(2)
        }
        .card()
    }

    private var networkCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Network / AgentLink").font(.headline)
            Text("Recommended: \(state.doctor?.recommendedRepairLevel ?? "unknown")")
            ForEach((state.doctor?.classifications ?? []).prefix(4), id: \.self) { item in
                StatusBadge(text: item, kind: item == "OK" ? .ok : .warn)
            }
            if let warnings = state.doctor?.warnings, !warnings.isEmpty {
                Text(warnings.joined(separator: "\n")).font(.caption).foregroundStyle(.secondary)
            }
        }
        .card()
    }

    private var brainCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Brain Status").font(.headline)
            let available = state.brainDoctor?.brainPackAvailable == true
            StatusBadge(text: available ? "Brain package available" : "Brain assets missing", kind: available ? .ok : .warn)
            Text("Model SHA OK: \(state.brainDoctor?.modelSha256OK == true ? "yes" : "no")")
            Text("Runtime executable: \(state.brainDoctor?.runtimeExecutable == true ? "yes" : "no")")
            if let fetch = state.brainDoctor?.fetchCommands?.joined(separator: "\n") {
                Text(fetch).font(.caption).textSelection(.enabled)
            }
        }
        .card()
    }

    private var sessionCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Last Repair Session").font(.headline)
            Text("Dry-run: \(state.dryRun?.status ?? "none")")
            Text("Repair: \(state.repair?.status ?? "none")")
            Text("Session: \(state.repair?.sessionId ?? state.dryRun?.sessionId ?? "none")")
            Text("Rollback available after repair if snapshot exists.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .card()
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
