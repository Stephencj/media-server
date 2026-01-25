import Foundation

@MainActor
class HomeViewModel: ObservableObject {
    @Published var continueWatching: [ContinueWatchingItem] = []
    @Published var recentlyAdded: [Media] = []
    @Published var movies: [Media] = []
    @Published var tvShows: [Media] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let api = APIClient.shared

    func loadContent() async {
        isLoading = true
        errorMessage = nil

        await withTaskGroup(of: Void.self) { group in
            group.addTask { await self.loadContinueWatching() }
            group.addTask { await self.loadRecentlyAdded() }
            group.addTask { await self.loadMovies() }
            group.addTask { await self.loadTVShows() }
        }

        isLoading = false
    }

    private func loadContinueWatching() async {
        do {
            let response = try await api.getContinueWatching(limit: 10)
            continueWatching = response.items
        } catch {
            // Don't show error for continue watching - it's optional
        }
    }

    private func loadRecentlyAdded() async {
        do {
            let response = try await api.getRecent(limit: 15)
            recentlyAdded = response.items
        } catch {
            errorMessage = error.localizedDescription
            AppState.shared.showError("Failed to Load Recent Media", message: error.localizedDescription)
        }
    }

    private func loadMovies() async {
        do {
            let response = try await api.getMovies(limit: 15, offset: 0)
            movies = response.items
        } catch {
            AppState.shared.showError("Failed to Load Movies", message: error.localizedDescription)
        }
    }

    private func loadTVShows() async {
        do {
            let response = try await api.getShows(limit: 15, offset: 0)
            tvShows = response.items.map { $0.toMedia() }
        } catch {
            AppState.shared.showError("Failed to Load TV Shows", message: error.localizedDescription)
        }
    }
}
