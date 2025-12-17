import SwiftUI

struct LibraryView: View {
    let mediaType: MediaType

    @StateObject private var viewModel: LibraryViewModel

    init(mediaType: MediaType) {
        self.mediaType = mediaType
        _viewModel = StateObject(wrappedValue: LibraryViewModel(mediaType: mediaType))
    }

    private let columns = [
        GridItem(.adaptive(minimum: 250, maximum: 300), spacing: 40)
    ]

    var body: some View {
        NavigationStack {
            ScrollView {
                if viewModel.items.isEmpty && !viewModel.isLoading {
                    ContentUnavailableView(
                        "No \(mediaType == .movie ? "Movies" : "TV Shows")",
                        systemImage: mediaType == .movie ? "film" : "tv",
                        description: Text("Add media sources in Settings to scan for content")
                    )
                } else {
                    LazyVGrid(columns: columns, spacing: 50) {
                        ForEach(viewModel.items) { media in
                            NavigationLink(value: media) {
                                MediaCardView(media: media)
                            }
                            .buttonStyle(.card)
                        }
                    }
                    .padding(50)
                }
            }
            .navigationTitle(mediaType == .movie ? "Movies" : "TV Shows")
            .navigationDestination(for: Media.self) { media in
                MediaDetailView(media: media)
            }
            .overlay {
                if viewModel.isLoading && viewModel.items.isEmpty {
                    ProgressView("Loading...")
                }
            }
        }
        .task {
            await viewModel.loadItems()
        }
        .refreshable {
            await viewModel.loadItems()
        }
    }
}

#Preview {
    LibraryView(mediaType: .movie)
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
