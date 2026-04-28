import SwiftUI

struct SearchView: View {
    @State private var searchText = ""
    @State private var results: [Media] = []
    @State private var isSearching = false
    @State private var searchTask: Task<Void, Never>?

    private let columns = [
        GridItem(.adaptive(minimum: 150, maximum: 200), spacing: 16)
    ]

    var body: some View {
        VStack {
            if results.isEmpty && !searchText.isEmpty && !isSearching {
                VStack(spacing: 12) {
                    Image(systemName: "magnifyingglass")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("No Results for \"\(searchText)\"")
                        .font(.title2)
                        .fontWeight(.semibold)
                    Text("Check the spelling or try a different search term")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.top, 100)
                .frame(maxWidth: .infinity)
            } else if results.isEmpty && searchText.isEmpty {
                VStack(spacing: 12) {
                    Image(systemName: "magnifyingglass")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("Search")
                        .font(.title2)
                        .fontWeight(.semibold)
                    Text("Enter a title to search your library")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.top, 100)
                .frame(maxWidth: .infinity)
            } else {
                ScrollView {
                    LazyVGrid(columns: columns, spacing: 16) {
                        ForEach(results) { media in
                            NavigationLink(destination: MediaDetailView(media: media)) {
                                MediaCardView(media: media)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle("Search")
        .searchable(text: $searchText, prompt: "Search movies and shows")
        .onChange(of: searchText) { newValue in
            performSearch(query: newValue)
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
