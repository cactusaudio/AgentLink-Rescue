import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Package").font(.headline)
                    InfoRow(label: "Package root", value: state.client.packageRoot.path, monospaced: true)
                    InfoRow(label: "Binary", value: state.client.binaryURL.path, monospaced: true)
                    InfoRow(label: "Redaction", value: "enabled")
                    InfoRow(label: "Model override", value: ProcessInfo.processInfo.environment["AGENTLINK_MODEL_PATH"] ?? "not set", monospaced: true)
                    InfoRow(label: "Runtime override", value: ProcessInfo.processInfo.environment["AGENTLINK_LLAMA_CLI"] ?? "not set", monospaced: true)
                }
                .card()
                OutputCard(title: "Recent Command Log", text: state.logLines.joined(separator: "\n"), placeholder: "No commands have run yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }
}
