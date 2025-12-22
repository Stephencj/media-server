import Foundation

enum SortOption: String, CaseIterable {
    case title = "Title"
    case year = "Year"
    case rating = "Rating"
    case dateAdded = "Date Added"
    case duration = "Duration"
}

enum SortOrder {
    case ascending, descending
}

@MainActor
class LibraryViewModel: ObservableObject {
    @Published var items: [Media] = []
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var sortOption: SortOption = .dateAdded
    @Published var sortOrder: SortOrder = .descending

    private let mediaType: MediaType
    private let api = APIClient.shared

    private var currentOffset = 0
    private let pageSize = 50
    private var hasMoreItems = true

    init(mediaType: MediaType) {
        self.mediaType = mediaType
    }

    var sortedItems: [Media] {
        items.sorted { a, b in
            let result: Bool
            switch sortOption {
            case .title:
                result = a.title.localizedCaseInsensitiveCompare(b.title) == .orderedAscending
            case .year:
                result = (a.year ?? 0) < (b.year ?? 0)
            case .rating:
                result = (a.rating ?? 0) < (b.rating ?? 0)
            case .dateAdded:
                result = (a.createdAt ?? "") < (b.createdAt ?? "")
            case .duration:
                result = (a.duration ?? 0) < (b.duration ?? 0)
            }
            return sortOrder == .ascending ? result : !result
        }
    }

    func loadItems() async {
        isLoading = true
        errorMessage = nil
        currentOffset = 0
        hasMoreItems = true

        do {
            let response: PaginatedResponse<Media>
            if mediaType == .movie {
                response = try await api.getMovies(limit: pageSize, offset: 0)
            } else {
                response = try await api.getShows(limit: pageSize, offset: 0)
            }

            items = response.items
            currentOffset = response.items.count
            hasMoreItems = response.items.count >= pageSize
        } catch {
            errorMessage = error.localizedDescription
            let title = mediaType == .movie ? "Failed to Load Movies" : "Failed to Load TV Shows"
            AppState.shared.showError(title, message: error.localizedDescription)
        }

        isLoading = false
    }

    func loadMoreIfNeeded(currentItem: Media) async {
        guard hasMoreItems, !isLoading else { return }

        let thresholdIndex = items.index(items.endIndex, offsetBy: -5)
        guard let itemIndex = items.firstIndex(where: { $0.id == currentItem.id }),
              itemIndex >= thresholdIndex else {
            return
        }

        await loadMore()
    }

    private func loadMore() async {
        isLoading = true

        do {
            let response: PaginatedResponse<Media>
            if mediaType == .movie {
                response = try await api.getMovies(limit: pageSize, offset: currentOffset)
            } else {
                response = try await api.getShows(limit: pageSize, offset: currentOffset)
            }

            items.append(contentsOf: response.items)
            currentOffset += response.items.count
            hasMoreItems = response.items.count >= pageSize
        } catch {
            // Silent fail for pagination
        }

        isLoading = false
    }
}
