import SwiftUI

struct SettingsView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Package").font(.headline)
                    Text("Package root: \(state.client.packageRoot.path)").textSelection(.enabled)
                    Text("Binary: \(state.client.binaryURL.path)").textSelection(.enabled)
                    Text("Redaction: enabled")
                    Text("AGENTLINK_MODEL_PATH: \(ProcessInfo.processInfo.environment["AGENTLINK_MODEL_PATH"] ?? "not set")").textSelection(.enabled)
                    Text("AGENTLINK_LLAMA_CLI: \(ProcessInfo.processInfo.environment["AGENTLINK_LLAMA_CLI"] ?? "not set")").textSelection(.enabled)
                }
                .card()
                OutputCard(title: "Recent Command Log", text: state.logLines.joined(separator: "\n"))
            }
            .padding()
        }
    }
}
