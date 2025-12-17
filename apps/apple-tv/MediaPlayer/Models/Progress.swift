import Foundation

struct WatchProgress: Codable, Identifiable {
    let id: Int64?
    let userId: Int64?
    let mediaId: Int64
    let mediaType: String
    let position: Int
    let duration: Int
    let completed: Bool
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case id, position, duration, completed
        case userId = "user_id"
        case mediaId = "media_id"
        case mediaType = "media_type"
        case updatedAt = "updated_at"
    }

    var progressPercentage: Double {
        guard duration > 0 else { return 0 }
        return Double(position) / Double(duration)
    }

    var formattedPosition: String {
        formatTime(position)
    }

    var formattedRemaining: String {
        let remaining = duration - position
        return formatTime(remaining)
    }

    private func formatTime(_ seconds: Int) -> String {
        let hours = seconds / 3600
        let minutes = (seconds % 3600) / 60
        let secs = seconds % 60

        if hours > 0 {
            return String(format: "%d:%02d:%02d", hours, minutes, secs)
        }
        return String(format: "%d:%02d", minutes, secs)
    }
}

struct ContinueWatchingItem: Codable, Identifiable {
    let media: Media
    let progress: WatchProgress

    var id: Int64 { media.id }
}

struct ContinueWatchingResponse: Codable {
    let items: [ContinueWatchingItem]
}

struct UpdateProgressRequest: Codable {
    let position: Int
    let duration: Int
    let mediaType: String
    let completed: Bool

    enum CodingKeys: String, CodingKey {
        case position, duration, completed
        case mediaType = "media_type"
    }
}
