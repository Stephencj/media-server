import SwiftUI

@MainActor
class TVShowDetailViewModel: ObservableObject {
    @Published var seasons: [Season] = []
    @Published var episodes: [Int: [Episode]] = [:] // keyed by season number
    @Published var isLoading = false
    @Published var isLoadingRandom = false
    @Published var selectedSeason = 1
    @Published var errorMessage: String?

    private let showId: Int64
    private let api = APIClient.shared

    init(showId: Int64) {
        self.showId = showId
    }

    func loadSeasons() async {
        isLoading = true
        errorMessage = nil
        do {
            let response = try await api.getSeasons(showId: showId)
            seasons = response.items
            if let first = seasons.first {
                selectedSeason = first.seasonNumber
                await loadEpisodes(for: first.seasonNumber)
            }
        } catch {
            errorMessage = "Failed to load seasons: \(error.localizedDescription)"
        }
        isLoading = false
    }

    func loadEpisodes(for seasonNumber: Int) async {
        // Don't reload if already loaded
        guard episodes[seasonNumber] == nil else { return }

        do {
            let response = try await api.getEpisodes(showId: showId, seasonNumber: seasonNumber)
            episodes[seasonNumber] = response.items
        } catch {
            errorMessage = "Failed to load episodes: \(error.localizedDescription)"
        }
    }

    var currentEpisodes: [Episode] {
        episodes[selectedSeason] ?? []
    }

    func selectSeason(_ seasonNumber: Int) {
        selectedSeason = seasonNumber
        Task {
            await loadEpisodes(for: seasonNumber)
        }
    }

    func playRandomEpisode() async -> Episode? {
        isLoadingRandom = true
        defer { isLoadingRandom = false }

        do {
            let response = try await api.getRandomEpisode(showId: showId)
            return response.episode
        } catch {
            errorMessage = "Failed to get random episode: \(error.localizedDescription)"
            return nil
        }
    }

    func playRandomEpisodeFromSeason() async -> Episode? {
        isLoadingRandom = true
        defer { isLoadingRandom = false }

        do {
            let response = try await api.getRandomEpisodeFromSeason(
                showId: showId,
                seasonNumber: selectedSeason
            )
            return response.episode
        } catch {
            errorMessage = "Failed to get random episode: \(error.localizedDescription)"
            return nil
        }
    }
}
