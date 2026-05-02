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
            .navigationTitle(state.page.rawValue)
        }
    }
}

struct SidebarView: View {
    @EnvironmentObject var state: AppState

    var body: some View {
        List(RescuePage.allCases, selection: $state.page) { page in
            Label(page.rawValue, systemImage: icon(for: page))
                .tag(page)
        }
        .navigationSplitViewColumnWidth(220)
    }

    private func icon(for page: RescuePage) -> String {
        switch page {
        case .dashboard: return "gauge.with.dots.needle.50percent"
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
