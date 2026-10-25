import Foundation

/// How far a channel's viewer is behind what the channel is airing.
///
/// A channel tunes in live. Pausing, or skipping back, puts the viewer behind,
/// and they stay that many seconds behind from then on, through the end of
/// the programme and into the next, until they skip forward or jump to live.
/// Skipping forward can only ever catch them up to live, never past it.
///
/// Kept apart from the player so its rules can be tested on their own.
struct ChannelTimeshift: Equatable {
    /// Seconds behind live, not counting a pause still in progress.
    private(set) var settled: TimeInterval = 0
    /// When the current pause began, if paused.
    private(set) var pausedAt: Date?

    /// Within this of live counts as live: a seek never lands exactly.
    static let liveTolerance: TimeInterval = 3

    init() {}

    /// Playing, `seconds` behind live.
    init(behind seconds: TimeInterval) {
        settled = seconds < Self.liveTolerance ? 0 : seconds
    }

    var isPaused: Bool { pausedAt != nil }

    /// Seconds behind live at `now`, including a pause still running.
    func behind(at now: Date) -> TimeInterval {
        settled + (pausedAt.map { max(0, now.timeIntervalSince($0)) } ?? 0)
    }

    func isLive(at now: Date) -> Bool {
        behind(at: now) < Self.liveTolerance
    }

    /// The moment on the channel's clock the viewer is watching.
    func watching(at now: Date) -> Date {
        now.addingTimeInterval(-behind(at: now))
    }

    mutating func pause(at now: Date) {
        guard pausedAt == nil else { return }
        pausedAt = now
    }

    mutating func resume(at now: Date) {
        settled = behind(at: now)
        pausedAt = nil
    }

    /// Back to live: nothing behind, not paused.
    mutating func goLive() {
        settled = 0
        pausedAt = nil
    }

    /// Where a skip of `delta` seconds from `position` in a programme of
    /// `duration` seconds lands, and the viewer's new place behind live.
    ///
    /// It stays within the programme, and forward it stops at live.
    mutating func skip(from position: TimeInterval, by delta: TimeInterval,
                       duration: TimeInterval, at now: Date) -> TimeInterval {
        let target = Self.clamp(position + delta, from: position, duration: duration, behind: behind(at: now))
        settled = behind(at: now) - (target - position)
        if settled < Self.liveTolerance { settled = 0 }
        if pausedAt != nil { pausedAt = now }
        return target
    }

    /// The furthest forward a viewer at `position` can go: live, or the end
    /// of the programme if live is past it.
    static func limit(from position: TimeInterval, duration: TimeInterval, behind: TimeInterval) -> TimeInterval {
        min(position + behind, max(0, duration - 1))
    }

    static func clamp(_ target: TimeInterval, from position: TimeInterval,
                      duration: TimeInterval, behind: TimeInterval) -> TimeInterval {
        min(max(0, target), max(position, limit(from: position, duration: duration, behind: behind)))
    }

    /// Which programme follows, behind live, when the one being watched ends:
    /// the one after it in the guide (`starts` in air order, `ids` their
    /// schedule rows), or failing that whatever the guide has on at `watching`.
    static func nextProgramme(ids: [Int64], starts: [Date], ending current: Int64?, watching: Date) -> Int? {
        if let current, let index = ids.indices.last(where: { ids[$0] == current && starts[$0] <= watching }),
           index + 1 < ids.count {
            return index + 1
        }
        return starts.indices.last(where: { starts[$0] <= watching }) ?? starts.indices.first
    }

    /// "12:05" or "1:02:05".
    static func clock(_ seconds: TimeInterval) -> String {
        let total = max(0, Int(seconds.rounded(.down)))
        let h = total / 3600, m = (total % 3600) / 60, s = total % 60
        return h > 0 ? String(format: "%d:%02d:%02d", h, m, s) : String(format: "%d:%02d", m, s)
    }
}
