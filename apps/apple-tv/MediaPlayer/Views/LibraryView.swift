import SwiftUI

struct LibraryView: View {
    let mediaType: MediaType

    @StateObject private var viewModel: LibraryViewModel

    // Drives default focus so that moving down from the tab bar lands on the
    // grid instead of the Sort menu in the toolbar.
    @FocusState private var focusedMediaID: Int64?
    @State private var showMatchReview = false

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
                        ForEach(viewModel.sortedItems) { media in
                            NavigationLink(value: media) {
                                MediaCardView(media: media)
                            }
                            .buttonStyle(.card)
                            .focused($focusedMediaID, equals: media.id)
                        }
                    }
                    .padding(50)
                    .focusSection()
                }
            }
            .defaultFocus($focusedMediaID, viewModel.sortedItems.first?.id)
            .navigationTitle(mediaType == .movie ? "Movies" : "TV Shows")
            .toolbar {
                if mediaType == .movie {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            showMatchReview = true
                        } label: {
                            Label("Check Matches", systemImage: "checkmark.seal")
                        }
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Picker("Sort by", selection: $viewModel.sortOption) {
                            ForEach(SortOption.allCases, id: \.self) { option in
                                Text(option.rawValue).tag(option)
                            }
                        }

                        Button {
                            viewModel.sortOrder = viewModel.sortOrder == .ascending ? .descending : .ascending
                        } label: {
                            Label(
                                viewModel.sortOrder == .ascending ? "Ascending" : "Descending",
                                systemImage: viewModel.sortOrder == .ascending ? "arrow.up" : "arrow.down"
                            )
                        }
                    } label: {
                        Label("Sort", systemImage: "arrow.up.arrow.down")
                    }
                }
            }
            .navigationDestination(for: Media.self) { media in
                if media.type == .tvshow {
                    TVShowDetailView(show: media)
                } else {
                    MediaDetailView(media: media)
                }
            }
            .navigationDestination(isPresented: $showMatchReview) {
                MatchReviewView()
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
