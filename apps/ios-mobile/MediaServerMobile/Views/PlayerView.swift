import SwiftUI
import AVKit
import UIKit

enum SeekDirection: Equatable { case forward, backward }

struct PlayerView: View {
    let media: Media
    let startPosition: Int

    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: PlayerViewModel
    @State private var seekFlash: SeekDirection? = nil
    @State private var flashTask: Task<Void, Never>? = nil

    init(media: Media, startPosition: Int = 0) {
        self.media = media
        self.startPosition = startPosition
        _viewModel = StateObject(wrappedValue: PlayerViewModel(media: media, startPosition: startPosition))
    }

    var body: some View {
        ZStack {
            AVPlayerView(player: viewModel.player)
                .ignoresSafeArea()
                .onAppear {
                    AudioSessionManager.shared.activatePlayback()
                    Task { await viewModel.loadAndPlay() }
                }
                .onDisappear {
                    viewModel.cleanup()
                }

            if viewModel.isBuffering {
                ProgressView()
                    .scaleEffect(1.5)
                    .tint(.white)
            }

            // Double-tap-to-seek catchers on the leading + trailing 30%
            // of the player. Center 40% stays clear so AVKit's tap-to-toggle
            // works naturally.
            GeometryReader { geo in
                HStack(spacing: 0) {
                    sideTapCatcher(side: .backward)
                        .frame(width: geo.size.width * 0.30)
                    Spacer(minLength: 0)
                    sideTapCatcher(side: .forward)
                        .frame(width: geo.size.width * 0.30)
                }
                .frame(width: geo.size.width, height: geo.size.height)
            }
            .ignoresSafeArea()

            if let errorMessage = viewModel.errorMessage {
                VStack(spacing: 20) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 60))
                        .foregroundColor(.yellow)

                    Text("Playback error")
                        .font(.title)
                        .foregroundColor(.white)

                    Text(errorMessage)
                        .font(.body)
                        .foregroundColor(.white.opacity(0.8))
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 40)

                    Button("Go Back") {
                        dismiss()
                    }
                    .padding(.top, 20)
                }
                .padding(40)
                .background(Color.black.opacity(0.8))
                .cornerRadius(20)
            }

            // Transient ±10s flash anchored to whichever side was tapped.
            if let dir = seekFlash {
                HStack {
                    if dir == .backward {
                        SeekFlashView(direction: .backward)
                        Spacer()
                    } else {
                        Spacer()
                        SeekFlashView(direction: .forward)
                    }
                }
                .padding(.horizontal, 48)
                .transition(.opacity.combined(with: .scale(scale: 0.85)))
                .allowsHitTesting(false)
            }

            // Top bar: close (leading) + title (centered).
            ZStack {
                HStack {
                    Button {
                        viewModel.pause()
                        dismiss()
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.title)
                            .foregroundColor(.white.opacity(0.8))
                            .padding(16)
                    }
                    Spacer()
                }
                TitleBar(title: media.title)
            }
            .frame(maxHeight: .infinity, alignment: .top)
        }
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { _ in
            viewModel.markAsCompleted()
            dismiss()
        }
    }

    private func sideTapCatcher(side: SeekDirection) -> some View {
        Color.clear
            .contentShape(Rectangle())
            .onTapGesture(count: 2) { triggerSeek(side) }
    }

    private func triggerSeek(_ side: SeekDirection) {
        flashTask?.cancel()
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        switch side {
        case .backward:
            viewModel.seek(to: max(0, viewModel.currentTime - 10))
        case .forward:
            let target = viewModel.duration > 0
                ? min(viewModel.duration, viewModel.currentTime + 10)
                : viewModel.currentTime + 10
            viewModel.seek(to: target)
        }
        withAnimation(.easeOut(duration: 0.15)) { seekFlash = side }
        flashTask = Task {
            try? await Task.sleep(nanoseconds: 500_000_000)
            if !Task.isCancelled {
                await MainActor.run {
                    withAnimation(.easeOut(duration: 0.35)) { seekFlash = nil }
                }
            }
        }
    }
}

private struct TitleBar: View {
    let title: String
    var body: some View {
        Text(title)
            .font(.headline.weight(.semibold))
            .foregroundColor(.white)
            .lineLimit(1)
            .truncationMode(.tail)
            .shadow(color: .black.opacity(0.6), radius: 3, x: 0, y: 1)
            .padding(.horizontal, 60)
            .padding(.top, 18)
            .frame(maxWidth: .infinity, alignment: .center)
            .allowsHitTesting(false)
    }
}

private struct SeekFlashView: View {
    let direction: SeekDirection
    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: direction == .forward ? "goforward.10" : "gobackward.10")
                .font(.system(size: 38, weight: .semibold))
            Text("10s").font(.caption.weight(.semibold))
        }
        .foregroundColor(.white)
        .padding(22)
        .background(Circle().fill(.black.opacity(0.55)))
    }
}

/// Wraps `AVPlayerViewController` so SwiftUI gets Apple's full playback chrome
/// plus Picture-in-Picture and AirPlay support.
struct AVPlayerView: UIViewControllerRepresentable {
    let player: AVPlayer?

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeUIViewController(context: Context) -> AVPlayerViewController {
        let controller = AVPlayerViewController()
        controller.player = player
        controller.showsPlaybackControls = true
        controller.allowsPictureInPicturePlayback = true
        controller.canStartPictureInPictureAutomaticallyFromInline = true
        controller.videoGravity = .resizeAspect
        controller.delegate = context.coordinator
        return controller
    }

    func updateUIViewController(_ uiViewController: AVPlayerViewController, context: Context) {
        if uiViewController.player !== player {
            uiViewController.player = player
        }
    }

    final class Coordinator: NSObject, AVPlayerViewControllerDelegate {
        // Required for the user to be able to tap the floating PiP window
        // and return to the full player UI.
        func playerViewController(
            _ playerViewController: AVPlayerViewController,
            restoreUserInterfaceForPictureInPictureStopWithCompletionHandler completionHandler: @escaping (Bool) -> Void
        ) {
            completionHandler(true)
        }
    }
}

#Preview {
    PlayerView(
        media: Media(
            id: 1,
            title: "Sample Movie",
            originalTitle: nil,
            type: .movie,
            year: 2024,
            overview: nil,
            posterPath: nil,
            backdropPath: nil,
            rating: nil,
            runtime: nil,
            genres: nil,
            tmdbId: nil,
            imdbId: nil,
            seasonCount: nil,
            episodeCount: nil,
            sourceId: nil,
            filePath: nil,
            fileSize: nil,
            duration: 7200,
            videoCodec: nil,
            audioCodec: nil,
            resolution: nil,
            audioTracks: nil,
            subtitleTracks: nil,
            createdAt: nil,
            updatedAt: nil
        )
    )
}
