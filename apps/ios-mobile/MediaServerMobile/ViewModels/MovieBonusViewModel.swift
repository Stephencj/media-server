import Foundation

// MARK: - Models

/// A commentary track or bonus clip belonging to a film.
struct MovieExtra: Codable, Identifiable, Hashable {
    let id: Int64
    let title: String
    let category: String
    let duration: Int?
    let resolution: String?
    let fileSize: Int64?

    enum CodingKeys: String, CodingKey {
        case id, title, category, duration, resolution
        case fileSize = "file_size"
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
struct MovieExtraGroup: Identifiable {
    let title: String
    let extras: [MovieExtra]
    var id: String { title }
}

// MARK: - API

extension APIClient {
    func getMovieExtras(mediaId: Int64) async throws -> [MovieExtra] {
        let response: ItemsResponse<MovieExtra> = try await get("/api/media/\(mediaId)/extras")
        return response.items
    }

    /// Other files the library holds of the same film.
    func getMovieCopies(mediaId: Int64) async throws -> [Media] {
        let response: ItemsResponse<Media> = try await get("/api/media/\(mediaId)/copies")
        return response.items
    }
}

// MARK: - View model

/// Everything kept with a film besides the film: commentary tracks, bonus
/// clips by kind, and other copies of the film itself.
@MainActor
class MovieBonusViewModel: ObservableObject {
    @Published var commentaries: [MovieExtra] = []
    @Published var groups: [MovieExtraGroup] = []
    @Published var copies: [Media] = []

    let mediaId: Int64
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

    init(mediaId: Int64) {
        self.mediaId = mediaId
    }

    func load() async {
        // Each is optional decoration on the page; a failure just leaves its
        // section out.
        async let extras = try? api.getMovieExtras(mediaId: mediaId)
        async let others = try? api.getMovieCopies(mediaId: mediaId)

        let all = await extras ?? []
        commentaries = all.filter(\.isCommentary)

        let known = Set(Self.groupOrder.map(\.category))
        groups = Self.groupOrder.compactMap { entry in
            let matching = all.filter { extra in
                !extra.isCommentary &&
                    (extra.category == entry.category || (entry.category == "other" && !known.contains(extra.category)))
            }
            return matching.isEmpty ? nil : MovieExtraGroup(title: entry.title, extras: matching)
        }

        copies = await others ?? []
    }

    /// "1080p · 1h 44m · 2.1 GB"
    static func describe(copy: Media) -> String {
        var parts: [String] = []
        if let resolution = copy.resolution, !resolution.isEmpty { parts.append(resolution) }
        if !copy.formattedDuration.isEmpty { parts.append(copy.formattedDuration) }
        if let size = copy.fileSize, size > 0 {
            parts.append(ByteCountFormatter.string(fromByteCount: size, countStyle: .file))
        }
        return parts.joined(separator: " · ")
    }

    static func fileName(of copy: Media) -> String {
        guard let path = copy.filePath else { return copy.title }
        return (path as NSString).lastPathComponent
    }
}
