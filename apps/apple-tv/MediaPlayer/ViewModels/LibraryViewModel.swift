import Foundation

@MainActor
class LibraryViewModel: ObservableObject {
    @Published var items: [Media] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let mediaType: MediaType
    private let api = APIClient.shared

    private var currentOffset = 0
    private let pageSize = 50
    private var hasMoreItems = true

    init(mediaType: MediaType) {
        self.mediaType = mediaType
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
