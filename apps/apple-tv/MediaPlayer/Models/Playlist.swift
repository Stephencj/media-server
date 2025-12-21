import Foundation

struct Playlist: Identifiable, Codable, Hashable {
    let id: Int64
    let userId: Int64
    let name: String
    let description: String?
    let itemCount: Int
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id, name, description
        case userId = "user_id"
        case itemCount = "item_count"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct PlaylistItem: Identifiable, Codable, Hashable {
    let id: Int64
    let playlistId: Int64
    let mediaId: Int64
    let mediaType: MediaType
    let position: Int
    let addedAt: String
    let title: String
    let year: Int?
    let posterPath: String?
    let duration: Int?
    let overview: String?
    let rating: Double?
    let resolution: String?

    enum CodingKeys: String, CodingKey {
        case id, position, title, year, duration, overview, rating, resolution
        case playlistId = "playlist_id"
        case mediaId = "media_id"
        case mediaType = "media_type"
        case addedAt = "added_at"
        case posterPath = "poster_path"
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
}

struct PlaylistWithItems: Codable {
    let playlist: Playlist
    let items: [PlaylistItem]
}

struct CreatePlaylistRequest: Codable {
    let name: String
    let description: String?
}

struct ReorderPlaylistRequest: Codable {
    let itemIds: [Int64]

    enum CodingKeys: String, CodingKey {
        case itemIds = "item_ids"
    }
}

struct PlaylistsResponse: Codable {
    let items: [Playlist]
}
