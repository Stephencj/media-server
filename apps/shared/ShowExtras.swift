import Foundation

// A TV show's extras, as the iPhone and Apple TV apps both show them. One
// file, built into both apps, so the two can't drift apart.

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

    /// The season it belongs to: as the library linked it, or as its name
    /// says ("S01 - Deleted Scenes", "DSS01E12 Cloudy Chance of Murder").
    var season: Int? { seasonNumber ?? ShowExtras.code(in: title)?.season }

    /// The episode it belongs to, the same way.
    var episode: Int? {
        if seasonNumber != nil { return episodeNumber }
        return ShowExtras.code(in: title)?.episode
    }

    /// "S01E02", when it belongs to one episode.
    var episodeLabel: String? {
        guard let s = season, let e = episode else { return nil }
        return String(format: "S%02dE%02d", s, e)
    }

    /// Its name without the code the files are named with:
    /// "Com1S01E01 - Pilot" is "Pilot".
    var displayTitle: String { ShowExtras.cleanTitle(title) }

    /// "1h 46m", "22m", or "4m 12s" for the short ones.
    var formattedDuration: String {
        guard let duration, duration > 0 else { return "" }
        if duration >= 3600 {
            return "\(duration / 3600)h \((duration % 3600) / 60)m"
        }
        return duration >= 600 ? "\(duration / 60)m" : "\(duration / 60)m \(duration % 60)s"
    }

    /// The line under its name: episode and length.
    var detail: String {
        [episodeLabel, formattedDuration.isEmpty ? nil : formattedDuration]
            .compactMap { $0 }
            .joined(separator: " \u{00B7} ")
    }
}

/// Extras of one kind, in the order they are shown.
struct ShowExtraGroup: Identifiable, Equatable {
    let title: String
    let extras: [ShowExtra]
    var id: String { title }
}

enum ShowExtras {
    /// Kinds in the order they're shown, commentaries first.
    static let kinds: [(category: String, title: String)] = [
        ("commentary", "Commentaries"),
        ("deleted_scene", "Deleted Scenes"),
        ("featurette", "Featurettes"),
        ("behind_the_scenes", "Behind the Scenes"),
        ("interview", "Interviews"),
        ("gag_reel", "Gag Reels"),
        ("music_video", "Music Videos"),
        ("other", "More"),
    ]

    /// The extras of one season, by kind.
    static func season(_ number: Int, of all: [ShowExtra]) -> [ShowExtraGroup] {
        groups(all.filter { $0.season == number })
    }

    /// The extras about the whole show rather than one season, by kind.
    static func wholeShow(_ all: [ShowExtra]) -> [ShowExtraGroup] {
        groups(all.filter { $0.season == nil })
    }

    /// An episode's commentary tracks.
    static func commentaries(season: Int, episode: Int, of all: [ShowExtra]) -> [ShowExtra] {
        all.filter { $0.isCommentary && $0.season == season && $0.episode == episode }
    }

    /// Extras by kind in the order of `kinds`, each in episode order, then by
    /// name. Kinds the apps don't know go under More.
    static func groups(_ extras: [ShowExtra]) -> [ShowExtraGroup] {
        let known = Set(kinds.map(\.category))
        return kinds.compactMap { kind in
            let matching = extras.filter {
                $0.category == kind.category || (kind.category == "other" && !known.contains($0.category))
            }
            .sorted { a, b in
                let ka = (a.season ?? Int.max, a.episode ?? Int.max), kb = (b.season ?? Int.max, b.episode ?? Int.max)
                if ka != kb { return ka < kb }
                return a.title.localizedStandardCompare(b.title) == .orderedAscending
            }
            return matching.isEmpty ? nil : ShowExtraGroup(title: kind.title, extras: matching)
        }
    }

    /// The season and episode a file name codes: "S01E12", "s1e2", or a
    /// season alone ("S02BTS01", "S01 - Deleted Scenes").
    static func code(in title: String) -> (season: Int, episode: Int?)? {
        let pattern = #"(?i)(?:^|[^a-z0-9]|com\d*|ds|le)s(\d{1,2})(?:e(\d{1,3}))?(?![0-9])"#
        guard let regex = try? NSRegularExpression(pattern: pattern),
              let match = regex.firstMatch(in: title, range: NSRange(title.startIndex..., in: title)),
              let seasonRange = Range(match.range(at: 1), in: title),
              let season = Int(title[seasonRange]) else { return nil }
        let episode = Range(match.range(at: 2), in: title).flatMap { Int(title[$0]) }
        return (season, episode)
    }

    /// A file's name without its leading code and separators.
    static func cleanTitle(_ title: String) -> String {
        let pattern = #"(?i)^\s*(?:com\d*|ds|le)?\s*s\d{1,2}(?:e\d{1,3})?\s*[-–:]?\s*"#
        let cleaned = title.replacingOccurrences(of: pattern, with: "", options: .regularExpression)
            .trimmingCharacters(in: .whitespaces)
        return cleaned.isEmpty ? title : cleaned
    }
}
