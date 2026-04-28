import Foundation

enum MediaType: String, Codable {
    case movie
    case tvshow
    case episode
    case extra
}

struct Media: Identifiable, Codable, Hashable {
    let id: Int64
    let title: String
    let originalTitle: String?
    let type: MediaType
    let year: Int?
    let overview: String?
    let posterPath: String?
    let backdropPath: String?
    let rating: Double?
    let runtime: Int?
    let genres: String?
    let tmdbId: Int?
    let imdbId: String?
    let seasonCount: Int?
    let episodeCount: Int?
    let sourceId: Int64?
    let filePath: String?
    let fileSize: Int64?
    let duration: Int?
    let videoCodec: String?
    let audioCodec: String?
    let resolution: String?
    let audioTracks: String?
    let subtitleTracks: String?
    let createdAt: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id, title, type, year, overview, rating, runtime, genres, duration
        case originalTitle = "original_title"
        case posterPath = "poster_path"
        case backdropPath = "backdrop_path"
        case tmdbId = "tmdb_id"
        case imdbId = "imdb_id"
        case seasonCount = "season_count"
        case episodeCount = "episode_count"
        case sourceId = "source_id"
        case filePath = "file_path"
        case fileSize = "file_size"
        case videoCodec = "video_codec"
        case audioCodec = "audio_codec"
        case resolution
        case audioTracks = "audio_tracks"
        case subtitleTracks = "subtitle_tracks"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    var formattedDuration: String {
        guard let duration = duration else { return "" }
        let hours = duration / 3600
        let minutes = (duration % 3600) / 60
        if hours > 0 {
            return "\(hours)h \(minutes)m"
        }
        return "\(minutes)m"
    }

    var genreList: [String] {
        genres?.split(separator: ",").map { String($0).trimmingCharacters(in: .whitespaces) } ?? []
    }

    var decodedAudioTracks: [AudioTrack] {
        guard let data = audioTracks?.data(using: .utf8) else { return [] }
        return (try? JSONDecoder().decode([AudioTrack].self, from: data)) ?? []
    }

    var decodedSubtitleTracks: [SubtitleTrack] {
        guard let data = subtitleTracks?.data(using: .utf8) else { return [] }
        return (try? JSONDecoder().decode([SubtitleTrack].self, from: data)) ?? []
    }
}

struct AudioTrack: Codable, Identifiable, Hashable {
    let index: Int
    let language: String
    let codec: String
    let channels: Int
    let title: String?

    var id: Int { index }

    var displayName: String {
        if let title = title, !title.isEmpty {
            return title
        }
        let langName = Locale.current.localizedString(forLanguageCode: language) ?? language
        return "\(langName) (\(codec.uppercased()))"
    }
}

struct SubtitleTrack: Codable, Identifiable, Hashable {
    let index: Int
    let language: String
    let codec: String
    let title: String?
    let forced: Bool?

    var id: Int { index }

    var displayName: String {
        if let title = title, !title.isEmpty {
            return title
        }
        let langName = Locale.current.localizedString(forLanguageCode: language) ?? language
        var name = langName
        if forced == true {
            name += " (Forced)"
        }
        return name
    }
}

struct TVShow: Identifiable, Codable {
    let id: Int64
    let title: String
    let originalTitle: String?
    let year: Int?
    let overview: String?
    let posterPath: String?
    let backdropPath: String?
    let rating: Double?
    let genres: String?
    let tmdbId: Int?
    let imdbId: String?
    let status: String?

    enum CodingKeys: String, CodingKey {
        case id, title, year, overview, rating, genres, status
        case originalTitle = "original_title"
        case posterPath = "poster_path"
        case backdropPath = "backdrop_path"
        case tmdbId = "tmdb_id"
        case imdbId = "imdb_id"
    }
}

struct Season: Identifiable, Codable {
    let id: Int64
    let tvShowId: Int64
    let seasonNumber: Int
    let name: String?
    let overview: String?
    let posterPath: String?
    let airDate: String?
    let episodeCount: Int

    enum CodingKeys: String, CodingKey {
        case id, name, overview
        case tvShowId = "tv_show_id"
        case seasonNumber = "season_number"
        case posterPath = "poster_path"
        case airDate = "air_date"
        case episodeCount = "episode_count"
    }
}

struct Episode: Identifiable, Codable {
    let id: Int64
    let tvShowId: Int64
    let seasonId: Int64
    let seasonNumber: Int
    let episodeNumber: Int
    let title: String
    let overview: String?
    let stillPath: String?
    let airDate: String?
    let runtime: Int?
    let rating: Double?
    let filePath: String?
    let duration: Int?

    enum CodingKeys: String, CodingKey {
        case id, title, overview, runtime, rating, duration
        case tvShowId = "tv_show_id"
        case seasonId = "season_id"
        case seasonNumber = "season_number"
        case episodeNumber = "episode_number"
        case stillPath = "still_path"
        case airDate = "air_date"
        case filePath = "file_path"
    }

    var episodeCode: String {
        String(format: "S%02dE%02d", seasonNumber, episodeNumber)
    }
}

struct RandomEpisodeResponse: Codable {
    let episode: Episode
    let showTitle: String

    enum CodingKeys: String, CodingKey {
        case episode
        case showTitle = "show_title"
    }
}

extension TVShow {
    func toMedia() -> Media {
        Media(
            id: id,
            title: title,
            originalTitle: originalTitle,
            type: .tvshow,
            year: year,
            overview: overview,
            posterPath: posterPath,
            backdropPath: backdropPath,
            rating: rating,
            runtime: nil,
            genres: genres,
            tmdbId: tmdbId,
            imdbId: imdbId,
            seasonCount: nil,
            episodeCount: nil,
            sourceId: nil,
            filePath: nil,
            fileSize: nil,
            duration: nil,
            videoCodec: nil,
            audioCodec: nil,
            resolution: nil,
            audioTracks: nil,
            subtitleTracks: nil,
            createdAt: nil,
            updatedAt: nil
        )
    }
}
