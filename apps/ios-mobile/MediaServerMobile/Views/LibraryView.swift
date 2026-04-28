import SwiftUI

struct LibraryView: View {
    let mediaType: MediaType

    @StateObject private var viewModel: LibraryViewModel

    init(mediaType: MediaType) {
        self.mediaType = mediaType
        _viewModel = StateObject(wrappedValue: LibraryViewModel(mediaType: mediaType))
    }

    private let columns = [
        GridItem(.adaptive(minimum: 150, maximum: 200), spacing: 16)
    ]

    var body: some View {
        ScrollView {
            if viewModel.items.isEmpty && !viewModel.isLoading {
                VStack(spacing: 12) {
                    Image(systemName: mediaType == .movie ? "film" : "tv")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("No \(mediaType == .movie ? "Movies" : "TV Shows")")
                        .font(.title2)
                        .fontWeight(.semibold)
                    Text("Add media sources in Settings to scan for content")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.top, 100)
                .frame(maxWidth: .infinity)
            } else {
                LazyVGrid(columns: columns, spacing: 16) {
                    ForEach(viewModel.sortedItems) { media in
                        NavigationLink(destination: destination(for: media)) {
                            MediaCardView(media: media)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(mediaType == .movie ? "Movies" : "TV Shows")
        .toolbar {
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
        .overlay {
            if viewModel.isLoading && viewModel.items.isEmpty {
                ProgressView("Loading...")
            }
        }
        .task {
            await viewModel.loadItems()
        }
        .refreshable {
            await viewModel.loadItems()
        }
    }

    @ViewBuilder
    private func destination(for media: Media) -> some View {
        if media.type == .tvshow {
            TVShowDetailView(show: media)
        } else {
            MediaDetailView(media: media)
        }
    }
}

#Preview {
    LibraryView(mediaType: .movie)
        .environmentObject(AuthService.shared)
        .environmentObject(AppState.shared)
}
