import SwiftUI

struct SearchView: View {
    @State private var searchText = ""
    @State private var results: [Media] = []
    @State private var isSearching = false

    private let columns = [
        GridItem(.adaptive(minimum: 250, maximum: 300), spacing: 40)
    ]

    var body: some View {
        NavigationStack {
            VStack {
                if results.isEmpty && !searchText.isEmpty && !isSearching {
                    ContentUnavailableView.search(text: searchText)
                } else if results.isEmpty && searchText.isEmpty {
                    ContentUnavailableView(
                        "Search",
                        systemImage: "magnifyingglass",
                        description: Text("Enter a title to search your library")
                    )
                } else {
                    ScrollView {
                        LazyVGrid(columns: columns, spacing: 50) {
                            ForEach(results) { media in
                                NavigationLink(value: media) {
                                    MediaCardView(media: media)
                                }
                                .buttonStyle(.card)
                            }
                        }
                        .padding(50)
                    }
                }
            }
            .navigationTitle("Search")
            .searchable(text: $searchText, prompt: "Search movies and shows")
            .navigationDestination(for: Media.self) { media in
                MediaDetailView(media: media)
            }
            .onChange(of: searchText) { _, newValue in
                performSearch(query: newValue)
            }
        }
    }

    private func performSearch(query: String) {
        guard !query.isEmpty else {
            results = []
            return
        }

        isSearching = true

        // Local filtering - in a real app, you'd call an API endpoint
        Task {
            // Simulate search delay
            try? await Task.sleep(nanoseconds: 300_000_000)

            // For now, we'll need to fetch all and filter client-side
            // In production, add a search endpoint to the API
            do {
                let movies = try await APIClient.shared.getMovies(limit: 100, offset: 0)
                let shows = try await APIClient.shared.getShows(limit: 100, offset: 0)

                let allMedia = movies.items + shows.items
                let lowercaseQuery = query.lowercased()

                await MainActor.run {
                    results = allMedia.filter { media in
                        media.title.lowercased().contains(lowercaseQuery) ||
                        (media.originalTitle?.lowercased().contains(lowercaseQuery) ?? false)
                    }
                    isSearching = false
                }
            } catch {
                await MainActor.run {
                    isSearching = false
                }
            }
        }
    }
}

#Preview {
    SearchView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
