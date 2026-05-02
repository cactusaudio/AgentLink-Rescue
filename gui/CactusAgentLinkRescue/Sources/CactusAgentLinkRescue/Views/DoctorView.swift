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
                    Text("Recommended repair: \(state.doctor?.recommendedRepairLevel ?? "unknown")")
                    if let path = state.doctor?.reportPath {
                        Text(path).font(.caption).textSelection(.enabled)
                    }
                }
                .card()
                OutputCard(title: "Raw Doctor Output", text: state.latestResult?.stdout ?? "")
            }
            .padding()
        }
    }
}
