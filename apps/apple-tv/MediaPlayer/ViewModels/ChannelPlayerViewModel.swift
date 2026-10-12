import Foundation
import AVKit
import Combine

@MainActor
class ChannelPlayerViewModel: ObservableObject {
    @Published var channel: Channel?
    @Published var nowPlaying: ChannelScheduleItem?
    @Published var upNext: [ChannelScheduleItem] = []
    @Published var elapsed: Int = 0
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Published var player: AVPlayer?
    @Published var isBuffering = false

    private let api = APIClient.shared
    private var timeObserver: Any?
    private var cancellables = Set<AnyCancellable>()
    private var refreshTask: Task<Void, Never>?
    private var currentMediaId: Int64?
    private var currentScheduleItemId: Int64?

    /// How often to top the up-next queue back up from the server. The server
    /// only ever returns three items, so without this the queue drains and
    /// playback dead-ends after three programs.
    private let refreshInterval: UInt64 = 30_000_000_000

    // MARK: Sleep timer

    /// When playback is due to stop, or nil when no timer is armed.
    @Published private(set) var sleepFiresAt: Date?
    /// Seconds left, republished every second so the countdown can be shown.
    @Published private(set) var sleepRemaining: TimeInterval = 0
    /// Bumped each time the timer fires. Views watch this to dismiss the
    /// player — a counter rather than a callback, so it survives view identity
    /// changes and can't leak a retained closure.
    @Published private(set) var sleepFireCount = 0

    private var sleepTask: Task<Void, Never>?

    var isSleepTimerArmed: Bool { sleepFiresAt != nil }

    func loadChannel(channelId: Int64) async {
        isLoading = true
        errorMessage = nil

        print("[ChannelPlayer] Loading channel: \(channelId)")

        // Ensure we're authenticated before trying to play
        await AuthService.shared.ensureAuthenticated()

        do {
            let response = try await api.getChannelNowPlaying(channelId: channelId)
            channel = response.channel
            nowPlaying = response.nowPlaying
            upNext = response.upNext
            elapsed = response.elapsed

            print("[ChannelPlayer] Channel loaded: \(response.channel.name)")
            print("[ChannelPlayer] Now playing: \(response.nowPlaying?.title ?? "nil")")
            print("[ChannelPlayer] Elapsed: \(response.elapsed)s")
            print("[ChannelPlayer] Up next count: \(response.upNext.count)")
            print("[ChannelPlayer] Stream URL from server: \(response.streamUrl ?? "nil")")

            // Setup player with the current item
            if let nowPlaying = response.nowPlaying {
                setupPlayer(for: nowPlaying, seekTo: response.elapsed, serverStreamUrl: response.streamUrl)
                startRefreshTask()
            } else {
                errorMessage = "No content currently playing on this channel"
                print("[ChannelPlayer] ERROR: No nowPlaying item in response")
            }
        } catch {
            errorMessage = "Load failed: \(error.localizedDescription) [server: \(AppState.shared.serverURL)]"
            print("[ChannelPlayer] ERROR loading channel: \(error)")
        }

        isLoading = false
    }

