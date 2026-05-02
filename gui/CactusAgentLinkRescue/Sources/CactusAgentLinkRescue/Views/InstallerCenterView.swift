import SwiftUI

struct InstallerCenterView: View {
    @EnvironmentObject var state: AppState
    @State private var pendingInstallID: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                header
                HStack {
                    ActionButton(title: "Refresh Installer Doctor", systemImage: "arrow.clockwise", disabled: state.isRunning) {
                        Task { await state.runInstallerDoctor() }
                    }
                }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 340), spacing: 12)], spacing: 12) {
                    installerCard(id: "clash-verge-rev", title: "Proxy Recovery", icon: "network", primary: "Open DMG")
                    installerCard(id: "codex-cli", title: "OpenAI Codex CLI", icon: "terminal", primary: "Install")
                    installerCard(id: "claude-code-cli", title: "Claude Code CLI", icon: "terminal", primary: "Install")
                    installerCard(id: "gemini-cli", title: "Gemini CLI", icon: "terminal", primary: "Install")
                    installerCard(id: "codex-app", title: "OpenAI Codex App", icon: "app.dashed", primary: "Open Official Page")
                }
                OutputCard(title: "Installer Output", text: state.installerResult?.combinedOutput ?? "", placeholder: "Run installer doctor, dry-run, verify, or open to see output.", collapsedByDefault: true)
            }
            .padding()
        }
        .task {
            if state.installerDoctor == nil {
                await state.runInstallerDoctor()
            }
        }
        .alert("Install with AgentLink?", isPresented: Binding(
            get: { pendingInstallID != nil },
            set: { if !$0 { pendingInstallID = nil } }
        )) {
            Button("Cancel", role: .cancel) { pendingInstallID = nil }
            Button("Install") {
                if let id = pendingInstallID {
                    Task { await state.runInstallerInstall(id) }
                }
                pendingInstallID = nil
            }
        } message: {
            Text("AgentLink will run only the fixed installer command from its catalog. It will not use sudo, collect passwords, or install from unofficial mirrors.")
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Installer Center")
                .font(.title2.weight(.semibold))
            Text("Recover local agent tools from official sources. Dry-run first; no silent sudo, no password capture, no proxy/TUN auto-enable.")
                .foregroundStyle(.secondary)
        }
    }

    private func installerCard(id: String, title: String, icon: String, primary: String) -> some View {
        let report = installerReport(id)
        return VStack(alignment: .leading, spacing: 10) {
            HStack {
                Label(title, systemImage: icon)
                    .font(.headline)
                Spacer()
                StatusBadge(text: statusLabel(report?.status), kind: statusKind(report?.status))
            }
            Text(report?.nextAction ?? defaultDescription(id))
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
            if id == "clash-verge-rev" {
                StatusBadge(text: (report?.assetPath?.isEmpty == false) ? "Embedded DMG available" : "Embedded DMG missing", kind: (report?.assetPath?.isEmpty == false) ? .ok : .warn)
                Text("AgentLink will not enable proxy/TUN automatically.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if let command = report?.commands?.first?.display {
                CommandPreview(title: "Copyable Command", command: command)
            }
            HStack {
                Button("Dry Run") { Task { await state.runInstallerDryRun(id) } }
                    .disabled(state.isRunning)
                Button("Verify") { Task { await state.runInstallerVerify(id) } }
                    .disabled(state.isRunning)
                if id == "clash-verge-rev" || id == "codex-app" {
                    Button(primary) { Task { await state.runInstallerOpen(id) } }
                        .disabled(state.isRunning)
                } else {
                    Button(primary) { pendingInstallID = id }
                        .disabled(state.isRunning)
                        .buttonStyle(.borderedProminent)
                }
            }
            ForEach(report?.warnings ?? [], id: \.self) { warning in
                Text(warning)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .card()
    }

    private func installerReport(_ id: String) -> InstallerReport? {
        if state.installerActionReport?.id == id {
            return state.installerActionReport
        }
        return state.installerDoctor?.installers?.first(where: { $0.id == id })
    }

    private func statusLabel(_ status: String?) -> String {
        switch status {
        case "installed", "installed_verified": return "Installed"
        case "available": return "Available"
        case "dry_run": return "Dry-run"
        case "missing_dependency": return "Missing dependency"
        case "manual_action_required": return "Manual"
        case "failed": return "Failed"
        default: return "Unknown"
        }
    }

    private func statusKind(_ status: String?) -> StatusBadge.Kind {
        switch status {
        case "installed", "installed_verified": return .ok
        case "available", "dry_run", "manual_action_required": return .neutral
        case "missing_dependency": return .warn
        case "failed": return .fail
        default: return .neutral
        }
    }

    private func defaultDescription(_ id: String) -> String {
        switch id {
        case "clash-verge-rev":
            return "Open the cached Clash Verge Rev DMG when present, or fetch it while online."
        case "codex-app":
            return "Open the official OpenAI Codex App download page."
        default:
            return "Install from the fixed official package name in AgentLink's catalog."
        }
    }
}
