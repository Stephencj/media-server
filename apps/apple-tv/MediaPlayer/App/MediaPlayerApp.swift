import SwiftUI

@main
struct MediaPlayerApp: App {
    @StateObject private var authService = AuthService.shared
    @StateObject private var appState = AppState.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(authService)
                .environmentObject(appState)
        }
    }
}

struct RootView: View {
    @EnvironmentObject var authService: AuthService

    var body: some View {
        Group {
            if authService.isAuthenticated {
                MainTabView()
            } else {
                LoginView()
            }
        }
        .task {
            await authService.checkAuthState()
        }
    }
}
