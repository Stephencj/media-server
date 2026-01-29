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

    func loadChannel(channelId: Int64) async {
        isLoading = true
        errorMessage = nil

        print("[ChannelPlayer] Loading channel: \(channelId)")

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

            // Setup player with the current item
            if let nowPlaying = response.nowPlaying {
                setupPlayer(for: nowPlaying, seekTo: response.elapsed)
            } else {
                errorMessage = "No content currently playing on this channel"
                print("[ChannelPlayer] ERROR: No nowPlaying item in response")
            }
        } catch {
            errorMessage = error.localizedDescription
            print("[ChannelPlayer] ERROR loading channel: \(error)")
        }

        isLoading = false
    }

    private func setupPlayer(for item: ChannelScheduleItem, seekTo elapsed: Int) {
        // Clean up existing player
        cleanup()

        let media = item.toMedia()
        currentMediaId = item.mediaId

        print("[ChannelPlayer] Setting up player for: \(item.title) (mediaId: \(item.mediaId), type: \(item.mediaType))")

        // Get stream URL for the media
        let streamURL: URL?
        if let directURL = api.getDirectPlayURL(mediaId: media.id, mediaType: media.type) {
            streamURL = directURL
            print("[ChannelPlayer] Using direct play URL: \(directURL)")
        } else if let hlsURL = api.getStreamURL(mediaId: media.id, mediaType: media.type) {
            streamURL = hlsURL
            print("[ChannelPlayer] Using HLS URL: \(hlsURL)")
        } else {
            streamURL = nil
        }

        guard let url = streamURL else {
            errorMessage = "Failed to get stream URL"
            print("[ChannelPlayer] ERROR: Failed to get stream URL")
            return
        }

        let playerItem = AVPlayerItem(url: url)
        player = AVPlayer(playerItem: playerItem)

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

    func refreshNowPlaying() async {
        guard let channelId = channel?.id else { return }

        do {
            let response = try await api.getChannelNowPlaying(channelId: channelId)

            // Check if the current item changed
            if response.nowPlaying?.mediaId != currentMediaId {
                // Switch to new item
                nowPlaying = response.nowPlaying
                upNext = response.upNext
                elapsed = response.elapsed

                if let newItem = response.nowPlaying {
                    setupPlayer(for: newItem, seekTo: response.elapsed)
                }
            } else {
                // Just update up next
                upNext = response.upNext
            }
        } catch {
            // Silent fail on refresh
        }
    }

    func switchToChannel(channelId: Int64) async {
        // Load the new channel
        await loadChannel(channelId: channelId)
    }

    func advanceToNextItem() {
        guard !upNext.isEmpty else { return }

        let nextItem = upNext.removeFirst()
        nowPlaying = nextItem
        elapsed = 0

        setupPlayer(for: nextItem, seekTo: 0)
    }

    func handleVideoEnd() {
        // When video ends, advance to next item
        advanceToNextItem()
    }

    func play() {
        player?.play()
    }

    func pause() {
        player?.pause()
    }

    func cleanup() {
        if let observer = timeObserver, let player = player {
            player.removeTimeObserver(observer)
            timeObserver = nil
        }
        player?.pause()
        player = nil
        cancellables.removeAll()
        refreshTask?.cancel()
        refreshTask = nil
    }
}
