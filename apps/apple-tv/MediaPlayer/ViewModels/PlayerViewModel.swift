import Foundation
import AVKit
import Combine

@MainActor
class PlayerViewModel: ObservableObject {
    @Published var player: AVPlayer
    @Published var isBuffering = false
    @Published var showControls = false
    @Published var currentTime: Double = 0
    @Published var duration: Double = 0
    @Published var progress: Double = 0

    private let media: Media
    private let startPosition: Int
    private let api = APIClient.shared

    private var timeObserver: Any?
    private var cancellables = Set<AnyCancellable>()
    private var lastSavedPosition: Int = 0
    private let saveInterval: Int = 10 // Save every 10 seconds of playback

    init(media: Media, startPosition: Int) {
        self.media = media
        self.startPosition = startPosition

        // Create player with stream URL
        let streamURL: URL?
        if let directURL = api.getDirectPlayURL(mediaId: media.id) {
            streamURL = directURL
        } else {
            streamURL = api.getStreamURL(mediaId: media.id)
        }

        if let url = streamURL {
            // Token is included in URL query parameter for AVPlayer compatibility
            let playerItem = AVPlayerItem(url: url)
            self.player = AVPlayer(playerItem: playerItem)
        } else {
            self.player = AVPlayer()
        }

        setupObservers()
    }

    func cleanup() {
        if let observer = timeObserver {
            player.removeTimeObserver(observer)
            timeObserver = nil
        }
        player.pause()
    }

    private func setupObservers() {
        // Time observer for progress updates
        let interval = CMTime(seconds: 1, preferredTimescale: CMTimeScale(NSEC_PER_SEC))
        timeObserver = player.addPeriodicTimeObserver(forInterval: interval, queue: .main) { [weak self] time in
            Task { @MainActor in
                self?.updateTime(time)
            }
        }

        // Buffering state
        player.publisher(for: \.timeControlStatus)
            .receive(on: DispatchQueue.main)
            .sink { [weak self] status in
                self?.isBuffering = status == .waitingToPlayAtSpecifiedRate
            }
            .store(in: &cancellables)

        // Duration
        player.publisher(for: \.currentItem?.duration)
            .compactMap { $0 }
            .filter { $0.isNumeric }
            .receive(on: DispatchQueue.main)
            .sink { [weak self] duration in
                self?.duration = duration.seconds
            }
            .store(in: &cancellables)
    }

    private func updateTime(_ time: CMTime) {
        currentTime = time.seconds
        if duration > 0 {
            progress = currentTime / duration
        }

        // Save progress periodically
        let currentSeconds = Int(currentTime)
        if currentSeconds - lastSavedPosition >= saveInterval {
            lastSavedPosition = currentSeconds
            Task {
                await saveProgress()
            }
        }
    }

    func play() {
        // Seek to start position if provided
        if startPosition > 0 {
            let time = CMTime(seconds: Double(startPosition), preferredTimescale: CMTimeScale(NSEC_PER_SEC))
            player.seek(to: time)
        }

        player.play()

        // Auto-hide controls after a delay
        Task {
            try? await Task.sleep(nanoseconds: 3_000_000_000)
            showControls = false
        }
    }

    func pause() {
        player.pause()
        Task {
            await saveProgress()
        }
    }

    func seek(to seconds: Double) {
        let time = CMTime(seconds: seconds, preferredTimescale: CMTimeScale(NSEC_PER_SEC))
        player.seek(to: time)
    }

    func selectAudioTrack(_ index: Int) {
        guard let playerItem = player.currentItem else { return }

        Task {
            if let group = try? await playerItem.asset.loadMediaSelectionGroup(for: .audible),
               index < group.options.count {
                playerItem.select(group.options[index], in: group)
            }
        }
    }

    func selectSubtitle(_ language: String?) {
        guard let playerItem = player.currentItem else { return }

        Task {
            if let group = try? await playerItem.asset.loadMediaSelectionGroup(for: .legible) {
                if let language = language {
                    if let option = group.options.first(where: {
                        $0.locale?.language.languageCode?.identifier == language
                    }) {
                        playerItem.select(option, in: group)
                    }
                } else {
                    playerItem.select(nil, in: group)
                }
            }
        }
    }

    func markAsCompleted() {
        Task {
            do {
                _ = try await api.updateProgress(
                    mediaId: media.id,
                    position: Int(currentTime),
                    duration: Int(duration),
                    mediaType: media.type.rawValue,
                    completed: true
                )
            } catch {
                // Silent fail
            }
        }
    }

    private func saveProgress() async {
        guard currentTime > 0, duration > 0 else { return }

        let completed = progress > 0.95

        do {
            _ = try await api.updateProgress(
                mediaId: media.id,
                position: Int(currentTime),
                duration: Int(duration),
                mediaType: media.type.rawValue,
                completed: completed
            )
        } catch {
            // Silent fail - don't interrupt playback for progress save failures
        }
    }
}
