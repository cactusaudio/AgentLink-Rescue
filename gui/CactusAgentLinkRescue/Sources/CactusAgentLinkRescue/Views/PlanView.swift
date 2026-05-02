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
                Text("Gemma proposes a recipe. AgentLink validates it before execution. Gemma does not run commands.")
                    .foregroundStyle(.secondary)
                decisionCard
                OutputCard(title: "Raw Plan JSON", text: state.planResult?.stdout ?? "", placeholder: "No plan has been run yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }

    private var decisionCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Planner Decision").font(.headline)
            if let decision = state.plan?.decision {
                InfoRow(label: "Intent", value: decision.intent ?? "none")
                InfoRow(label: "Recipe", value: decision.selectedRecipe?.id ?? "none", monospaced: true)
                HStack {
                    Text("Risk").foregroundStyle(.secondary).frame(width: 120, alignment: .leading)
                    StatusBadge(text: decision.risk ?? "unknown", kind: riskKind(decision.risk))
                }
                InfoRow(label: "Confidence", value: String(format: "%.2f", decision.confidence ?? 0))
                if let verifiers = decision.expectedVerifiers, !verifiers.isEmpty {
                    Text("Expected verifiers").foregroundStyle(.secondary)
                    FlowText(items: verifiers)
                }
                Text(decision.explanationForUser ?? "")
                    .foregroundStyle(.secondary)
            } else {
                StatusBadge(text: state.isRunning ? "Running" : "Not run", kind: .neutral)
                Text("Choose a target and run Brain Plan. The result is validated before any dry-run or execution.")
                    .foregroundStyle(.secondary)
            }
            if let errors = state.plan?.validationErrors, !errors.isEmpty {
                Text(errors.joined(separator: "\n")).foregroundStyle(.red)
            }
        }
        .card()
    }

    private func riskKind(_ risk: String?) -> StatusBadge.Kind {
        switch risk {
        case "read_only", "safe_patch", "reversible_patch": return .ok
        case "network_action": return .warn
        case "privileged_action", "destructive_action": return .fail
        default: return .neutral
        }
    }
}

private struct FlowText: View {
    let items: [String]

    var body: some View {
        Text(items.joined(separator: ", "))
            .font(.caption)
            .foregroundStyle(.secondary)
            .textSelection(.enabled)
    }
}
