import SwiftUI

struct ReportsView: View {
    @EnvironmentObject var state: AppState
    @State private var selectedReport = "Human"

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    ActionButton(title: "Refresh Reports", systemImage: "arrow.clockwise", disabled: state.isRunning) { Task { await state.loadReports() } }
                    Button("Export Human Report") { export(state.humanReport, name: "AgentLink-Human-Report.txt") }
                    Button("Export Codex Dispatch") { export(state.codexDispatch, name: "AgentLink-Codex-Dispatch.txt") }
                }
                VStack(alignment: .leading, spacing: 8) {
                    Text("Latest Session").font(.headline)
                    InfoRow(label: "Loaded", value: state.reportsLoadedAt.map(Self.timestamp.string(from:)) ?? "Not loaded")
                    if let error = state.reportLoadError {
                        StatusBadge(text: "No latest report", kind: .warn)
                        Text(error).foregroundStyle(.secondary)
                    } else {
                        StatusBadge(text: "Latest report loaded", kind: .ok)
                    }
                }
                .card()
                Picker("Report", selection: $selectedReport) {
                    Text("Human Report").tag("Human")
                    Text("Codex Dispatch").tag("Codex")
                }
                .pickerStyle(.segmented)
                if selectedReport == "Human" {
                    OutputCard(title: "Human Report", text: state.humanReport, placeholder: "No human report found yet.")
                } else {
                    OutputCard(title: "Codex Dispatch", text: state.codexDispatch, placeholder: "No Codex dispatch found yet.")
                }
            }
            .padding()
        }
        .task {
            await state.loadReports()
        }
    }

    private func export(_ text: String, name: String) {
        let url = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Desktop").appendingPathComponent(name)
        try? text.write(to: url, atomically: true, encoding: .utf8)
    }

    private static let timestamp: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .short
        formatter.timeStyle = .medium
        return formatter
    }()
}
