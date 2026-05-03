import SwiftUI

struct ReadinessCenterView: View {
    @EnvironmentObject var state: AppState
    @State private var confirmRestoreLastGood = false
    @State private var confirmSupportBundle = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                header
                HStack {
                    ActionButton(title: "Run Readiness Check", systemImage: "checklist.checked", disabled: state.isRunning) {
                        Task { await state.runReadiness() }
                    }
                    Button("Dev Essentials") {
                        Task { await state.runDevEssentials() }
                    }
                    .disabled(state.isRunning)
                    Button("Save Last-Good") {
                        Task { await state.saveLastGood() }
                    }
                    .disabled(state.isRunning)
                    Button("Support Bundle") {
                        confirmSupportBundle = true
                    }
                    .disabled(state.isRunning)
                }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 320), spacing: 12)], spacing: 12) {
                    readinessCard
                    lastGoodCard
                    supportCard
                    devCard
                }
                OutputCard(title: "Readiness Output", text: state.readinessResult?.combinedOutput ?? state.lastGoodResult?.combinedOutput ?? state.supportBundleResult?.combinedOutput ?? "", placeholder: "Run readiness, last-good, support bundle, or dev essentials to see output.", collapsedByDefault: true)
            }
            .padding()
        }
        .task {
            if state.readiness == nil {
                await state.runReadiness()
            }
        }
        .alert("Restore last-good profile?", isPresented: $confirmRestoreLastGood) {
            Button("Cancel", role: .cancel) {}
            Button("Restore") {
                Task { await state.restoreLastGood() }
            }
        } message: {
            Text("AgentLink will restore saved AI-tool config files only. It creates a user snapshot first and does not run sudo.")
        }
        .alert("Export support bundle?", isPresented: $confirmSupportBundle) {
            Button("Cancel", role: .cancel) {}
            Button("Export") {
                Task { await state.createSupportBundle() }
            }
        } message: {
            Text("This support bundle contains redacted local diagnostics, including tool presence, local paths, network/proxy status, readiness reports, and latest session metadata. It does not include private keys, browser cookies, shell history, Wi-Fi passwords, or full API keys.")
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Readiness Center")
                .font(.title2.weight(.semibold))
            Text("Checks whether this Mac is ready to recover the AI path offline: network, proxy, Brain, installers, secrets, and base tools. It does not index repos or manage project dependencies.")
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    private var readinessCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Offline Readiness").font(.headline)
                Spacer()
                StatusBadge(text: readinessStatusLabel, kind: readinessStatusKind)
            }
            ForEach(state.readiness?.checks ?? []) { check in
                VStack(alignment: .leading, spacing: 4) {
                    HStack {
                        StatusBadge(text: check.status ?? "unknown", kind: kind(check.status))
                        Text(check.title ?? check.id)
                            .font(.subheadline.weight(.medium))
                    }
                    Text(check.evidence ?? "")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let action = check.action {
                        Text(action)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                Divider()
            }
        }
        .card()
    }

    private var lastGoodCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Last-Good Profiles").font(.headline)
            Text("Save and restore known-good AI-tool configuration. This is recovery state, not project backup.")
                .foregroundStyle(.secondary)
            InfoRow(label: "Latest action", value: state.lastGood?.action ?? "None")
            InfoRow(label: "Status", value: state.lastGood?.status ?? "Unknown")
            InfoRow(label: "Profile", value: state.lastGood?.profileID ?? "None", monospaced: true)
            HStack {
                Button("List") { Task { await state.listLastGood() } }
                    .disabled(state.isRunning)
                Button("Save") { Task { await state.saveLastGood() } }
                    .disabled(state.isRunning)
                Button("Restore Last") { confirmRestoreLastGood = true }
                    .disabled(state.isRunning)
            }
        }
        .card()
    }

    private var supportCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Support Bundle").font(.headline)
            Text("Exports redacted doctor, Brain, installer, readiness, dev, and latest session reports for Codex or a human helper.")
                .foregroundStyle(.secondary)
            InfoRow(label: "Status", value: state.supportBundle?.status ?? "Not created")
            InfoRow(label: "Path", value: state.supportBundle?.bundlePath ?? "None", monospaced: true)
            Button("Create Support Bundle") {
                confirmSupportBundle = true
            }
            .disabled(state.isRunning)
        }
        .card()
    }

    private var devCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Dev Essentials Doctor").font(.headline)
            Text("Only checks base tools needed to restore AI CLIs: Git, curl, Node/npm, Homebrew, and Xcode CLT.")
                .foregroundStyle(.secondary)
            StatusBadge(text: state.devEssentials?.status ?? "not run", kind: state.devEssentials?.status == "ok" ? .ok : .warn)
            ForEach(state.devEssentials?.tools ?? []) { tool in
                HStack {
                    Text(tool.name ?? tool.id)
                    Spacer()
                    StatusBadge(text: tool.status ?? "unknown", kind: kind(tool.status))
                }
            }
        }
        .card()
    }

    private var readinessStatusLabel: String {
        switch state.readiness?.status {
        case "ready": return "Ready"
        case "ready_with_warnings": return "Warnings"
        case "not_ready": return "Not ready"
        default: return "Unknown"
        }
    }

    private var readinessStatusKind: StatusBadge.Kind {
        switch state.readiness?.status {
        case "ready": return .ok
        case "ready_with_warnings": return .warn
        case "not_ready": return .fail
        default: return .neutral
        }
    }

    private func kind(_ status: String?) -> StatusBadge.Kind {
        switch status {
        case "ok", "installed", "pass": return .ok
        case "warn", "missing": return .warn
        case "fail", "failed": return .fail
        default: return .neutral
        }
    }
}
