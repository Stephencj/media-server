import Foundation

// MARK: - Models

/// One side of a match comparison: the film a movie is shown as, the film
/// suggested for it, or a candidate to choose.
struct MatchFilm: Codable, Hashable {
    let tmdbId: Int
    let title: String
    let year: Int
    /// Minutes. 0 when TMDB has none recorded.
    let runtime: Int
    let posterPath: String?

    enum CodingKeys: String, CodingKey {
        case title, year, runtime
        case tmdbId = "tmdb_id"
        case posterPath = "poster_path"
    }

    /// "The Kid (1921)"
    var titleWithYear: String {
        year > 0 ? "\(title) (\(String(year)))" : title
    }
}

/// A movie whose match looked wrong when the library was last checked.
struct MatchReviewItem: Codable, Identifiable, Hashable {
    let mediaId: Int64
    let fileName: String
    let fileTitle: String
    let fileYear: Int
    let fileRuntime: Int
    /// Nil when the movie has no match yet.
    let current: MatchFilm?
    /// Nil when the suggestion is to drop the match and show the file's name.
    let suggested: MatchFilm?
    let reason: String

    var id: Int64 { mediaId }

    enum CodingKeys: String, CodingKey {
        case current, suggested, reason
        case mediaId = "media_id"
        case fileName = "file_name"
        case fileTitle = "file_title"
        case fileYear = "file_year"
        case fileRuntime = "file_runtime"
    }

    /// "Named 1921 · runs 1h 8m"
    var fileFacts: String {
        var parts: [String] = []
        parts.append(fileYear > 0 ? "Named \(String(fileYear))" : "No year in the name")
        if fileRuntime > 0 { parts.append("runs \(MatchFormat.minutes(fileRuntime))") }
        return parts.joined(separator: " · ")
    }
}

struct MatchCheckStatus: Codable {
    let running: Bool
    let checked: Int
    let total: Int
    let finishedAt: String?
    let error: String?

    enum CodingKeys: String, CodingKey {
        case running, checked, total, error
        case finishedAt = "finished_at"
    }
}

struct MatchReviewResponse: Codable {
    let status: MatchCheckStatus
    let items: [MatchReviewItem]
}

/// A film a movie could be, with how well it fits the file.
struct MatchCandidate: Codable, Identifiable, Hashable {
    let tmdbId: Int
    let title: String
    let year: Int
    let runtime: Int
    let posterPath: String?
    let overview: String?
    /// "Same year, same running time", "79 years off, 36 min longer"
    let fit: String
    let isCurrent: Bool
    let isBestFit: Bool

    var id: Int { tmdbId }

    enum CodingKeys: String, CodingKey {
        case title, year, runtime, overview, fit
        case tmdbId = "tmdb_id"
        case posterPath = "poster_path"
        case isCurrent = "is_current"
        case isBestFit = "is_best_fit"
    }

    /// "1921 · 1h 8m"
    var facts: String {
        var parts: [String] = []
        if year > 0 { parts.append(String(year)) }
        if runtime > 0 { parts.append(MatchFormat.minutes(runtime)) }
        return parts.joined(separator: " · ")
    }
}

struct MatchCandidatesResponse: Codable {
    let query: String
    let fileName: String
    let fileYear: Int
    let fileRuntime: Int
    let candidates: [MatchCandidate]

    enum CodingKeys: String, CodingKey {
        case query, candidates
        case fileName = "file_name"
        case fileYear = "file_year"
        case fileRuntime = "file_runtime"
    }
}

private struct MatchCheckStartResponse: Codable {
    let status: MatchCheckStatus
}

struct MatchAcceptAllResponse: Codable {
    let accepted: Int
    let failed: Int
}

enum MatchFormat {
    /// "1h 8m"
    static func minutes(_ minutes: Int) -> String {
        if minutes < 60 { return "\(minutes)m" }
        let (hours, rest) = (minutes / 60, minutes % 60)
        return rest == 0 ? "\(hours)h" : "\(hours)h \(rest)m"
    }
}

// MARK: - API

extension APIClient {
    func getMatchReview() async throws -> MatchReviewResponse {
        try await get("/api/library/match-review")
    }

    /// Starts checking every movie in the background. Poll `getMatchReview`
    /// for progress.
    func startMatchCheck() async throws -> MatchCheckStatus {
        let response: MatchCheckStartResponse = try await post("/api/library/match-review/scan", body: EmptyBody())
        return response.status
    }

    func acceptMatchSuggestion(mediaId: Int64) async throws {
        let _: EmptyResponse = try await post("/api/library/match-review/\(mediaId)/accept", body: EmptyBody())
    }

    func keepCurrentMatch(mediaId: Int64) async throws {
        let _: EmptyResponse = try await post("/api/library/match-review/\(mediaId)/keep", body: EmptyBody())
    }

    func acceptAllMatchSuggestions() async throws -> MatchAcceptAllResponse {
        try await post("/api/library/match-review/accept-all", body: EmptyBody())
    }

