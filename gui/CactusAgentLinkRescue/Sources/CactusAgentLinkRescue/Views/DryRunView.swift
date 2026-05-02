import SwiftUI

struct DryRunView: View {
    @EnvironmentObject var state: AppState
    let targets = ["path", "proxy", "codex", "keys", "network"]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    Picker("Target", selection: $state.target) {
                        ForEach(targets, id: \.self) { Text($0).tag($0) }
                    }
                    .pickerStyle(.segmented)
                    ActionButton(title: "Run Dry-Run", systemImage: "play.rectangle", disabled: state.isRunning) { Task { await state.runDryRun() } }
                }
                VStack(alignment: .leading, spacing: 8) {
                    Text("Dry-Run Result").font(.headline)
                    Text("Status: \(state.dryRun?.status ?? "none")")
                    Text("Target: \(state.dryRun?.target ?? state.target)")
                    Text("Recipe: \(state.dryRun?.plannerDecision?.selectedRecipe?.id ?? "none")")
                    Text("Risk: \(state.dryRun?.plannerDecision?.risk ?? "unknown")")
                    StatusBadge(text: state.executeEnabled ? "Execute enabled for this target" : "Execute disabled until successful dry-run", kind: state.executeEnabled ? .ok : .warn)
                }
                .card()
                OutputCard(title: "Raw Dry-Run Output", text: state.latestResult?.combinedOutput ?? "")
            }
            .padding()
        }
    }
}
