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
            // Video Player
            VideoPlayer(player: viewModel.player)
                .ignoresSafeArea()
                .onAppear {
                    viewModel.play()
                }
                .onDisappear {
                    viewModel.pause()
                }

            // Custom overlay for additional controls
            if viewModel.showControls {
                VStack {
                    // Top bar
                    HStack {
                        Button(action: { dismiss() }) {
                            Image(systemName: "xmark")
                                .font(.title2)
                                .padding()
                                .background(.ultraThinMaterial)
                                .clipShape(Circle())
                        }
                        .buttonStyle(.plain)

                        Spacer()

                        Text(media.title)
                            .font(.headline)

                        Spacer()

                        // Audio/Subtitle selector
                        Menu {
                            if !media.decodedAudioTracks.isEmpty {
                                Section("Audio") {
                                    ForEach(media.decodedAudioTracks) { track in
                                        Button(track.displayName) {
                                            viewModel.selectAudioTrack(track.index)
                                        }
                                    }
                                }
                            }

                            if !media.decodedSubtitleTracks.isEmpty {
                                Section("Subtitles") {
                                    Button("Off") {
                                        viewModel.selectSubtitle(nil)
                                    }
                                    ForEach(media.decodedSubtitleTracks) { track in
                                        Button(track.displayName) {
                                            viewModel.selectSubtitle(track.language)
                                        }
                                    }
                                }
                            }
                        } label: {
                            Image(systemName: "ellipsis.circle")
                                .font(.title2)
                                .padding()
                                .background(.ultraThinMaterial)
                                .clipShape(Circle())
                        }
                    }
                    .padding()

                    Spacer()

                    // Bottom progress bar
                    VStack(spacing: 10) {
                        // Time labels
                        HStack {
                            Text(formatTime(viewModel.currentTime))
                            Spacer()
                            Text("-\(formatTime(viewModel.duration - viewModel.currentTime))")
                        }
                        .font(.caption)
                        .foregroundColor(.white)

                        // Progress
                        ProgressView(value: viewModel.progress)
                            .tint(.white)
                    }
                    .padding()
                    .background(.ultraThinMaterial)
                }
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
        .gesture(
            TapGesture()
                .onEnded { _ in
                    withAnimation {
                        viewModel.showControls.toggle()
                    }
                }
        )
    }

    private func formatTime(_ seconds: Double) -> String {
        let hours = Int(seconds) / 3600
        let minutes = (Int(seconds) % 3600) / 60
        let secs = Int(seconds) % 60

        if hours > 0 {
            return String(format: "%d:%02d:%02d", hours, minutes, secs)
        }
        return String(format: "%d:%02d", minutes, secs)
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
