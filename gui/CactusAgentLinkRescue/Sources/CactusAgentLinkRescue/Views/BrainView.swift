import SwiftUI

struct BrainView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    ActionButton(title: "Brain Doctor", systemImage: "brain.head.profile", disabled: state.isRunning) { Task { await state.runBrainDoctor() } }
                    ActionButton(title: "Brain Selftest", systemImage: "checkmark.seal", disabled: state.isRunning) { Task { await state.runBrainSelftest() } }
                }
                VStack(alignment: .leading, spacing: 8) {
                    Text(state.brainDoctor?.brainPackAvailable == true ? "Brain package detected. Qwen model and llama.cpp runtime are package-local. Brain planning works offline." : "Core package detected. Network rescue and deterministic recipes are available. Qwen Brain assets are not bundled. Use Brain package for offline planning, or fetch assets while online.")
                        .font(.body)
                    Text("Model: \(state.brainDoctor?.modelPath ?? "missing")").font(.caption).textSelection(.enabled)
                    Text("Runtime: \(state.brainDoctor?.runtimePath ?? "missing")").font(.caption).textSelection(.enabled)
                    StatusBadge(text: "SHA: \(state.brainDoctor?.modelSha256OK == true ? "OK" : "missing/failed")", kind: state.brainDoctor?.modelSha256OK == true ? .ok : .warn)
                    if let commands = state.brainDoctor?.fetchCommands, !commands.isEmpty {
                        CommandPreview(title: "Fetch / Offline Package", command: commands.joined(separator: "\n"))
                    }
                }
                .card()
                OutputCard(title: "Brain Selftest", text: state.brainSelftest?.ok == true ? "OK\nRecipe: \(state.brainSelftest?.decision?.selectedRecipe?.id ?? "")" : (state.brainSelftest?.error ?? state.latestResult?.combinedOutput ?? ""))
            }
            .padding()
        }
    }
}
