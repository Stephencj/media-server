import SwiftUI

struct HomeView: View {
    @StateObject private var viewModel = HomeViewModel()

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 50) {
                    // Continue Watching
                    if !viewModel.continueWatching.isEmpty {
                        MediaRowView(
                            title: "Continue Watching",
                            items: viewModel.continueWatching.map { $0.media },
                            progress: Dictionary(uniqueKeysWithValues: viewModel.continueWatching.map { ($0.media.id, $0.progress) })
                        )
                    }

                    // Recently Added
                    if !viewModel.recentlyAdded.isEmpty {
                        MediaRowView(
                            title: "Recently Added",
                            items: viewModel.recentlyAdded
                        )
                    }

                    // Movies
                    if !viewModel.movies.isEmpty {
                        MediaRowView(
                            title: "Movies",
                            items: viewModel.movies
                        )
                    }

                    // TV Shows
                    if !viewModel.tvShows.isEmpty {
                        MediaRowView(
                            title: "TV Shows",
                            items: viewModel.tvShows
                        )
                    }
                }
                .padding(.vertical, 50)
            }
            .navigationTitle("Home")
            .overlay {
                if viewModel.isLoading && viewModel.recentlyAdded.isEmpty {
                    ProgressView("Loading...")
                }
            }
            .alert("Error", isPresented: .constant(viewModel.errorMessage != nil)) {
                Button("OK") { viewModel.errorMessage = nil }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
        .task {
            await viewModel.loadContent()
        }
        .refreshable {
            await viewModel.loadContent()
        }
    }
}

struct MediaRowView: View {
    let title: String
    let items: [Media]
    var progress: [Int64: WatchProgress]? = nil

    @State private var showingSaveConfirmation = false
    @State private var showingWatchedConfirmation = false
    @State private var selectedMedia: Media?

    private let api = APIClient.shared

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            Text(title)
                .font(.title2)
                .fontWeight(.bold)
                .padding(.horizontal, 50)

            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(spacing: 40) {
                    ForEach(items) { media in
                        NavigationLink(value: media) {
                            MediaCardView(
                                media: media,
                                progress: progress?[media.id],
                                onSave: { media in
                                    selectedMedia = media
                                    Task {
                                        try? await api.addToWatchlist(mediaId: media.id, mediaType: media.type.rawValue)
                                        showingSaveConfirmation = true
                                    }
                                },
                                onMarkWatched: { media in
                                    selectedMedia = media
                                    Task {
                                        try? await api.markAsWatched(mediaId: media.id, mediaType: media.type.rawValue)
                                        showingWatchedConfirmation = true
                                    }
                                }
                            )
                        }
                        .buttonStyle(.card)
                    }
                }
                .padding(.horizontal, 50)
            }
        }
        .navigationDestination(for: Media.self) { media in
            if media.type == .tvshow {
                TVShowDetailView(show: media)
            } else {
                MediaDetailView(media: media)
            }
        }
        .alert("Saved", isPresented: $showingSaveConfirmation) {
            Button("OK", role: .cancel) { }
        } message: {
            Text("\(selectedMedia?.title ?? "Item") added to your list")
        }
        .alert("Marked as Watched", isPresented: $showingWatchedConfirmation) {
            Button("OK", role: .cancel) { }
        } message: {
            Text("\(selectedMedia?.title ?? "Item") marked as watched")
        }
    }
}

#Preview {
    HomeView()
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
