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
                dryRunCard
                plannedActionsCard
                OutputCard(title: "Raw Dry-Run Output", text: state.dryRunResult?.combinedOutput ?? "", placeholder: "No dry-run has been run yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }

    private var dryRunCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Dry-Run Result").font(.headline)
            if let dryRun = state.dryRun {
                StatusBadge(text: "No changes made", kind: .ok)
                InfoRow(label: "Status", value: dryRun.status ?? "none")
                InfoRow(label: "Target", value: dryRun.target ?? state.target)
                InfoRow(label: "Recipe", value: dryRun.plannerDecision?.selectedRecipe?.id ?? dryRun.recipeResult?.recipeId ?? "none", monospaced: true)
                HStack {
                    Text("Risk").foregroundStyle(.secondary).frame(width: 120, alignment: .leading)
                    StatusBadge(text: dryRun.plannerDecision?.risk ?? "unknown", kind: riskKind(dryRun.plannerDecision?.risk))
                }
            } else {
                StatusBadge(text: state.isRunning ? "Running" : "Not run", kind: .neutral)
                Text("Dry-run asks the deterministic runner what it would do. It must complete before Execute can be enabled.")
                    .foregroundStyle(.secondary)
            }
            StatusBadge(text: state.executeEnabled ? "Execute enabled for this target" : "Execute disabled until matching dry-run succeeds", kind: state.executeEnabled ? .ok : .warn)
        }
        .card()
    }

    private var plannedActionsCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Planned Actions").font(.headline)
            let actions = state.dryRun?.recipeResult?.plannedActions ?? []
            if actions.isEmpty {
                Text("No planned actions yet.")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(Array(actions.enumerated()), id: \.offset) { _, action in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(action.description ?? action.id ?? "planned action")
                            .font(.subheadline.weight(.semibold))
                        if let path = action.path {
                            Text(path)
                                .font(.system(.caption, design: .monospaced))
                                .foregroundStyle(.secondary)
                                .lineLimit(2)
                                .truncationMode(.middle)
                                .textSelection(.enabled)
                        }
                    }
                    .padding(.vertical, 4)
                }
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
