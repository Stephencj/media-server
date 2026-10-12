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

    // MARK: - Tuning

    /// What a channel was airing when last asked, so that flipping to it can
    /// show its name and start its picture without waiting on the server.
    private struct TunedIn {
        let response: ChannelNowPlayingResponse
        let fetchedAt: Date

        /// Where the channel has got to by now, in seconds into its programme.
        var livePosition: Int { response.elapsed + Int(Date().timeIntervalSince(fetchedAt)) }

        /// Whether that programme is still on.
        var stillOn: Bool {
            guard let item = response.nowPlaying else { return false }
            return Double(livePosition) < Double(item.duration) - 2
        }
    }

    /// A neighbouring channel held ready: its stream open and buffered at
    /// the point it has reached, paused and unseen, so that flipping to it is
    /// a matter of showing it.
    private struct Standby {
        let player: AVPlayer
        let tunedIn: TunedIn
        let scheduleItemId: Int64
    }

    private var tunedIn: [Int64: TunedIn] = [:]
    /// At most the two channels one press away. Everything else is closed.
    private var standby: [Int64: Standby] = [:]
    /// What the channel being watched was airing when it was tuned, kept so
    /// it can go to standby when flipped away from.
    private var currentTunedIn: TunedIn?

    private var tuneTask: Task<Void, Never>?
    private var standbyTask: Task<Void, Never>?
    /// Bumped on every tune, so a slow answer for a channel since flipped
    /// past can be recognised and dropped.
    private var tuneGeneration = 0

    /// How long to wait after a press before loading a channel that is not
    /// on standby. Holding the button to surf then loads only where it
    /// stops, not every channel on the way.
    private let tuneDelay: UInt64 = 250_000_000
    /// How long a channel has to be settled on before its neighbours are
    /// opened. Surfing past a channel should cost it nothing.
    private let standbySettle: UInt64 = 700_000_000
    /// How often standby channels are moved up to where they have got to. A
    /// paused stream buffers ahead of where it sits, so kept this fresh the
    /// live point is already loaded when it is needed.
    private let standbyRefresh: UInt64 = 15_000_000_000
    private let tunedInLifetime: TimeInterval = 120

    /// What is known about a channel, if it is recent and its programme is
    /// still on.
    private func remembered(_ channelId: Int64) -> TunedIn? {
        guard let entry = tunedIn[channelId] else { return nil }
        guard Date().timeIntervalSince(entry.fetchedAt) < tunedInLifetime, entry.stillOn else { return nil }
        return entry
    }

    /// Closes a stream for good: playback, anything still loading, and the
    /// item itself. An idle player holding an item keeps its connection and
    /// its decoder, which is what there must never be a pile of.
    private func kill(_ player: AVPlayer) {
        player.pause()
        player.cancelPendingPrerolls()
        player.currentItem?.cancelPendingSeeks()
        player.currentItem?.asset.cancelLoading()
        player.replaceCurrentItem(with: nil)
    }

    private func killStandby(except keep: Set<Int64>) {
        for (channelId, held) in standby where !keep.contains(channelId) {
            kill(held.player)
            standby[channelId] = nil
        }
    }

    private func time(_ seconds: Int) -> CMTime {
        CMTime(seconds: Double(seconds), preferredTimescale: CMTimeScale(NSEC_PER_SEC))
    }

    /// Changes channel the way a television does. A channel one press away is
    /// already open and simply takes over the screen. Any other cuts to black
    /// with its name up at once and loads from cold.
    ///
    /// `neighbours` are the channels a further press would reach. Once this
    /// one has settled they are opened and held ready, and every other
    /// stream is closed: never more than three open at a time.
    func tune(to target: Channel, neighbours: [Channel], immediately: Bool = false) {
        tuneTask?.cancel()
        standbyTask?.cancel()
        tuneGeneration += 1
        let generation = tuneGeneration
        let neighbourIds = Set(neighbours.map(\.id))

        // Whatever is no longer one press away goes now, before anything
        // new is opened.
        killStandby(except: neighbourIds.union([target.id]))

        // The channel being left stays open if it is still a neighbour.
        if let leaving = player {
            if let leftId = channel?.id, leftId != target.id, neighbourIds.contains(leftId),
               let was = currentTunedIn, was.stillOn, let itemId = currentScheduleItemId {
                leaving.pause()
                standby[leftId] = Standby(player: leaving, tunedIn: was, scheduleItemId: itemId)
            } else {
                kill(leaving)
            }
        }
        cancellables.removeAll()
        player = nil
        currentTunedIn = nil
        errorMessage = nil
        channel = target

        if let ready = standby[target.id], ready.tunedIn.stillOn, let item = ready.tunedIn.response.nowPlaying {
            standby[target.id] = nil
            player = ready.player
            setupObservers()

            currentTunedIn = ready.tunedIn
            currentMediaId = item.mediaId
            currentScheduleItemId = item.id
            nowPlaying = item
            upNext = ready.tunedIn.response.upNext
            elapsed = ready.tunedIn.livePosition
            isLoading = false

            ready.player.seek(to: time(elapsed))
            ready.player.play()
            startRefreshTask()
        } else {
            if let stale = standby[target.id] {
                kill(stale.player)
                standby[target.id] = nil
            }
            let known = remembered(target.id)
            nowPlaying = known?.response.nowPlaying
            upNext = known?.response.upNext ?? []
            isLoading = true

            tuneTask = Task { [weak self] in
                if !immediately {
                    try? await Task.sleep(nanoseconds: self?.tuneDelay ?? 0)
                }
                guard let self, !Task.isCancelled, generation == self.tuneGeneration else { return }

                if let known = self.remembered(target.id), let item = known.response.nowPlaying {
                    await AuthService.shared.ensureAuthenticated()
                    guard generation == self.tuneGeneration else { return }
                    self.currentTunedIn = known
                    self.elapsed = known.livePosition
                    self.setupPlayer(for: item, seekTo: known.livePosition, serverStreamUrl: known.response.streamUrl)
                    self.startRefreshTask()
                    self.isLoading = false
                } else {
                    await self.loadChannel(channelId: target.id, generation: generation)
                }
            }
        }

        standbyTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: self?.standbySettle ?? 0)
            // The channel being watched comes first.
            while let self, self.isLoading, !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 200_000_000)
            }
            while let self, !Task.isCancelled, generation == self.tuneGeneration {
                await self.holdReady(neighbours, generation: generation)
                try? await Task.sleep(nanoseconds: self.standbyRefresh)
            }
        }
    }

    /// Opens each neighbour that isn't open, replaces any whose programme has
    /// ended, and moves the rest up to where they have got to.
    private func holdReady(_ neighbours: [Channel], generation: Int) async {
        for neighbour in neighbours {
            guard !Task.isCancelled, generation == tuneGeneration else { return }

            if let held = standby[neighbour.id] {
                if held.tunedIn.stillOn {
                    held.player.seek(to: time(held.tunedIn.livePosition)) { _ in }
                    continue
                }
                kill(held.player)
                standby[neighbour.id] = nil
            }

            var known = remembered(neighbour.id)
            if known == nil, let response = try? await api.getChannelNowPlaying(channelId: neighbour.id) {
                let fresh = TunedIn(response: response, fetchedAt: Date())
                tunedIn[neighbour.id] = fresh
                known = fresh
            }
            // The wait above may have outlived this tune.
            guard !Task.isCancelled, generation == tuneGeneration else { return }
            guard let known, known.stillOn, let item = known.response.nowPlaying,
                  let playerItem = makeItem(for: item, serverStreamUrl: known.response.streamUrl, buffer: 20) else {
                continue
            }

            let held = AVPlayer(playerItem: playerItem)
            held.automaticallyWaitsToMinimizeStalling = true
            // Paused: it loads and buffers from here without playing.
            held.seek(to: time(known.livePosition)) { _ in }
            standby[neighbour.id] = Standby(player: held, tunedIn: known, scheduleItemId: item.id)
        }
    }

    func loadChannel(channelId: Int64, generation: Int? = nil) async {
        isLoading = true
        errorMessage = nil

        print("[ChannelPlayer] Loading channel: \(channelId)")

        // Ensure we're authenticated before trying to play
        await AuthService.shared.ensureAuthenticated()

        do {
            let response = try await api.getChannelNowPlaying(channelId: channelId)
            // Flipped past this channel while it was loading: let it go.
            if let generation, generation != tuneGeneration { return }
            let fresh = TunedIn(response: response, fetchedAt: Date())
            tunedIn[channelId] = fresh
            currentTunedIn = fresh

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
            if let generation, generation != tuneGeneration { return }
            errorMessage = "Load failed: \(error.localizedDescription) [server: \(AppState.shared.serverURL)]"
            print("[ChannelPlayer] ERROR loading channel: \(error)")
        }

        isLoading = false
    }

    /// Builds the stream for a programme, or nil if no address can be made
    /// for it. `buffer` is how far ahead to load, in seconds.
    private func makeItem(for item: ChannelScheduleItem, serverStreamUrl: String?, buffer: TimeInterval) -> AVPlayerItem? {
        let media = item.toMedia()
        let baseURL = AppState.shared.serverURL

        // Prefer the address the server gave, which carries the right type.
        let streamURL: URL?
        if let serverPath = serverStreamUrl, let token = AuthService.shared.token {
            let separator = serverPath.contains("?") ? "&" : "?"
            streamURL = URL(string: "\(baseURL)\(serverPath)\(separator)token=\(token)")
        } else if let directURL = api.getDirectPlayURL(mediaId: media.id, mediaType: media.type) {
            streamURL = directURL
        } else {
            streamURL = api.getStreamURL(mediaId: media.id, mediaType: media.type)
        }
        guard let url = streamURL else { return nil }

        let asset = AVURLAsset(url: url, options: [
            "AVURLAssetHTTPHeaderFieldsKey": ["Connection": "keep-alive"]
        ])
        let playerItem = AVPlayerItem(asset: asset)
        playerItem.preferredForwardBufferDuration = buffer
        return playerItem
    }

    private func setupPlayer(for item: ChannelScheduleItem, seekTo elapsed: Int, serverStreamUrl: String? = nil) {
        // Tear down only the player — NOT the refresh loop, which must survive
        // program changes (and may itself be what triggered this call).
        teardownPlayer()

        currentMediaId = item.mediaId
        currentScheduleItemId = item.id

        print("[ChannelPlayer] Setting up player for: \(item.title) (mediaId: \(item.mediaId), type: \(item.mediaType))")

        guard let playerItem = makeItem(for: item, serverStreamUrl: serverStreamUrl, buffer: 60) else {
            let token = AuthService.shared.token
            errorMessage = "Failed to get stream URL (token: \(token != nil ? "yes" : "NO"), server: \(AppState.shared.serverURL))"
            print("[ChannelPlayer] ERROR: Failed to get stream URL. serverStreamUrl=\(serverStreamUrl ?? "nil")")
            return
        }

        // The channel keeps one player and hands it each programme in turn.
        // Building a player per programme rebuilt the video view with it,
        // which is slower and flashes on every change.
        if let player {
            player.replaceCurrentItem(with: playerItem)
        } else {
            let newPlayer = AVPlayer(playerItem: playerItem)
            newPlayer.automaticallyWaitsToMinimizeStalling = true
            player = newPlayer
            setupObservers()
        }

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
                // Standby channels have items of their own; only the one on
                // screen has anything to report.
                guard let self, (notification.object as? AVPlayerItem) === self.player?.currentItem else { return }
                if let error = notification.userInfo?[AVPlayerItemFailedToPlayToEndTimeErrorKey] as? Error {
                    self.errorMessage = "Playback error: \(error.localizedDescription)"
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

        // What was remembered about this channel is now a programme behind.
        if let channelId = channel?.id { tunedIn[channelId] = nil }
        currentTunedIn = nil

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

    /// Stops what is playing but keeps the player, ready for the next item.
    /// Safe to call between programs.
    private func teardownPlayer() {
        player?.pause()
        player?.replaceCurrentItem(with: nil)
    }

    /// Full teardown, for leaving the player entirely.
    func cleanup() {
        tuneTask?.cancel()
        tuneTask = nil
        standbyTask?.cancel()
        standbyTask = nil
        killStandby(except: [])
        if let observer = timeObserver, let player = player {
            player.removeTimeObserver(observer)
            timeObserver = nil
        }
        if let player { kill(player) }
        player = nil
        currentTunedIn = nil
        cancellables.removeAll()
        refreshTask?.cancel()
        refreshTask = nil
        cancelSleepTimer()
    }
}
