import SwiftUI

struct PlanView: View {
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
                    ActionButton(title: "Run Brain Plan", systemImage: "list.bullet.clipboard", disabled: state.isRunning) { Task { await state.runPlan() } }
                }
                Text("Qwen proposes a recipe. AgentLink validates it before execution. Qwen does not run commands.")
                    .foregroundStyle(.secondary)
                decisionCard
                OutputCard(title: "Raw Plan JSON", text: state.latestResult?.stdout ?? "")
            }
            .padding()
        }
    }

    private var decisionCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Planner Decision").font(.headline)
            Text("Intent: \(state.plan?.decision?.intent ?? "none")")
            Text("Recipe: \(state.plan?.decision?.selectedRecipe?.id ?? "none")")
            Text("Risk: \(state.plan?.decision?.risk ?? "unknown")")
            Text(String(format: "Confidence: %.2f", state.plan?.decision?.confidence ?? 0))
            Text("Verifiers: \((state.plan?.decision?.expectedVerifiers ?? []).joined(separator: ", "))")
            Text(state.plan?.decision?.explanationForUser ?? "")
                .foregroundStyle(.secondary)
            if let errors = state.plan?.validationErrors, !errors.isEmpty {
                Text(errors.joined(separator: "\n")).foregroundStyle(.red)
            }
        }
        .card()
    }
}
