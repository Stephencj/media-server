import Foundation

struct Channel: Codable, Identifiable, Hashable {
    let id: Int64
    let name: String
    let description: String?
    let icon: String
    let createdAt: String
    let updatedAt: String
    let totalDuration: Int?
    let itemCount: Int?

    enum CodingKeys: String, CodingKey {
        case id, name, description, icon
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case totalDuration = "total_duration"
        case itemCount = "item_count"
    }

    var formattedDuration: String {
        guard let duration = totalDuration, duration > 0 else { return "" }
        let hours = duration / 3600
        let minutes = (duration % 3600) / 60
        if hours > 0 {
            return "\(hours)h \(minutes)m"
        }
        return "\(minutes)m"
    }
}

struct ChannelNowPlayingResponse: Codable {
    let channel: Channel
    let nowPlaying: ChannelScheduleItem?
    let elapsed: Int
    let upNext: [ChannelScheduleItem]
    let cycleStart: String?
    let streamUrl: String?

    enum CodingKeys: String, CodingKey {
        case channel, elapsed
        case nowPlaying = "now_playing"
        case upNext = "up_next"
        case cycleStart = "cycle_start"
        case streamUrl = "stream_url"
    }
}

struct ChannelScheduleItem: Codable, Identifiable, Hashable {
    let id: Int64
    let mediaId: Int64
    let mediaType: String
    let title: String
    let duration: Int
    let cumulativeStart: Int?
    let posterPath: String?
    let backdropPath: String?
    let showTitle: String?

    enum CodingKeys: String, CodingKey {
        case id, title, duration
        case mediaId = "media_id"
        case mediaType = "media_type"
        case cumulativeStart = "cumulative_start"
        case posterPath = "poster_path"
        case backdropPath = "backdrop_path"
        case showTitle = "show_title"
    }

    var formattedDuration: String {
        let hours = duration / 3600
        let minutes = (duration % 3600) / 60
        if hours > 0 {
            return "\(hours)h \(minutes)m"
        }
        return "\(minutes)m"
    }

    func toMedia() -> Media {
        let type: MediaType = mediaType == "episode" ? .episode : .movie
        return Media(
            id: mediaId,
            title: title,
            originalTitle: nil,
            type: type,
            year: nil,
            overview: nil,
            posterPath: posterPath,
            backdropPath: backdropPath,
            rating: nil,
            runtime: nil,
            genres: nil,
            tmdbId: nil,
            imdbId: nil,
            seasonCount: nil,
            episodeCount: nil,
            sourceId: nil,
            filePath: nil,
            fileSize: nil,
            duration: duration,
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

struct ChannelsResponse: Codable {
    let items: [Channel]
}

struct ChannelScheduleResponse: Codable {
    let items: [ChannelScheduleItem]
    let total: Int
    let limit: Int
    let offset: Int
}
