import Foundation

struct Channel: Codable, Identifiable, Hashable {
    let id: Int64
    let name: String
    let description: String?
    let icon: String
    /// Guide number (101 upward). Optional so the app still decodes against a
    /// server that predates the guide endpoint.
    let channelNumber: Int?
    let createdAt: String
    let updatedAt: String
    let totalDuration: Int?
    let itemCount: Int?

    enum CodingKeys: String, CodingKey {
        case id, name, description, icon
        case channelNumber = "channel_number"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case totalDuration = "total_duration"
        case itemCount = "item_count"
    }

    /// What the guide rail prints. Falls back to a stable synthetic number so
    /// the column is never blank against an older server.
    var displayNumber: Int {
        if let channelNumber, channelNumber > 0 { return channelNumber }
        return 100 + Int(id)
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
        let type: MediaType
        switch mediaType {
        case "episode":
            type = .episode
        case "extra":
            type = .extra
        default:
            type = .movie
        }
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


// MARK: - Guide

/// A schedule item projected onto wall-clock time by `/api/channels/guide`.
struct GuideProgram: Codable, Identifiable, Hashable {
    let id: Int64
    let mediaId: Int64
    let mediaType: String
    let title: String
    let duration: Int
    let posterPath: String?
    let backdropPath: String?
    let showTitle: String?
    let startTime: Date
    let endTime: Date

    enum CodingKeys: String, CodingKey {
        case id, title, duration
        case mediaId = "media_id"
        case mediaType = "media_type"
        case posterPath = "poster_path"
        case backdropPath = "backdrop_path"
        case showTitle = "show_title"
        case startTime = "start_time"
        case endTime = "end_time"
    }

    /// The same schedule row airs more than once in a window, so identity for
    /// SwiftUI has to include the air time.
    var airingId: String { "\(id)-\(startTime.timeIntervalSince1970)" }

    func isAiring(at date: Date) -> Bool {
        startTime <= date && date < endTime
    }

    func progress(at date: Date) -> Double {
        let total = endTime.timeIntervalSince(startTime)
        guard total > 0 else { return 0 }
        return min(1, max(0, date.timeIntervalSince(startTime) / total))
    }

    func remaining(at date: Date) -> Int {
        max(0, Int(endTime.timeIntervalSince(date)))
    }

    var formattedDuration: String {
        let hours = duration / 3600
        let minutes = (duration % 3600) / 60
        if hours > 0 { return minutes > 0 ? "\(hours)h \(minutes)m" : "\(hours)h" }
        return "\(minutes)m"
    }

    func toMedia() -> Media {
        let type: MediaType
        switch mediaType {
        case "episode": type = .episode
        case "extra": type = .extra
        default: type = .movie
        }
        return Media(
            id: mediaId, title: title, originalTitle: nil, type: type, year: nil,
            overview: nil, posterPath: posterPath, backdropPath: backdropPath,
            rating: nil, runtime: nil, genres: nil, tmdbId: nil, imdbId: nil,
            seasonCount: nil, episodeCount: nil, sourceId: nil, filePath: nil,
            fileSize: nil, duration: duration, videoCodec: nil, audioCodec: nil,
            resolution: nil, audioTracks: nil, subtitleTracks: nil,
            createdAt: nil, updatedAt: nil
        )
    }
}

extension GuideProgram {
    /// The same programme as the channel player plays it.
    var scheduleItem: ChannelScheduleItem {
        ChannelScheduleItem(
            id: id, mediaId: mediaId, mediaType: mediaType, title: title, duration: duration,
            cumulativeStart: nil, posterPath: posterPath, backdropPath: backdropPath, showTitle: showTitle
        )
    }
}

struct ChannelGuideEntry: Codable, Identifiable, Hashable {
    let channel: Channel
    let programs: [GuideProgram]

    var id: Int64 { channel.id }

    func program(airingAt date: Date) -> GuideProgram? {
        programs.first { $0.isAiring(at: date) }
    }

    func program(after date: Date) -> GuideProgram? {
        programs.first { $0.startTime > date }
    }
}

struct ChannelGuideResponse: Codable {
    let start: Date
    let end: Date
    let channels: [ChannelGuideEntry]
}
