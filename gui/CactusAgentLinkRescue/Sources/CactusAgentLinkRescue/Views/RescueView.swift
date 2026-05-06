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
                    Button("Execute through AgentLink runner", role: .destructive) { Task { await state.executeRepair() } }
                    Button("Cancel", role: .cancel) {}
                } message: {
                    Text("This uses the same target and recipe validated by the latest dry-run. AgentLink creates a snapshot before mutation.")
                }
                VStack(alignment: .leading, spacing: 8) {
                    Text("Repair Result").font(.headline)
                    Text("Status: \(state.repair?.status ?? "none")")
                    Text("Session: \(state.repair?.sessionId ?? "none")")
                    Text("Human report: \(state.repair?.humanReportPath ?? "none")").font(.caption).textSelection(.enabled)
                    StatusBadge(text: state.executeEnabled ? "Dry-run matched; execute available" : "Run a matching dry-run before execute", kind: state.executeEnabled ? .ok : .warn)
                }
                .card()
                networkAdvanced
                OutputCard(title: "Live Output", text: state.repairResult?.combinedOutput ?? state.dryRunResult?.combinedOutput ?? "", placeholder: "No repair command output yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }

    private var networkAdvanced: some View {
        DisclosureGroup("Network Rescue Advanced") {
            VStack(alignment: .leading, spacing: 12) {
                Text("Network rescue commands may require sudo and system network changes. GUI v0.5.1 does not collect passwords and does not run sudo commands. Copy the command and run it in Terminal. Use Targeted Clash/TUN before broad system reset when Clash, Mihomo, or stale utun signatures are present.")
                    .foregroundStyle(.secondary)
                CommandPreview(title: "Safe Rescue", command: state.sudoRescueCommand(level: "safe"))
                CommandPreview(title: "Targeted Clash/TUN Rescue", command: state.sudoRescueCommand(level: "tun"))
                CommandPreview(title: "Standard Rescue", command: state.sudoRescueCommand(level: "standard"))
                CommandPreview(title: "Clean Network Baseline Reset (last resort)", command: state.sudoRescueCommand(level: "clean-baseline"))
                CommandPreview(title: "Standard System Reset", command: state.sudoRescueCommand(level: "standard-system-reset"))
                CommandPreview(title: "Deep Rescue", command: state.sudoDeepCommand())
                CommandPreview(title: "Fallback rescue.sh", command: state.client.fallbackRescueCommand(level: "standard"))
            }
            .padding(.top, 8)
        }
        .card()
    }
}
