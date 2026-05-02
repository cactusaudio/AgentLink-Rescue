import SwiftUI

struct RescueView: View {
    @EnvironmentObject var state: AppState
    @State private var confirmExecute = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("You are about to run a reversible recipe through AgentLink's runner. A snapshot will be created before changes. Verifiers decide success. If verification fails, rollback is available.")
                    .foregroundStyle(.secondary)
                HStack {
                    ActionButton(title: "Execute Selected Repair", systemImage: "wrench.and.screwdriver", disabled: !state.executeEnabled || state.isRunning) {
                        confirmExecute = true
                    }
                    ActionButton(title: "Refresh Reports", systemImage: "doc.text", disabled: state.isRunning) { Task { await state.loadReports() } }
                }
                .confirmationDialog("Execute reversible repair?", isPresented: $confirmExecute) {
                    Button("Execute", role: .destructive) { Task { await state.executeRepair() } }
                    Button("Cancel", role: .cancel) {}
                }
                VStack(alignment: .leading, spacing: 8) {
                    Text("Repair Result").font(.headline)
                    Text("Status: \(state.repair?.status ?? "none")")
                    Text("Session: \(state.repair?.sessionId ?? "none")")
                    Text("Human report: \(state.repair?.humanReportPath ?? "none")").font(.caption).textSelection(.enabled)
                }
                .card()
                networkAdvanced
                OutputCard(title: "Live Output", text: state.latestResult?.combinedOutput ?? "")
            }
            .padding()
        }
    }

    private var networkAdvanced: some View {
        DisclosureGroup("Network Rescue Advanced") {
            VStack(alignment: .leading, spacing: 12) {
                Text("Network rescue safe/standard/deep may require sudo and system network changes. GUI v0.4.0 does not collect passwords. Copy the command and run it in Terminal.")
                    .foregroundStyle(.secondary)
                CommandPreview(title: "Safe Rescue", command: state.sudoRescueCommand(level: "safe"))
                CommandPreview(title: "Standard Rescue", command: state.sudoRescueCommand(level: "standard"))
                CommandPreview(title: "Deep Rescue", command: state.sudoDeepCommand())
                CommandPreview(title: "Fallback rescue.sh", command: state.client.fallbackRescueCommand(level: "standard"))
            }
            .padding(.top, 8)
        }
        .card()
    }
}