    private func setupPlayer(for item: ChannelScheduleItem, seekTo elapsed: Int, serverStreamUrl: String? = nil) {
        // Tear down only the player — NOT the refresh loop, which must survive
        // program changes (and may itself be what triggered this call).
        teardownPlayer()

        let media = item.toMedia()
        currentMediaId = item.mediaId
        currentScheduleItemId = item.id

        print("[ChannelPlayer] Setting up player for: \(item.title) (mediaId: \(item.mediaId), type: \(item.mediaType))")

        // Get stream URL - prefer server-provided URL, fall back to manual construction
        let streamURL: URL?
        let baseURL = AppState.shared.serverURL
        let token = AuthService.shared.token

        if let serverPath = serverStreamUrl, let token = token {
            // Use server-provided stream URL (includes correct type and start params)
            let separator = serverPath.contains("?") ? "&" : "?"
            let fullURL = "\(baseURL)\(serverPath)\(separator)token=\(token)"
            streamURL = URL(string: fullURL)
            print("[ChannelPlayer] Using server stream URL: \(fullURL)")
        } else if let directURL = api.getDirectPlayURL(mediaId: media.id, mediaType: media.type) {
            streamURL = directURL
            print("[ChannelPlayer] Using direct play URL: \(directURL)")
        } else if let hlsURL = api.getStreamURL(mediaId: media.id, mediaType: media.type) {
            streamURL = hlsURL
            print("[ChannelPlayer] Using HLS URL: \(hlsURL)")
        } else {
            streamURL = nil
            print("[ChannelPlayer] ERROR: No stream URL available. baseURL=\(baseURL), token=\(token != nil ? "set" : "nil"), serverStreamUrl=\(serverStreamUrl ?? "nil")")
        }

        guard let url = streamURL else {
            errorMessage = "Failed to get stream URL (token: \(token != nil ? "yes" : "NO"), server: \(baseURL))"
            print("[ChannelPlayer] ERROR: Failed to get stream URL. token=\(token != nil), baseURL=\(baseURL), serverStreamUrl=\(serverStreamUrl ?? "nil")")
            return
        }

        print("[ChannelPlayer] Playing: \(url)")

        let asset = AVURLAsset(url: url, options: [
            "AVURLAssetHTTPHeaderFieldsKey": ["Connection": "keep-alive"]
        ])
        let playerItem = AVPlayerItem(asset: asset)
        playerItem.preferredForwardBufferDuration = 60
        let newPlayer = AVPlayer(playerItem: playerItem)
        newPlayer.automaticallyWaitsToMinimizeStalling = true
        player = newPlayer

        setupObservers()

        // Seek to the elapsed position after player is ready, then play
        if elapsed > 0 {
            let time = CMTime(seconds: Double(elapsed), preferredTimescale: CMTimeScale(NSEC_PER_SEC))
            player?.seek(to: time) { [weak self] finished in
                Task { @MainActor in
                    self?.player?.play()
                    print("[ChannelPlayer] Seeked to \(elapsed)s and started playback")
                }
            }
        } else {
            player?.play()
            print("[ChannelPlayer] Started playback from beginning")
        }
    }

    private func setupObservers() {
        guard let player = player else { return }

        // Buffering state
        player.publisher(for: \.timeControlStatus)
            .receive(on: DispatchQueue.main)
            .sink { [weak self] status in
                self?.isBuffering = status == .waitingToPlayAtSpecifiedRate
                print("[ChannelPlayer] Player status: \(status.rawValue)")
            }
            .store(in: &cancellables)

        // Player error observer
        player.publisher(for: \.currentItem?.status)
            .receive(on: DispatchQueue.main)
            .sink { [weak self] status in
                if status == .failed {
                    if let error = player.currentItem?.error {
                        self?.errorMessage = "Playback failed: \(error.localizedDescription)"
                        print("[ChannelPlayer] ERROR: Playback failed: \(error)")
                    } else {
                        self?.errorMessage = "Playback failed"
                        print("[ChannelPlayer] ERROR: Playback failed (unknown error)")
                    }
                }
            }
            .store(in: &cancellables)

        // Error notification
        NotificationCenter.default.publisher(for: .AVPlayerItemFailedToPlayToEndTime)
            .receive(on: DispatchQueue.main)
            .sink { [weak self] notification in
                if let error = notification.userInfo?[AVPlayerItemFailedToPlayToEndTimeErrorKey] as? Error {
                    self?.errorMessage = "Playback error: \(error.localizedDescription)"
                    print("[ChannelPlayer] ERROR: Failed to play to end: \(error)")
                }
            }
            .store(in: &cancellables)
    }

    /// Re-syncs with the server's idea of what is on right now.
    ///
    /// `force` re-seats the player even when the server still reports the item
    /// we already have loaded. That is what rescues playback at the end of a
    /// cycle: the server's `up_next` query does not wrap, so the queue runs dry
    /// while the cycle itself keeps going.
    func refreshNowPlaying(force: Bool = false) async {
        guard let channelId = channel?.id else { return }

        do {
            let response = try await api.getChannelNowPlaying(channelId: channelId)

            // Compare on the schedule row, not the media id — a channel may
            // schedule the same title more than once in a cycle.
            let itemChanged = response.nowPlaying?.id != currentScheduleItemId

            if force || itemChanged {
                nowPlaying = response.nowPlaying
                upNext = response.upNext
                elapsed = response.elapsed

                if let newItem = response.nowPlaying {
                    setupPlayer(for: newItem, seekTo: response.elapsed, serverStreamUrl: response.streamUrl)
                }
            } else {
                upNext = response.upNext
            }
        } catch {
            // Silent fail on refresh
        }
    }

