import SwiftUI

struct RollbackView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                ActionButton(title: "Rollback Last User Snapshot", systemImage: "arrow.uturn.backward.circle", disabled: state.isRunning) {
                    Task { await state.rollbackLast() }
                }
                Text("User-level restore uses AgentLink directly. If the restore point touched system paths, copy the sudo command instead; GUI v0.4.1 does not run sudo.")
                    .foregroundStyle(.secondary)
                CommandPreview(title: "System rollback command", command: state.client.copyableTerminalCommand(["rollback", "--last"], sudo: true))
                OutputCard(title: "Rollback Output", text: state.rollbackOutput, placeholder: "No rollback has run yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }
}
