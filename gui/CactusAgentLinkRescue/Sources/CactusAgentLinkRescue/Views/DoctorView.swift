import SwiftUI

struct DoctorView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                ActionButton(title: "Run Doctor", systemImage: "stethoscope", disabled: state.isRunning) {
                    Task { await state.runDoctor() }
                }
                VStack(alignment: .leading, spacing: 8) {
                    Text("Likely Failures").font(.headline)
                    ForEach(state.doctor?.classifications ?? [], id: \.self) { item in
                        StatusBadge(text: item, kind: item == "OK" ? .ok : .warn)
                    }
                    HStack {
                        Text("Recommended repair:")
                        StatusBadge(text: state.recommendationLabel, kind: state.recommendationKind)
                    }
                    if let path = state.doctor?.reportPath {
                        Text(path).font(.caption).textSelection(.enabled)
                    }
                    if state.doctor == nil {
                        Text("Run Doctor to collect current network and agent-link facts.")
                            .foregroundStyle(.secondary)
                    }
                }
                .card()
                OutputCard(title: "Raw Doctor Output", text: state.doctorResult?.stdout ?? "", placeholder: "No doctor run has completed yet.", collapsedByDefault: true)
            }
            .padding()
        }
    }
}
