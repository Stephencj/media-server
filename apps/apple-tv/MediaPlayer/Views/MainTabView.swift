import SwiftUI

struct MainTabView: View {
    @EnvironmentObject var appState: AppState
    @State private var selectedTab = 0

    var body: some View {
        ZStack {
            TabView(selection: $selectedTab) {
                HomeView()
                    .tabItem {
                        Label("Home", systemImage: "house.fill")
                    }
                    .tag(0)

                LibraryView(mediaType: .movie)
                    .tabItem {
                        Label("Movies", systemImage: "film")
                    }
                    .tag(1)

                LibraryView(mediaType: .tvshow)
                    .tabItem {
                        Label("TV Shows", systemImage: "tv")
                    }
                    .tag(2)

                PlaylistView()
                    .tabItem {
                        Label("Playlists", systemImage: "music.note.list")
                    }
                    .tag(3)

                ChannelListView()
                    .tabItem {
                        Label("Channels", systemImage: "tv.inset.filled")
                    }
                    .tag(4)

                SearchView()
                    .tabItem {
                        Label("Search", systemImage: "magnifyingglass")
                    }
                    .tag(5)

                SettingsView()
                    .tabItem {
                        Label("Settings", systemImage: "gear")
                    }
                    .tag(6)
            }

            // Error toast overlay
            VStack {
                ForEach(appState.errors) { error in
                    ErrorToastView(error: error, onDismiss: {
                        appState.errors.removeAll { $0.id == error.id }
                    })
                    .padding(.horizontal, 60)
                    .padding(.top, 40)
                }
                Spacer()
            }
        }
    }
}

struct ErrorToastView: View {
    let error: AppError
    let onDismiss: () -> Void

    var body: some View {
        HStack(spacing: 20) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.title)
                .foregroundColor(.white)

            VStack(alignment: .leading, spacing: 5) {
                Text(error.title)
                    .font(.headline)
                    .foregroundColor(.white)

                Text(error.message)
                    .font(.subheadline)
                    .foregroundColor(.white.opacity(0.9))
            }

            Spacer()

            Button {
                onDismiss()
            } label: {
                Image(systemName: "xmark.circle.fill")
                    .font(.title2)
                    .foregroundColor(.white.opacity(0.7))
            }
        }
        .padding(30)
        .background(
            RoundedRectangle(cornerRadius: 15)
                .fill(Color.red.opacity(0.9))
                .shadow(radius: 10)
        )
        .transition(.move(edge: .top).combined(with: .opacity))
    }
}

#Preview {
    MainTabView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
