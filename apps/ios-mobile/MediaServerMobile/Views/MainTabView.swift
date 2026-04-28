import SwiftUI

struct MainTabView: View {
    @EnvironmentObject var appState: AppState
    @State private var selectedTab = 4

    var body: some View {
        ZStack {
            TabView(selection: $selectedTab) {
                NavigationView {
                    HomeView()
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("Home", systemImage: "house.fill")
                }
                .tag(0)

                NavigationView {
                    LibraryView(mediaType: .movie)
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("Movies", systemImage: "film")
                }
                .tag(1)

                NavigationView {
                    LibraryView(mediaType: .tvshow)
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("TV Shows", systemImage: "tv")
                }
                .tag(2)

                NavigationView {
                    PlaylistView()
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("Playlists", systemImage: "music.note.list")
                }
                .tag(3)

                NavigationView {
                    ChannelListView()
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("Channels", systemImage: "tv.inset.filled")
                }
                .tag(4)

                NavigationView {
                    SearchView()
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("Search", systemImage: "magnifyingglass")
                }
                .tag(5)

                NavigationView {
                    SettingsView()
                }
                .navigationViewStyle(.stack)
                .tabItem {
                    Label("Settings", systemImage: "gearshape.fill")
                }
                .tag(6)
            }
        }
    }
}
