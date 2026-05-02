import SwiftUI

struct ReportsView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    ActionButton(title: "Load Latest Reports", systemImage: "doc.text", disabled: state.isRunning) { Task { await state.loadReports() } }
                    Button("Export Human Report") { export(state.humanReport, name: "AgentLink-Human-Report.txt") }
                    Button("Export Codex Dispatch") { export(state.codexDispatch, name: "AgentLink-Codex-Dispatch.txt") }
                }
                OutputCard(title: "Human Report", text: state.humanReport)
                OutputCard(title: "Codex Dispatch", text: state.codexDispatch)
            }
            .padding()
        }
    }

    private func export(_ text: String, name: String) {
        let url = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Desktop").appendingPathComponent(name)
        try? text.write(to: url, atomically: true, encoding: .utf8)
    }
}
