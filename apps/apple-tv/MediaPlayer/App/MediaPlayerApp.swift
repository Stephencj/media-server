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
    @EnvironmentObject var appState: AppState

    var body: some View {
        Group {
            if authService.isAuthenticated {
                MainTabView()
            } else if authService.isLoading {
                ProgressView("Connecting...")
                    .font(.title2)
            } else {
                VStack(spacing: 30) {
                    Image(systemName: "wifi.exclamationmark")
                        .font(.system(size: 60))
                        .foregroundColor(.secondary)

                    Text("Unable to connect to server")
                        .font(.title2)

                    Text(appState.serverURL)
                        .font(.subheadline)
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
