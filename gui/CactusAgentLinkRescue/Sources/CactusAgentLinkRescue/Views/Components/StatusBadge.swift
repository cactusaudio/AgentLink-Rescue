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
    var placeholder = "No output."
    var collapsedByDefault = false
    var minHeight: CGFloat = 120
    var maxHeight: CGFloat = 260
    @State private var expanded: Bool

    init(title: String, text: String, placeholder: String = "No output.", collapsedByDefault: Bool = false, minHeight: CGFloat = 120, maxHeight: CGFloat = 260) {
        self.title = title
        self.text = text
        self.placeholder = placeholder
        self.collapsedByDefault = collapsedByDefault
        self.minHeight = minHeight
        self.maxHeight = maxHeight
        _expanded = State(initialValue: !collapsedByDefault)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            DisclosureGroup(isExpanded: $expanded) {
                ScrollView {
                    Text(text.isEmpty ? placeholder : text)
                        .font(.system(.caption, design: .monospaced))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .textSelection(.enabled)
                        .padding(10)
                }
                .frame(minHeight: minHeight, maxHeight: maxHeight)
                .background(Color(nsColor: .textBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
            } label: {
                HStack {
                    Text(title).font(.headline)
                    Spacer()
                    Button("Copy") {
                        NSPasteboard.general.clearContents()
                        NSPasteboard.general.setString(text, forType: .string)
                    }
                    .disabled(text.isEmpty)
                }
            }
        }
        .padding()
        .background(Color(nsColor: .controlBackgroundColor), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct CommandPreview: View {
    let title: String
    let command: String

    var body: some View {
        OutputCard(title: title, text: command, minHeight: 44, maxHeight: 92)
    }
}

struct InfoRow: View {
    let label: String
    let value: String
    var monospaced = false

    var body: some View {
        HStack(alignment: .top) {
            Text(label)
                .foregroundStyle(.secondary)
                .frame(width: 120, alignment: .leading)
            Text(value)
                .font(monospaced ? .system(.caption, design: .monospaced) : .body)
                .lineLimit(2)
                .truncationMode(.middle)
                .textSelection(.enabled)
            Spacer(minLength: 0)
        }
    }
}
