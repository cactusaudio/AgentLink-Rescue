import SwiftUI

struct BrainSandboxView: View {
    @EnvironmentObject var state: AppState
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Cactus Brain Sandbox")
                        .font(.title2.weight(.semibold))
                    Text("Local Gemma chat. It cannot run commands or change your system.")
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Close") { dismiss() }
            }
            if state.brainDoctor?.brainPackAvailable != true {
                VStack(alignment: .leading, spacing: 8) {
                    StatusBadge(text: "Brain assets missing", kind: .warn)
                    Text("Use the Brain package or fetch assets while online.")
                        .foregroundStyle(.secondary)
                    if let commands = state.brainDoctor?.fetchCommands, !commands.isEmpty {
                        CommandPreview(title: "Fetch Command", command: commands.joined(separator: "\n"))
                    }
                }
                .card()
            }
            TextEditor(text: $state.brainChatPrompt)
                .font(.body)
                .frame(minHeight: 100)
                .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.secondary.opacity(0.25)))
            HStack {
                ActionButton(title: "Send", systemImage: "paperplane", disabled: state.isRunning || state.brainDoctor?.brainPackAvailable != true || state.brainChatPrompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty) {
                    Task { await state.runBrainChat() }
                }
                Button("Attach latest redacted report") {
                    let report = state.codexDispatch.isEmpty ? state.humanReport : state.codexDispatch
                    if !report.isEmpty {
                        state.brainChatPrompt += "\n\nRedacted latest report:\n" + report
                    }
                }
                .disabled(state.humanReport.isEmpty && state.codexDispatch.isEmpty)
                Button("Clear") {
                    state.brainChatPrompt = ""
                    state.brainChatReport = nil
                    state.brainChatResult = nil
                }
                Spacer()
            }
            OutputCard(title: "Sandbox Response", text: state.brainChatReport?.response ?? state.brainChatResult?.combinedOutput ?? "", placeholder: "Ask a local explanation question. The sandbox cannot execute anything.", collapsedByDefault: false, minHeight: 160, maxHeight: 260)
            Text("If you need repair, use Guided Rescue or Expert Console. The sandbox is explanation-only.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding()
    }
}
