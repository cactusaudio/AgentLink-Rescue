import SwiftUI

struct GuidedRescueView: View {
    @EnvironmentObject var state: AppState
    @State private var confirmApply = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                hero
                stepTimeline
                if state.guidedCanApply || state.executeEnabled {
                    applyCard
                }
                OutputCard(
                    title: "Guided Rescue Output",
                    text: state.guidedResult?.combinedOutput ?? "",
                    placeholder: "Run Guided Rescue to see the latest command output.",
                    collapsedByDefault: true
                )
            }
            .padding()
        }
    }

    private var hero: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Guided Rescue")
                        .font(.title2.weight(.semibold))
                    Text("AgentLink CLI runs the guided kernel: detect, plan, dry-run, snapshot, execute reversible recipes when approved, verify, and rollback if needed.")
                        .foregroundStyle(.secondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                Spacer()
                StatusBadge(text: state.guidedStatus, kind: guidedKind(state.guidedStatus))
            }

            HStack(spacing: 10) {
                Button {
                    Task { await state.runGuidedRescue(allowRepair: false) }
                } label: {
                    Label("Analyze Only", systemImage: "sparkles.rectangle.stack")
                        .font(.headline)
                        .padding(.vertical, 6)
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isRunning)

                Button {
                    confirmApply = true
                } label: {
                    Label("Allow Reversible Repairs", systemImage: "checkmark.shield")
                        .padding(.vertical, 6)
                }
                .buttonStyle(.bordered)
                .disabled(state.isRunning)
                .confirmationDialog("Allow reversible repairs?", isPresented: $confirmApply) {
                    Button("Run with reversible user-level repairs", role: .destructive) {
                        Task { await state.runGuidedRescue(allowRepair: true) }
                    }
                    Button("Cancel", role: .cancel) {}
                } message: {
                    Text("AgentLink will still use dry-run first, create snapshots before writable recipes, and will not run sudo or network rescue from the GUI.")
                }
            }

            Text(state.guidedSummary)
                .foregroundStyle(.secondary)
        }
        .card()
    }

    private var stepTimeline: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Progress").font(.headline)
            if state.guidedSteps.isEmpty {
                Text("No guided rescue has been run yet.")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(Array(state.guidedSteps.enumerated()), id: \.element.id) { index, step in
                    HStack(alignment: .top, spacing: 12) {
                        Text("\(index + 1)")
                            .font(.caption.weight(.bold))
                            .frame(width: 24, height: 24)
                            .background(statusColor(step.status).opacity(0.18), in: Circle())
                            .foregroundStyle(statusColor(step.status))
                        VStack(alignment: .leading, spacing: 4) {
                            HStack {
                                Text(step.title).font(.subheadline.weight(.semibold))
                                StatusBadge(text: step.status, kind: guidedKind(step.status))
                            }
                            Text(step.detail)
                                .foregroundStyle(.secondary)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .card()
    }

    private var applyCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Ready State").font(.headline)
            InfoRow(label: "Target", value: state.guidedLastTarget ?? state.target)
            InfoRow(label: "Recipe", value: state.guidedLastRecipe ?? state.dryRun?.plannerDecision?.selectedRecipe?.id ?? "none", monospaced: true)
            Text("Execution stays inside the AgentLink CLI kernel. The GUI only starts the approved guided command and displays its report.")
                .foregroundStyle(.secondary)
            HStack {
                Button {
                    confirmApply = true
                } label: {
                    Label("Run Guided Repair", systemImage: "checkmark.shield")
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isRunning)
                Button {
                    Task { await state.runGuidedRescue(allowRepair: false) }
                } label: {
                    Label("Rerun Analyze-Only", systemImage: "arrow.clockwise")
                }
                .buttonStyle(.bordered)
                .disabled(state.isRunning)
            }
        }
        .card()
    }

    private func guidedKind(_ status: String) -> StatusBadge.Kind {
        switch status.lowercased() {
        case "ok", "healthy", "fixed", "ready", "ready to apply", "dry-run complete", "repaired":
            return .ok
        case "running", "warn", "needs review", "no local candidate", "manual action required", "no safe action", "rolled back":
            return .warn
        case "failed":
            return .fail
        default:
            return .neutral
        }
    }

    private func statusColor(_ status: String) -> Color {
        switch guidedKind(status) {
        case .ok: return .green
        case .warn: return .orange
        case .fail: return .red
        case .neutral: return .secondary
        }
    }
}
