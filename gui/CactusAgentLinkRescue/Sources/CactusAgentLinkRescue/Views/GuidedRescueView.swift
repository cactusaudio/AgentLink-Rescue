import AppKit
import SwiftUI

struct GuidedRescueView: View {
    @EnvironmentObject var state: AppState
    @State private var confirmApply = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                if state.fieldMode?.mode == "macbook-network-rescue" {
                    fieldLanding
                }
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

    private var fieldLanding: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Clash/TUN 网络恢复")
                        .font(.title2.weight(.semibold))
                    Text("Use this if Wi-Fi/Ethernet connects but the internet breaks after Clash Verge TUN mode. AgentLink prefers targeted TUN repair before broad standard reset.")
                        .foregroundStyle(.secondary)
                }
                Spacer()
                StatusBadge(text: state.fieldReport?.recommendedAction ?? "field mode", kind: fieldKind(state.fieldReport?.recommendedAction))
            }
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 190), spacing: 10)], spacing: 10) {
                fieldStatus("Proxy", state.fieldReport?.diagnosis?.proxyDirty == true ? "Dirty" : "Clean/unknown", state.fieldReport?.diagnosis?.proxyDirty == true ? .warn : .ok)
                fieldStatus("Clash residue", state.fieldReport?.diagnosis?.clashResidueDetected == true ? "Detected" : "Not detected", state.fieldReport?.diagnosis?.clashResidueDetected == true ? .warn : .neutral)
                fieldStatus("TUN/Extension", state.fieldReport?.diagnosis?.clashTunDetected == true ? "Clash TUN" : (state.fieldReport?.diagnosis?.networkExtensionSuspected == true ? "Suspected" : "Not suspected"), (state.fieldReport?.diagnosis?.clashTunDetected == true || state.fieldReport?.diagnosis?.networkExtensionSuspected == true) ? .warn : .neutral)
                fieldStatus("Brain", state.brainDoctor?.brainPackAvailable == true ? "Available" : "Missing", state.brainDoctor?.brainPackAvailable == true ? .ok : .warn)
            }
            HStack(spacing: 10) {
                Button {
                    Task { await state.analyzeFieldNetwork() }
                } label: {
                    Label("Analyze Network", systemImage: "network")
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isRunning)
                Button("Copy TUN Repair Command") {
                    copy(state.fieldRescueCommand(level: "tun"))
                }
                Button("Create Terminal Ticket") {
                    Task { await state.runGuidedRescue(allowRepair: true, target: "network") }
                }
                Button("Open Clash Verge Rev Installer") {
                    Task { await state.runInstallerOpen("clash-verge-rev") }
                }
                .disabled(state.isRunning)
            }
            DisclosureGroup("Advanced field commands") {
                VStack(alignment: .leading, spacing: 10) {
                    Text("GUI mode does not run sudo or collect passwords. Copy these commands into Terminal.")
                        .foregroundStyle(.secondary)
                    CommandPreview(title: "Targeted Clash/TUN Repair", command: state.fieldRescueCommand(level: "tun"))
                    CommandPreview(title: "Verify Network", command: "cd \(shellQuote(state.client.packageRoot.path))\n./bin/agentlink verify network --json")
                    CommandPreview(title: "Restart Gate Verify", command: "cd \(shellQuote(state.client.packageRoot.path))\n./bin/agentlink restart-gate verify --json")
                    CommandPreview(title: "Standard System Reset (fallback)", command: state.fieldRescueCommand(level: "standard-system-reset"))
                    CommandPreview(title: "Deep Repair (final resort)", command: state.fieldRescueCommand(level: "deep"))
                    CommandPreview(title: "Support Bundle", command: "cd \(shellQuote(state.client.packageRoot.path))\n./bin/agentlink support bundle")
                }
                .padding(.top, 8)
            }
        }
        .card()
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
        case "ok", "healthy", "fixed", "ready", "ready to apply", "dry-run complete", "repaired", "planned":
            return .ok
        case "running", "warn", "needs review", "no local candidate", "manual action required", "no safe action", "rolled back", "terminal ticket ready", "restart required", "rolled back after worsening", "ticket_created", "restart_required", "rolled_back_after_worsening":
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

    private func fieldStatus(_ title: String, _ value: String, _ kind: StatusBadge.Kind) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title).font(.caption).foregroundStyle(.secondary)
            StatusBadge(text: value, kind: kind)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(nsColor: .textBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
    }

    private func fieldKind(_ action: String?) -> StatusBadge.Kind {
        switch action {
        case "safe", "standard":
            return .warn
        case "deep":
            return .fail
        case "manual":
            return .neutral
        default:
            return .neutral
        }
    }

    private func copy(_ text: String) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(text, forType: .string)
    }
}
