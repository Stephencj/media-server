import SwiftUI

@main
struct MediaServerMobileApp: App {
    @StateObject private var authService = AuthService.shared
    @StateObject private var appState = AppState.shared

    init() {
        AudioSessionManager.shared.activatePlayback()
    }

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
    @EnvironmentObject var appState: AppState

    var body: some View {
        Group {
            if authService.isAuthenticated {
                MainTabView()
            } else if authService.isLoading {
                ProgressView("Connecting...")
                    .font(.title3)
            } else {
                VStack(spacing: 20) {
                    Image(systemName: "wifi.exclamationmark")
                        .font(.system(size: 50))
                        .foregroundColor(.secondary)

                    Text("Unable to connect")
                        .font(.title3)

                    Text(appState.serverURL)
                        .font(.caption)
                        .foregroundColor(.secondary)

                    if let err = authService.lastAuthError {
                        Text(err)
                            .font(.caption)
                            .foregroundColor(.red)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)
                    }

                    Button("Retry") {
                        Task {
                            await authService.ensureAuthenticated()
                        }
                    }
                }
            }
        }
        .task {
            await authService.ensureAuthenticated()
        }
    }
}
