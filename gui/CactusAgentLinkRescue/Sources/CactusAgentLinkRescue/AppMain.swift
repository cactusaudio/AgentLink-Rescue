import SwiftUI

@main
struct CactusAgentLinkRescueApp: App {
    @StateObject private var state: AppState

    init() {
        if CommandLine.arguments.contains("--selftest-gui") {
            let code = GUISelftest.run()
            Foundation.exit(code)
        }
        _state = StateObject(wrappedValue: AppState())
    }

    var body: some Scene {
        WindowGroup {
            MainWindow()
                .environmentObject(state)
                .frame(minWidth: 1120, minHeight: 720)
                .task {
                    await state.bootstrap()
                }
        }
        .windowStyle(.titleBar)
    }
}
