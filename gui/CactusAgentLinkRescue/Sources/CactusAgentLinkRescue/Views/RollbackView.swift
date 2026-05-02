import SwiftUI

struct RollbackView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                ActionButton(title: "Rollback Last User Snapshot", systemImage: "arrow.uturn.backward.circle", disabled: state.isRunning) {
                    Task { await state.rollbackLast() }
                }
                CommandPreview(title: "System rollback command", command: state.client.copyableTerminalCommand(["rollback", "--last"], sudo: true))
                OutputCard(title: "Rollback Output", text: state.rollbackOutput)
            }
            .padding()
        }
    }
}
