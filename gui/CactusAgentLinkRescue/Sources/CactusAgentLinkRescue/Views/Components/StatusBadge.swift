import SwiftUI

struct StatusBadge: View {
    let text: String
    var kind: Kind = .neutral

    enum Kind {
        case ok, warn, fail, neutral

        var color: Color {
            switch self {
            case .ok: return .green
            case .warn: return .orange
            case .fail: return .red
            case .neutral: return .secondary
            }
        }
    }

    var body: some View {
        Text(text)
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(kind.color.opacity(0.16), in: Capsule())
            .foregroundStyle(kind.color)
    }
}

struct ActionButton: View {
    let title: String
    var systemImage: String = "play.fill"
    var disabled = false
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            Label(title, systemImage: systemImage)
        }
        .buttonStyle(.borderedProminent)
        .disabled(disabled)
    }
}

struct OutputCard: View {
    let title: String
    let text: String

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(title).font(.headline)
                Spacer()
                Button("Copy") {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(text, forType: .string)
                }
            }
            ScrollView {
                Text(text.isEmpty ? "No output." : text)
                    .font(.system(.caption, design: .monospaced))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .textSelection(.enabled)
                    .padding(10)
            }
            .frame(minHeight: 120)
            .background(Color(nsColor: .textBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
        }
        .padding()
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct CommandPreview: View {
    let title: String
    let command: String

    var body: some View {
        OutputCard(title: title, text: command)
    }
}
