import Foundation

@MainActor
class ChannelGuideViewModel: ObservableObject {
    /// One lane per channel, in guide-number order as the server returns them.
    @Published var entries: [ChannelGuideEntry] = []
    @Published var windowStart = Date()
    @Published var windowEnd = Date()

    /// Ticks locally between reloads. Without it the now-line and every
    /// "x min left" would only move once a minute, in visible jumps.
    @Published var now = Date()

    @Published var isLoading = false
    @Published var errorMessage: String?

    /// Hours of schedule fetched per load. The server caps this at 12.
    let windowHours = 4

    private let api = APIClient.shared
    private var refreshTask: Task<Void, Never>?
    private var tickTask: Task<Void, Never>?

    /// Reload cadence. The guide only changes when a program ends, and the
    /// local tick covers everything in between.
    private let reloadInterval: UInt64 = 300_000_000_000  // 5 minutes
    private let tickInterval: UInt64 = 1_000_000_000

    var windowMinutes: Int {
        max(1, Int(windowEnd.timeIntervalSince(windowStart) / 60))
    }

    func loadGuide() async {
        if entries.isEmpty { isLoading = true }
        errorMessage = nil

        do {
            let response = try await api.getChannelGuide(start: Date(), hours: windowHours)
            entries = response.channels
            windowStart = response.start
            windowEnd = response.end
            now = Date()
        } catch {
            // Keep whatever is already on screen rather than blanking the guide.
            if entries.isEmpty {
                errorMessage = error.localizedDescription
            }
        }

        isLoading = false
    }

    /// The program a channel is airing right now, if any.
    func nowPlaying(for entry: ChannelGuideEntry) -> GuideProgram? {
        entry.program(airingAt: now)
    }

    func upNext(for entry: ChannelGuideEntry) -> GuideProgram? {
        entry.program(after: now)
    }

    func startAutoRefresh() {
        stopAutoRefresh()

        let reload = reloadInterval
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: reload)
                guard !Task.isCancelled, let self else { return }
                await self.loadGuide()
            }
        }

        let tick = tickInterval
        tickTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: tick)
                guard !Task.isCancelled, let self else { return }
                self.now = Date()

                // Roll the window forward once "now" walks past it rather than
                // waiting for the next scheduled reload.
                if self.now >= self.windowEnd {
                    await self.loadGuide()
                }
            }
        }
    }

    func stopAutoRefresh() {
        refreshTask?.cancel()
        refreshTask = nil
        tickTask?.cancel()
        tickTask = nil
    }
}
