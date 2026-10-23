import Foundation

// MARK: - Models

/// A commentary track or bonus clip belonging to a TV show.
struct ShowExtra: Codable, Identifiable, Hashable {
    let id: Int64
    let title: String
    let category: String
    let duration: Int?
    let resolution: String?
    let fileSize: Int64?
    let seasonNumber: Int?
    let episodeNumber: Int?

    enum CodingKeys: String, CodingKey {
        case id, title, category, duration, resolution
        case fileSize = "file_size"
        case seasonNumber = "season_number"
        case episodeNumber = "episode_number"
    }

    var isCommentary: Bool { category == "commentary" }

    /// "1h 46m", or "4m 12s" for the short ones.
    var formattedDuration: String {
        guard let duration, duration > 0 else { return "" }
        if duration >= 3600 {
            return "\(duration / 3600)h \((duration % 3600) / 60)m"
        }
        return duration >= 600 ? "\(duration / 60)m" : "\(duration / 60)m \(duration % 60)s"
    }

    /// Episode label for commentaries, e.g. "S01E02".
    var episodeLabel: String? {
        guard let s = seasonNumber, let e = episodeNumber else { return nil }
        return String(format: "S%02dE%02d", s, e)
    }

    /// The player takes a `Media`; an extra plays as one of type `.extra`.
    func toMedia() -> Media {
        Media(
            id: id, title: title, originalTitle: nil, type: .extra, year: nil,
            overview: nil, posterPath: nil, backdropPath: nil,
            rating: nil, runtime: nil, genres: nil, tmdbId: nil, imdbId: nil,
            seasonCount: nil, episodeCount: nil, sourceId: nil, filePath: nil,
            fileSize: fileSize, duration: duration, videoCodec: nil, audioCodec: nil,
            resolution: resolution, audioTracks: nil, subtitleTracks: nil,
            createdAt: nil, updatedAt: nil
        )
    }
}

/// Bonus clips of one kind, in the order they are shown.
struct ShowExtraGroup: Identifiable {
    let title: String
    let extras: [ShowExtra]
    var id: String { title }
}

// MARK: - API

extension APIClient {
    func getShowExtras(showId: Int64) async throws -> [ShowExtra] {
        let response: ItemsResponse<ShowExtra> = try await get("/api/shows/\(showId)/extras")
        return response.items
    }
}

// MARK: - View model

/// Extras for a TV show: commentary tracks (grouped by episode) and bonus
/// clips by kind.
@MainActor
class ShowBonusViewModel: ObservableObject {
    @Published var commentaries: [ShowExtra] = []
    @Published var groups: [ShowExtraGroup] = []

    let showId: Int64
    private let api = APIClient.shared

    private static let groupOrder: [(category: String, title: String)] = [
        ("deleted_scene", "Deleted Scenes"),
        ("featurette", "Featurettes"),
        ("behind_the_scenes", "Behind the Scenes"),
        ("interview", "Interviews"),
        ("gag_reel", "Gag Reels"),
        ("music_video", "Music Videos"),
        ("other", "More"),
    ]

    init(showId: Int64) {
        self.showId = showId
    }

    var hasContent: Bool {
        !commentaries.isEmpty || !groups.isEmpty
    }

    func load() async {
        let all = (try? await api.getShowExtras(showId: showId)) ?? []
        commentaries = all.filter(\.isCommentary).sorted {
            // Sort by season, then episode
            if $0.seasonNumber != $1.seasonNumber {
                return ($0.seasonNumber ?? 0) < ($1.seasonNumber ?? 0)
            }
            return ($0.episodeNumber ?? 0) < ($1.episodeNumber ?? 0)
        }

        let known = Set(Self.groupOrder.map(\.category))
        groups = Self.groupOrder.compactMap { entry in
            let matching = all.filter { extra in
                !extra.isCommentary &&
                    (extra.category == entry.category || (entry.category == "other" && !known.contains(extra.category)))
            }
            return matching.isEmpty ? nil : ShowExtraGroup(title: entry.title, extras: matching)
        }
    }
}
