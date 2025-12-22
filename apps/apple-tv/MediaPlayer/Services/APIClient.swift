import Foundation

enum APIError: Error, LocalizedError {
    case invalidURL
    case noData
    case decodingError(Error)
    case serverError(Int, String)
    case networkError(Error)
    case unauthorized

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid URL"
        case .noData:
            return "No data received"
        case .decodingError(let error):
            return "Failed to decode response: \(error.localizedDescription)"
        case .serverError(let code, let message):
            return "Server error (\(code)): \(message)"
        case .networkError(let error):
            return "Network error: \(error.localizedDescription)"
        case .unauthorized:
            return "Unauthorized - please log in again"
        }
    }
}

@MainActor
class APIClient: ObservableObject {
    static let shared = APIClient()

    private var baseURL: String {
        AppState.shared.serverURL
    }

    private var authToken: String? {
        AuthService.shared.token
    }

    private let session: URLSession
    private let decoder: JSONDecoder
    private var mediaCache: [Media] = []
    private var lastCacheUpdate: Date?

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 300
        self.session = URLSession(configuration: config)

        self.decoder = JSONDecoder()
    }

    // MARK: - Generic Request Methods

    func get<T: Decodable>(_ endpoint: String, authenticated: Bool = true) async throws -> T {
        try await request(endpoint, method: "GET", body: nil as String?, authenticated: authenticated)
    }

    func post<T: Decodable, B: Encodable>(_ endpoint: String, body: B, authenticated: Bool = true) async throws -> T {
        try await request(endpoint, method: "POST", body: body, authenticated: authenticated)
    }

    func delete(_ endpoint: String, authenticated: Bool = true) async throws {
        let _: EmptyResponse = try await request(endpoint, method: "DELETE", body: nil as String?, authenticated: authenticated)
    }

    private func request<T: Decodable, B: Encodable>(
        _ endpoint: String,
        method: String,
        body: B?,
        authenticated: Bool
    ) async throws -> T {
        guard let url = URL(string: baseURL + endpoint) else {
            throw APIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if authenticated, let token = authToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body = body {
            request.httpBody = try JSONEncoder().encode(body)
        }

        do {
            let (data, response) = try await session.data(for: request)

            guard let httpResponse = response as? HTTPURLResponse else {
                throw APIError.networkError(URLError(.badServerResponse))
            }

            if httpResponse.statusCode == 401 {
                throw APIError.unauthorized
            }

            if httpResponse.statusCode >= 400 {
                let errorMessage = String(data: data, encoding: .utf8) ?? "Unknown error"
                throw APIError.serverError(httpResponse.statusCode, errorMessage)
            }

            // Handle empty responses
            if data.isEmpty || T.self == EmptyResponse.self {
                if let empty = EmptyResponse() as? T {
                    return empty
                }
            }

            do {
                return try decoder.decode(T.self, from: data)
            } catch {
                throw APIError.decodingError(error)
            }
        } catch let error as APIError {
            throw error
        } catch {
            throw APIError.networkError(error)
        }
    }

    // MARK: - Media Cache

    func getCachedMedia() async -> [Media] {
        // Refresh cache if older than 5 minutes or empty
        if mediaCache.isEmpty || (lastCacheUpdate?.addingTimeInterval(300) ?? .distantPast) < Date() {
            do {
                // Fetch all media in parallel
                async let movies = getMovies(limit: 1000, offset: 0)
                async let shows = getShows(limit: 1000, offset: 0)

                let (moviesResponse, showsResponse) = try await (movies, shows)
                mediaCache = moviesResponse.items + showsResponse.items
                lastCacheUpdate = Date()
            } catch {
                // Return existing cache on error
                return mediaCache
            }
        }
        return mediaCache
    }

    // MARK: - Library Endpoints

    func getMovies(limit: Int = 50, offset: Int = 0) async throws -> PaginatedResponse<Media> {
        try await get("/api/library/movies?limit=\(limit)&offset=\(offset)")
    }

    func getShows(limit: Int = 50, offset: Int = 0) async throws -> PaginatedResponse<Media> {
        try await get("/api/library/shows?limit=\(limit)&offset=\(offset)")
    }

    func getRecent(limit: Int = 20) async throws -> ItemsResponse<Media> {
        try await get("/api/library/recent?limit=\(limit)")
    }

    func getMedia(id: Int64) async throws -> Media {
        try await get("/api/media/\(id)")
    }

    func triggerScan() async throws -> ScanResponse {
        try await post("/api/library/scan", body: EmptyBody())
    }

    // MARK: - Streaming Endpoints

    func getStreamURL(mediaId: Int64) -> URL? {
        guard let token = authToken else { return nil }
        return URL(string: "\(baseURL)/api/stream/\(mediaId)/manifest.m3u8?token=\(token)")
    }

    func getDirectPlayURL(mediaId: Int64) -> URL? {
        guard let token = authToken else { return nil }
        return URL(string: "\(baseURL)/api/stream/\(mediaId)/direct?token=\(token)")
    }

    func getSubtitleURL(mediaId: Int64, language: String) -> URL? {
        guard let token = authToken else { return nil }
        return URL(string: "\(baseURL)/api/stream/\(mediaId)/subtitles/\(language).vtt?token=\(token)")
    }

    // MARK: - Progress Endpoints

    func getProgress(mediaId: Int64, type: String = "movie") async throws -> WatchProgress {
        try await get("/api/progress/\(mediaId)?type=\(type)")
    }

    func updateProgress(mediaId: Int64, position: Int, duration: Int, mediaType: String, completed: Bool = false) async throws -> WatchProgress {
        let request = UpdateProgressRequest(
            position: position,
            duration: duration,
            mediaType: mediaType,
            completed: completed
        )
        return try await post("/api/progress/\(mediaId)", body: request)
    }

    func getContinueWatching(limit: Int = 10) async throws -> ContinueWatchingResponse {
        try await get("/api/continue-watching?limit=\(limit)")
    }

    // MARK: - Sources Endpoints

    func getSources() async throws -> SourcesResponse {
        try await get("/api/sources")
    }

    func createSource(name: String, path: String, type: String) async throws -> MediaSource {
        let request = CreateSourceRequest(name: name, path: path, type: type)
        return try await post("/api/sources", body: request)
    }

    func deleteSource(id: Int64) async throws {
        try await delete("/api/sources/\(id)")
    }

    // MARK: - Watchlist Endpoints

    func getWatchlist(limit: Int = 50) async throws -> ItemsResponse<Media> {
        try await get("/api/watchlist?limit=\(limit)")
    }

    func addToWatchlist(mediaId: Int64, mediaType: String) async throws {
        let body = ["media_type": mediaType]
        let _: MessageResponse = try await post("/api/watchlist/\(mediaId)", body: body)
    }

    func removeFromWatchlist(mediaId: Int64, mediaType: String) async throws {
        try await delete("/api/watchlist/\(mediaId)?type=\(mediaType)")
    }

    func markAsWatched(mediaId: Int64, mediaType: String) async throws {
        let body = ["media_type": mediaType]
        let _: MessageResponse = try await post("/api/media/\(mediaId)/watched", body: body)
    }

    // MARK: - Playlist Endpoints

    func getPlaylists() async throws -> PlaylistsResponse {
        try await get("/api/playlists")
    }

    func getPlaylist(id: Int64) async throws -> PlaylistWithItems {
        try await get("/api/playlists/\(id)")
    }

    func createPlaylist(name: String, description: String?) async throws -> Playlist {
        let request = CreatePlaylistRequest(name: name, description: description)
        return try await post("/api/playlists", body: request)
    }

    func updatePlaylist(id: Int64, name: String, description: String?) async throws {
        let body = CreatePlaylistRequest(name: name, description: description)
        let _: MessageResponse = try await self.request("/api/playlists/\(id)", method: "PUT", body: body, authenticated: true)
    }

    func deletePlaylist(id: Int64) async throws {
        try await delete("/api/playlists/\(id)")
    }

    func addToPlaylist(playlistId: Int64, mediaId: Int64, mediaType: String) async throws {
        let _: MessageResponse = try await post("/api/playlists/\(playlistId)/items/\(mediaId)?type=\(mediaType)", body: EmptyBody())
    }

    func removeFromPlaylist(playlistId: Int64, mediaId: Int64, mediaType: String) async throws {
        try await delete("/api/playlists/\(playlistId)/items/\(mediaId)?type=\(mediaType)")
    }

    func reorderPlaylist(playlistId: Int64, itemIds: [Int64]) async throws {
        let body = ReorderPlaylistRequest(itemIds: itemIds)
        let _: MessageResponse = try await self.request("/api/playlists/\(playlistId)/reorder", method: "PUT", body: body, authenticated: true)
    }

    // MARK: - TV Show Endpoints

    func getSeasons(showId: Int64) async throws -> ItemsResponse<Season> {
        try await get("/api/shows/\(showId)/seasons")
    }

    func getEpisodes(showId: Int64, seasonNumber: Int) async throws -> ItemsResponse<Episode> {
        try await get("/api/shows/\(showId)/seasons/\(seasonNumber)/episodes")
    }

    func getRandomEpisode(showId: Int64) async throws -> RandomEpisodeResponse {
        try await get("/api/shows/\(showId)/random")
    }

    func getRandomEpisodeFromSeason(showId: Int64, seasonNumber: Int) async throws -> RandomEpisodeResponse {
        try await get("/api/shows/\(showId)/seasons/\(seasonNumber)/random")
    }
}

struct MessageResponse: Codable {
    let message: String
}

// MARK: - Response Types

struct EmptyResponse: Codable {}
struct EmptyBody: Codable {}

struct PaginatedResponse<T: Codable>: Codable {
    let items: [T]
    let total: Int?
    let limit: Int
    let offset: Int
}

struct ItemsResponse<T: Codable>: Codable {
    let items: [T]
}

struct ScanResponse: Codable {
    let message: String
    let status: String
}

struct MediaSource: Codable, Identifiable {
    let id: Int64
    let name: String
    let path: String
    let type: String
    let enabled: Bool
    let lastScan: String?

    enum CodingKeys: String, CodingKey {
        case id, name, path, type, enabled
        case lastScan = "last_scan"
    }
}

struct SourcesResponse: Codable {
    let sources: [MediaSource]
}

struct CreateSourceRequest: Codable {
    let name: String
    let path: String
    let type: String
}
