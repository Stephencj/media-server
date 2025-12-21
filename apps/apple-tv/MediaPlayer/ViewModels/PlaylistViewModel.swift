import Foundation

@MainActor
class PlaylistViewModel: ObservableObject {
    @Published var playlists: [Playlist] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let api = APIClient.shared

    func loadPlaylists() async {
        isLoading = true
        errorMessage = nil

        do {
            let response = try await api.getPlaylists()
            playlists = response.items
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func createPlaylist(name: String, description: String?) async -> Bool {
        do {
            let playlist = try await api.createPlaylist(name: name, description: description)
            playlists.insert(playlist, at: 0)
            return true
        } catch {
            errorMessage = error.localizedDescription
            return false
        }
    }

    func deletePlaylist(_ playlist: Playlist) async {
        do {
            try await api.deletePlaylist(id: playlist.id)
            playlists.removeAll { $0.id == playlist.id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

@MainActor
class PlaylistDetailViewModel: ObservableObject {
    @Published var playlist: Playlist?
    @Published var items: [PlaylistItem] = []
    @Published var isLoading = false
    @Published var errorMessage: String?

    // Playback state
    @Published var isPlaying = false
    @Published var currentIndex = 0

    private let api = APIClient.shared

    func loadPlaylist(id: Int64) async {
        isLoading = true
        errorMessage = nil

        do {
            let response = try await api.getPlaylist(id: id)
            playlist = response.playlist
            items = response.items
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func removeItem(_ item: PlaylistItem) async {
        guard let playlist = playlist else { return }

        do {
            try await api.removeFromPlaylist(
                playlistId: playlist.id,
                mediaId: item.mediaId,
                mediaType: item.mediaType.rawValue
            )
            items.removeAll { $0.id == item.id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func moveItem(from source: IndexSet, to destination: Int) {
        items.move(fromOffsets: source, toOffset: destination)
        Task {
            await saveOrder()
        }
    }

    private func saveOrder() async {
        guard let playlist = playlist else { return }

        let itemIds = items.map { $0.id }
        do {
            try await api.reorderPlaylist(playlistId: playlist.id, itemIds: itemIds)
        } catch {
            errorMessage = error.localizedDescription
            // Reload to get correct order
            await loadPlaylist(id: playlist.id)
        }
    }

    func playAll() {
        guard !items.isEmpty else { return }
        currentIndex = 0
        isPlaying = true
    }

    func shuffle() {
        guard !items.isEmpty else { return }
        items.shuffle()
        currentIndex = 0
        isPlaying = true
    }

    func playItem(at index: Int) {
        guard index >= 0 && index < items.count else { return }
        currentIndex = index
        isPlaying = true
    }

    func playNext() -> Bool {
        if currentIndex < items.count - 1 {
            currentIndex += 1
            return true
        }
        isPlaying = false
        return false
    }

    var currentItem: PlaylistItem? {
        guard currentIndex >= 0 && currentIndex < items.count else { return nil }
        return items[currentIndex]
    }

    var hasNextItem: Bool {
        currentIndex < items.count - 1
    }

    var nextItem: PlaylistItem? {
        guard hasNextItem else { return nil }
        return items[currentIndex + 1]
    }
}
