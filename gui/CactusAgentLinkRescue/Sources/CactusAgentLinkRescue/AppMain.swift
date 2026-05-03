import AppKit
import SwiftUI

@main
struct CactusAgentLinkRescueLauncher {
    static func main() {
        if CommandLine.arguments.contains("--selftest-gui") {
            let code = GUISelftest.run()
            Foundation.exit(code)
        }
        if CommandLine.arguments.contains("--selftest-gui-long-output") {
            let code = GUISelftest.runLongOutput()
            Foundation.exit(code)
        }
        let app = NSApplication.shared
        let delegate = CactusAgentLinkRescueDelegate()
        CactusAgentLinkRescueDelegate.retained = delegate
        app.delegate = delegate
        app.setActivationPolicy(.regular)
        MainActor.assumeIsolated {
            delegate.createWindowIfNeeded()
        }
        app.activate(ignoringOtherApps: true)
        app.run()
    }
}

final class CactusAgentLinkRescueDelegate: NSObject, NSApplicationDelegate {
    static var retained: CactusAgentLinkRescueDelegate?
    private var window: NSWindow?
    private var state: AppState?

    func applicationDidFinishLaunching(_ notification: Notification) {
        MainActor.assumeIsolated {
            createWindowIfNeeded()
        }
    }

    @MainActor
    func createWindowIfNeeded() {
        guard window == nil else { return }
        let state = AppState()
        self.state = state
        let root = MainWindow()
            .environmentObject(state)
            .frame(minWidth: 1120, minHeight: 720)
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1180, height: 760),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Cactus AgentLink Rescue"
        window.center()
        window.contentView = NSHostingView(rootView: root)
        window.makeKeyAndOrderFront(nil)
        self.window = window

        Task { @MainActor in
            await state.bootstrap()
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        true
    }
}
