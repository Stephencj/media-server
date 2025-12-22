import SwiftUI

struct SearchView: View {
    @State private var searchText = ""
    @State private var results: [Media] = []
    @State private var isSearching = false
    @State private var searchTask: Task<Void, Never>?

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
        // Cancel previous search task
        searchTask?.cancel()

        guard !query.isEmpty else {
            results = []
            return
        }

        isSearching = true

        searchTask = Task {
            // 300ms debounce
            try? await Task.sleep(nanoseconds: 300_000_000)
            guard !Task.isCancelled else { return }

            // Use cached media for faster search
            do {
                let allMedia = await APIClient.shared.getCachedMedia()
                guard !Task.isCancelled else { return }

                let lowercaseQuery = query.lowercased()

                await MainActor.run {
                    results = allMedia.filter { media in
                        media.title.lowercased().contains(lowercaseQuery) ||
                        (media.originalTitle?.lowercased().contains(lowercaseQuery) ?? false)
                    }
                    isSearching = false
                }
            } catch {
                guard !Task.isCancelled else { return }
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
