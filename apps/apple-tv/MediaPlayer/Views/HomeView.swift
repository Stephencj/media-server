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

                    // Dynamic sections based on what's loaded from API
                    ForEach(viewModel.sections.sorted(by: { $0.displayOrder < $1.displayOrder })) { section in
                        if let media = viewModel.sectionMedia[section.slug], !media.isEmpty {
                            MediaRowView(
                                title: section.name,
                                items: media
                            )
                        }
                    }
                }
                .padding(.vertical, 50)
            }
            .navigationTitle("Home")
            .overlay {
                if viewModel.isLoading && viewModel.sections.isEmpty {
                    ProgressView("Loading...")
                }
            }
            .alert("Error", isPresented: Binding(
                get: { viewModel.error != nil },
                set: { if !$0 { viewModel.error = nil } }
            )) {
                Button("OK") { viewModel.error = nil }
            } message: {
                Text(viewModel.error?.localizedDescription ?? "")
            }
        }
        .task {
            await viewModel.loadData()
        }
        .refreshable {
            await viewModel.loadData()
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
