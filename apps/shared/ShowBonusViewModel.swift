import Foundation

// Loading a show's extras for its page, built into both the iPhone and
// Apple TV apps, which have the same APIClient and Media. The rules for
// laying extras out are in ShowExtras.swift beside this.

extension ShowExtra {
    /// The player takes a `Media`; an extra plays as one of type `.extra`.
    func toMedia() -> Media {
        Media(
            id: id, title: displayTitle, originalTitle: nil, type: .extra, year: nil,
            overview: nil, posterPath: nil, backdropPath: nil,
            rating: nil, runtime: nil, genres: nil, tmdbId: nil, imdbId: nil,
            seasonCount: nil, episodeCount: nil, sourceId: nil, filePath: nil,
            fileSize: fileSize, duration: duration, videoCodec: nil, audioCodec: nil,
            resolution: resolution, audioTracks: nil, subtitleTracks: nil,
            createdAt: nil, updatedAt: nil
        )
    }
}

extension APIClient {
    func getShowExtras(showId: Int64) async throws -> [ShowExtra] {
        let response: ItemsResponse<ShowExtra> = try await get("/api/shows/\(showId)/extras")
        return response.items
    }
}

/// A TV show's commentaries and bonus clips, loaded once for its page.
@MainActor
class ShowBonusViewModel: ObservableObject {
    @Published private(set) var all: [ShowExtra] = []

    let showId: Int64
    private let api = APIClient.shared

    init(showId: Int64) {
        self.showId = showId
    }

    func load() async {
        all = (try? await api.getShowExtras(showId: showId)) ?? []
    }

    func season(_ number: Int) -> [ShowExtraGroup] { ShowExtras.season(number, of: all) }
    var wholeShow: [ShowExtraGroup] { ShowExtras.wholeShow(all) }
    func commentaries(season: Int, episode: Int) -> [ShowExtra] {
        ShowExtras.commentaries(season: season, episode: episode, of: all)
    }
}