    /// Films a movie could be. An empty query searches on the title in the
    /// file's name.
    func getMatchCandidates(mediaId: Int64, query: String = "") async throws -> MatchCandidatesResponse {
        let encoded = query.addingPercentEncoding(withAllowedCharacters: .alphanumerics) ?? ""
        return try await get("/api/library/match-review/\(mediaId)/candidates?query=\(encoded)")
    }

    func chooseMatch(mediaId: Int64, tmdbId: Int) async throws {
        let _: EmptyResponse = try await post("/api/library/match-review/\(mediaId)/choose", body: ["tmdb_id": tmdbId])
    }
}

// MARK: - View models

/// The queue of movies whose match looks wrong, and the check that fills it.
@MainActor
class MatchReviewViewModel: ObservableObject {
    @Published var items: [MatchReviewItem] = []
    @Published var status: MatchCheckStatus?
    @Published var isLoading = false
    @Published var errorMessage: String?

    /// Movies with a request in flight, so their buttons can be disabled.
    @Published var busy: Set<Int64> = []
    @Published var isAcceptingAll = false

    private let api = APIClient.shared
    private var pollTask: Task<Void, Never>?
    private let pollInterval: UInt64 = 2_000_000_000

    var isChecking: Bool { status?.running == true }

    /// One line on where the check stands.
    var statusText: String {
        guard let status else { return "" }
        if status.running {
            return status.total > 0
                ? "Checking \(status.checked) of \(status.total) movies…"
                : "Starting check…"
        }
        if let error = status.error, !error.isEmpty { return error }
        if status.finishedAt == nil && items.isEmpty {
            return "Check the library to find movies matched to the wrong film."
        }
        switch items.count {
        case 0: return "Every movie checked out."
        case 1: return "1 movie looks wrong."
        default: return "\(items.count) movies look wrong."
        }
    }

    func load() async {
        if items.isEmpty { isLoading = true }
        await refresh()
        isLoading = false
        // A check started earlier, here or on another device, may still be
        // running.
        if isChecking { startPolling() }
    }

    func startCheck() async {
        errorMessage = nil
        do {
            status = try await api.startMatchCheck()
        } catch APIError.serverError(409, _) {
            // Already running; just follow it.
        } catch {
            errorMessage = error.localizedDescription
            return
        }
        startPolling()
    }

    func accept(_ item: MatchReviewItem) async {
        await resolve(item) { try await self.api.acceptMatchSuggestion(mediaId: item.mediaId) }
    }

    func keep(_ item: MatchReviewItem) async {
        await resolve(item) { try await self.api.keepCurrentMatch(mediaId: item.mediaId) }
    }

    /// Drops a movie from the queue after it was fixed by hand.
    func remove(mediaId: Int64) {
        items.removeAll { $0.mediaId == mediaId }
    }

    func acceptAll() async {
        isAcceptingAll = true
        errorMessage = nil
        do {
            let result = try await api.acceptAllMatchSuggestions()
            if result.failed > 0 {
                errorMessage = "\(result.failed) could not be applied."
            }
        } catch {
            errorMessage = error.localizedDescription
        }
        await refresh()
        isAcceptingAll = false
    }

    func stopPolling() {
        pollTask?.cancel()
        pollTask = nil
    }

    private func resolve(_ item: MatchReviewItem, _ action: @escaping () async throws -> Void) async {
        busy.insert(item.mediaId)
        errorMessage = nil
        do {
            try await action()
            remove(mediaId: item.mediaId)
        } catch {
            errorMessage = error.localizedDescription
        }
        busy.remove(item.mediaId)
    }

    private func refresh() async {
        do {
            let response = try await api.getMatchReview()
            items = response.items
            status = response.status
        } catch {
            if items.isEmpty { errorMessage = error.localizedDescription }
        }
    }

    private func startPolling() {
        stopPolling()
        let interval = pollInterval
        pollTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: interval)
                guard !Task.isCancelled, let self else { return }
                await self.refresh()
                if !self.isChecking { return }
            }
        }
    }
}

/// Choosing a movie's match by hand.
@MainActor
class FixMatchViewModel: ObservableObject {
    @Published var candidates: [MatchCandidate] = []
    @Published var query = ""
    @Published var isLoading = false
    @Published var isChoosing = false
    @Published var errorMessage: String?

    let mediaId: Int64
    private let api = APIClient.shared

    init(mediaId: Int64) {
        self.mediaId = mediaId
    }

    /// Searches on `query`, or on the title in the file's name while it is
    /// empty.
    func search() async {
        isLoading = true
        errorMessage = nil
        do {
            let response = try await api.getMatchCandidates(mediaId: mediaId, query: query)
            candidates = response.candidates
            if query.isEmpty { query = response.query }
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }

    /// Returns whether the match was applied.
    func choose(_ candidate: MatchCandidate) async -> Bool {
        isChoosing = true
        errorMessage = nil
        defer { isChoosing = false }
        do {
            try await api.chooseMatch(mediaId: mediaId, tmdbId: candidate.tmdbId)
            return true
        } catch {
            errorMessage = error.localizedDescription
            return false
        }
    }
}
