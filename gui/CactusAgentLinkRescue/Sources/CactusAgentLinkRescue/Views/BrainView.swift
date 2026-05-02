import SwiftUI

struct BrainView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    ActionButton(title: "Brain Doctor", systemImage: "brain.head.profile", disabled: state.isRunning) { Task { await state.runBrainDoctor() } }
                    ActionButton(title: "Brain Selftest", systemImage: "checkmark.seal", disabled: state.isRunning || state.brainDoctor?.brainPackAvailable == false) { Task { await state.runBrainSelftest() } }
                }
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 320), spacing: 12)], spacing: 12) {
                    packCard
                    modelCard
                    runtimeCard
                    selftestCard
                }
                OutputCard(title: "Raw Brain Output", text: state.brainSelftestResult?.combinedOutput ?? state.brainDoctorResult?.combinedOutput ?? "", placeholder: "Run Brain Doctor or Brain Selftest to see raw output.", collapsedByDefault: true)
            }
            .padding()
        }
    }

    private var packCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Brain Pack Status").font(.headline)
            let available = state.brainDoctor?.brainPackAvailable == true
            StatusBadge(text: available ? "Brain package detected" : "Brain assets missing", kind: available ? .ok : .warn)
            Text(available ? "Gemma 4 E4B model and llama.cpp runtime are package-local. Brain planning works offline." : "Core package detected. Network rescue and deterministic recipes remain available.")
                .foregroundStyle(.secondary)
            if state.brainDoctor?.brainPackAvailable == false {
                Text("Use the Brain package for offline planning, or fetch assets while online.")
                    .foregroundStyle(.secondary)
            }
            if let commands = state.brainDoctor?.fetchCommands, !commands.isEmpty {
                CommandPreview(title: "Fetch / Offline Package", command: commands.joined(separator: "\n"))
            }
        }
        .card()
    }

    private var modelCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Model").font(.headline)
            StatusBadge(text: state.brainDoctor?.modelSha256OK == true ? "SHA OK" : "Missing or failed", kind: state.brainDoctor?.modelSha256OK == true ? .ok : .warn)
            InfoRow(label: "Name", value: state.brainDoctor?.modelName ?? "Gemma 4 E4B-it Q4_K_M")
            InfoRow(label: "Approx size", value: "5.07 GB")
            InfoRow(label: "Path", value: state.brainDoctor?.modelPath ?? "missing", monospaced: true)
        }
        .card()
    }

    private var runtimeCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Runtime").font(.headline)
            StatusBadge(text: state.brainDoctor?.runtimeExecutable == true ? "Executable" : "Missing", kind: state.brainDoctor?.runtimeExecutable == true ? .ok : .warn)
            InfoRow(label: "Backend", value: state.brainDoctor?.backend ?? "llama-cli")
            InfoRow(label: "Path", value: state.brainDoctor?.runtimePath ?? "missing", monospaced: true)
        }
        .card()
    }

    private var selftestCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Selftest").font(.headline)
            if state.isRunning {
                StatusBadge(text: "Running", kind: .neutral)
            } else if state.brainSelftest?.ok == true {
                StatusBadge(text: "OK", kind: .ok)
                InfoRow(label: "Recipe", value: state.brainSelftest?.decision?.selectedRecipe?.id ?? "none")
            } else if let error = state.brainSelftest?.error {
                StatusBadge(text: "Failed", kind: .fail)
                Text(error).foregroundStyle(.secondary)
            } else {
                StatusBadge(text: "Not run", kind: .neutral)
                Text("Runs a local Gemma planner-shaped JSON test. It does not modify files.")
                    .foregroundStyle(.secondary)
            }
        }
        .card()
    }
}
