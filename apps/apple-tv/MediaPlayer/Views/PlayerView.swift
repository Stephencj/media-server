import SwiftUI
import AVKit

struct PlayerView: View {
    let media: Media
    let startPosition: Int

    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: PlayerViewModel

    init(media: Media, startPosition: Int = 0) {
        self.media = media
        self.startPosition = startPosition
        _viewModel = StateObject(wrappedValue: PlayerViewModel(media: media, startPosition: startPosition))
    }

    var body: some View {
        ZStack {
            // Video Player - use AVPlayerViewController for proper tvOS controls
            AVPlayerView(player: viewModel.player)
                .ignoresSafeArea()
                .onAppear {
                    viewModel.play()
                }
                .onDisappear {
                    viewModel.cleanup()
                }

            // Loading indicator
            if viewModel.isBuffering {
                ProgressView()
                    .scaleEffect(2)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { _ in
            viewModel.markAsCompleted()
            dismiss()
        }
        // Menu button dismisses
        .onExitCommand {
            viewModel.pause()
            dismiss()
        }
    }
}

// Wrap AVPlayerViewController for proper tvOS playback controls
struct AVPlayerView: UIViewControllerRepresentable {
    let player: AVPlayer

    func makeUIViewController(context: Context) -> AVPlayerViewController {
        let controller = AVPlayerViewController()
        controller.player = player
        controller.showsPlaybackControls = true
        return controller
    }

    func updateUIViewController(_ uiViewController: AVPlayerViewController, context: Context) {
        uiViewController.player = player
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
