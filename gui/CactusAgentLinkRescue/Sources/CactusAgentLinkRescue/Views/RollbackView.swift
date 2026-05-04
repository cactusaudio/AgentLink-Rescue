import SwiftUI

struct RollbackView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                ActionButton(title: "Rollback Last User Snapshot", systemImage: "arrow.uturn.backward.circle", disabled: state.isRunning || !state.rollbackAvailable) {
                    Task { await state.rollbackLast() }
                }
                if state.rollbackAvailable {
                    Text("User-level restore uses AgentLink directly. If the restore point touched system paths, copy the sudo command instead; GUI v0.5.0 does not run sudo.")
                        .foregroundStyle(.secondary)
                } else {
                    Text("No rollbackable user-level session is currently detected. Refresh Reports or use the Expert Console command only when you know the restore point.")
                        .foregroundStyle(.secondary)
                }
                ActionButton(title: "Refresh Sessions", systemImage: "arrow.clockwise", disabled: state.isRunning) {
                    Task { await state.loadReports() }
                }
                .frame(width: 190)
                Text("System-level rollback requires Terminal. The GUI never collects passwords and never runs sudo.")
                    .foregroundStyle(.secondary)
                CommandPreview(title: "System rollback command", command: state.client.copyableTerminalCommand(["rollback", "--last"], sudo: true))
                OutputCard(title: "Rollback Output", text: state.rollbackOutput, placeholder: "No rollback has run yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }
}