    /// Tops the queue back up without disturbing playback.
    ///
    /// Deliberately does not re-seat the player: the client advances when the
    /// video file ends, which can legitimately run slightly ahead of the
    /// server's wall-clock position when a file is shorter than its recorded
    /// duration. Yanking the player back on that drift would rewind the viewer
    /// mid-program.
    private func refreshUpNext() async {
        guard let channelId = channel?.id else { return }

        if let response = try? await api.getChannelNowPlaying(channelId: channelId) {
            upNext = response.upNext
        }
    }

    private func startRefreshTask() {
        refreshTask?.cancel()
        let interval = refreshInterval
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: interval)
                guard !Task.isCancelled, let self else { return }
                await self.refreshUpNext()
            }
        }
    }

    func switchToChannel(channelId: Int64) async {
        // Load the new channel
        await loadChannel(channelId: channelId)
    }

    func advanceToNextItem() {
        guard !upNext.isEmpty else {
            // Queue drained (end of cycle, or three programs since the last
            // top-up). Ask the server what should be on instead of stopping.
            Task { await refreshNowPlaying(force: true) }
            return
        }

        let nextItem = upNext.removeFirst()
        nowPlaying = nextItem
        elapsed = 0

        setupPlayer(for: nextItem, seekTo: 0)

        // Refill straight away so a short program can't drain the queue faster
        // than the periodic refresh replenishes it.
        Task { await refreshUpNext() }
    }

    func handleVideoEnd() {
        // When video ends, advance to next item
        advanceToNextItem()
    }

    // MARK: - Sleep timer

    /// Arms the timer for a fixed number of minutes.
    func armSleepTimer(minutes: Int) {
        armSleepTimer(seconds: Double(minutes) * 60)
    }

    /// Arms the timer to stop when the current program ends.
    ///
    /// Uses the player's own position rather than the stored `elapsed`, which
    /// is only accurate at load time — the player has been running since.
    func armSleepTimerForEndOfProgram() {
        guard let item = nowPlaying else { return }
        let played = player?.currentTime().seconds ?? Double(elapsed)
        let remaining = Double(item.duration) - (played.isFinite ? played : Double(elapsed))
        // Never arm for less than a few seconds — an almost-finished program
        // would otherwise stop playback the instant you chose the option.
        armSleepTimer(seconds: max(15, remaining))
    }

    /// Seconds until the current program ends, for labelling the option.
    var secondsUntilEndOfProgram: Int? {
        guard let item = nowPlaying else { return nil }
        let played = player?.currentTime().seconds ?? Double(elapsed)
        let remaining = Double(item.duration) - (played.isFinite ? played : Double(elapsed))
        return max(0, Int(remaining))
    }

    private func armSleepTimer(seconds: Double) {
        sleepTask?.cancel()

        let firesAt = Date().addingTimeInterval(seconds)
        sleepFiresAt = firesAt
        sleepRemaining = seconds

        sleepTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 1_000_000_000)
                guard !Task.isCancelled, let self else { return }

                let left = firesAt.timeIntervalSinceNow
                if left <= 0 {
                    self.fireSleepTimer()
                    return
                }
                self.sleepRemaining = left
            }
        }
    }

    func cancelSleepTimer() {
        sleepTask?.cancel()
        sleepTask = nil
        sleepFiresAt = nil
        sleepRemaining = 0
    }

    private func fireSleepTimer() {
        pause()
        cancelSleepTimer()
        sleepFireCount += 1
    }

    /// Countdown for the badge: m:ss under an hour, h:mm above it.
    var sleepCountdownText: String? {
        guard sleepFiresAt != nil else { return nil }
        let total = max(0, Int(sleepRemaining.rounded()))
        if total >= 3600 {
            return String(format: "%d:%02d", total / 3600, (total % 3600) / 60)
        }
        return String(format: "%d:%02d", total / 60, total % 60)
    }

    func play() {
        player?.play()
    }

    func pause() {
        player?.pause()
    }

    /// Tears down the player only. Safe to call between programs.
    private func teardownPlayer() {
        if let observer = timeObserver, let player = player {
            player.removeTimeObserver(observer)
            timeObserver = nil
        }
        player?.pause()
        player = nil
        cancellables.removeAll()
    }

    /// Full teardown, for leaving the player entirely.
    func cleanup() {
        teardownPlayer()
        refreshTask?.cancel()
        refreshTask = nil
        cancelSleepTimer()
    }
}
