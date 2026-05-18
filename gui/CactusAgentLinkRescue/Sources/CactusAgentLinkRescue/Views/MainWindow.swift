import SwiftUI

struct MainWindow: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        NavigationSplitView {
            SidebarView()
        } detail: {
            Group {
                switch state.page {
                case .dashboard: StatusDashboardView()
                case .guided: GuidedRescueView()
                case .installer: InstallerCenterView()
                case .readiness: ReadinessCenterView()
                case .doctor: DoctorView()
                case .brain: BrainView()
                case .plan: PlanView()
                case .dryRun: DryRunView()
                case .rescue: RescueView()
                case .rollback: RollbackView()
                case .reports: ReportsView()
                case .settings: SettingsView()
                }
            }
            .navigationTitle(state.page == .dashboard ? "Rescue" : state.page.rawValue)
        }
        .sheet(isPresented: $state.brainSandboxPresented) {
            BrainSandboxView()
                .environmentObject(state)
                .frame(minWidth: 620, minHeight: 520)
        }
    }
}

struct SidebarView: View {
    @EnvironmentObject var state: AppState
    private let mainPages: [RescuePage] = [.dashboard, .reports, .settings]
    private let developerPages: [RescuePage] = [.guided, .readiness, .installer, .doctor, .brain, .plan, .dryRun, .rescue, .rollback]

    var body: some View {
        List(selection: $state.page) {
            Section("Main") {
                ForEach(mainPages) { page in
                    Label(title(for: page), systemImage: icon(for: page))
                        .tag(page)
                }
            }
            if state.developerModeEnabled {
                Section("Developer Mode") {
                    ForEach(developerPages) { page in
                        Label(title(for: page), systemImage: icon(for: page))
                            .tag(page)
                    }
                }
            }
        }
        .navigationSplitViewColumnWidth(220)
        .onChange(of: state.developerModeEnabled) { enabled in
            if !enabled && developerPages.contains(state.page) {
                state.page = .dashboard
            }
        }
    }

    private func title(for page: RescuePage) -> String {
        switch page {
        case .dashboard: return "Rescue"
        default: return page.rawValue
        }
    }

    private func icon(for page: RescuePage) -> String {
        switch page {
        case .dashboard: return "lifepreserver"
        case .guided: return "sparkles.rectangle.stack"
        case .installer: return "square.and.arrow.down.on.square"
        case .readiness: return "checklist.checked"
        case .doctor: return "stethoscope"
        case .brain: return "brain.head.profile"
        case .plan: return "list.bullet.clipboard"
        case .dryRun: return "play.rectangle"
        case .rescue: return "wrench.and.screwdriver"
        case .rollback: return "arrow.uturn.backward.circle"
        case .reports: return "doc.text"
        case .settings: return "gearshape"
        }
    }
}
