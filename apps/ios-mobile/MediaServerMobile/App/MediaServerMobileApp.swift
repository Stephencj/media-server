import SwiftUI

@main
struct MediaServerMobileApp: App {
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
            if appState.serverURL.isEmpty {
                SettingsView()
            } else if authService.isAuthenticated {
                WebView()
            } else {
                LoginPromptView()
            }
        }
        .task {
            await authService.checkAuthState()
        }
    }
}

struct LoginPromptView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                Image(systemName: "play.tv")
                    .font(.system(size: 80))
                    .foregroundColor(.blue)

                Text("Welcome to Media Server")
                    .font(.title)
                    .fontWeight(.bold)

                Text("Please configure your server URL in settings, then login through the web interface.")
                    .font(.body)
                    .multilineTextAlignment(.center)
                    .foregroundColor(.secondary)
                    .padding(.horizontal)

                NavigationLink(destination: SettingsView()) {
                    Label("Open Settings", systemImage: "gear")
                        .font(.headline)
                        .foregroundColor(.white)
                        .padding()
                        .background(Color.blue)
                        .cornerRadius(10)
                }
                .padding(.top)
            }
            .padding()
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}

#Preview {
    RootView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
