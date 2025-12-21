import SwiftUI

struct MainTabView: View {
    @State private var selectedTab = 0

    var body: some View {
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

            SearchView()
                .tabItem {
                    Label("Search", systemImage: "magnifyingglass")
                }
                .tag(4)

            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gear")
                }
                .tag(5)
        }
    }
}

#Preview {
    MainTabView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
