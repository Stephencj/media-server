import SwiftUI
import AVKit

struct MediaDetailView: View {
    let media: Media

    @StateObject private var viewModel: MediaDetailViewModel
    @State private var showPlayer = false
    @State private var showAddToPlaylist = false

    init(media: Media) {
        self.media = media
        _viewModel = StateObject(wrappedValue: MediaDetailViewModel(media: media))
    }

    private var backdropPlaceholder: some View {
        Rectangle()
            .fill(
                LinearGradient(
                    colors: [.blue.opacity(0.3), .black],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )
            .frame(height: 250)
    }

    private var posterPlaceholder: some View {
        RoundedRectangle(cornerRadius: 8)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 120, height: 180)
            .overlay {
                Image(systemName: media.type == .movie ? "film" : "tv")
                    .font(.system(size: 36))
                    .foregroundColor(.gray)
            }
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                // Hero section with backdrop
                ZStack(alignment: .bottomLeading) {
                    // Backdrop image or placeholder
                    if let backdropPath = media.backdropPath {
                        AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w780\(backdropPath)")) { phase in
                            switch phase {
                            case .success(let image):
                                image
                                    .resizable()
                                    .aspectRatio(contentMode: .fill)
                                    .frame(height: 250)
                                    .clipped()
                                    .overlay {
                                        LinearGradient(
                                            colors: [.clear, .black.opacity(0.7)],
                                            startPoint: .top,
                                            endPoint: .bottom
                                        )
                                    }
                            case .failure, .empty:
                                backdropPlaceholder
                            @unknown default:
                                backdropPlaceholder
                            }
                        }
                        .frame(height: 250)
                    } else {
                        backdropPlaceholder
                    }

                    // Content overlay
                    HStack(alignment: .bottom, spacing: 16) {
                        // Poster
                        if let posterPath = media.posterPath {
                            AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w342\(posterPath)")) { phase in
                                switch phase {
                                case .success(let image):
                                    image
                                        .resizable()
                                        .aspectRatio(contentMode: .fill)
                                        .frame(width: 120, height: 180)
                                        .clipShape(RoundedRectangle(cornerRadius: 8))
                                case .failure, .empty:
                                    posterPlaceholder
                                @unknown default:
                                    posterPlaceholder
                                }
                            }
                            .frame(width: 120, height: 180)
                        } else {
                            posterPlaceholder
                        }

                        // Info
                        VStack(alignment: .leading, spacing: 8) {
                            Text(media.title)
                                .font(.title2)
                                .fontWeight(.bold)

                            HStack(spacing: 10) {
                                if let year = media.year {
                                    Text(String(year))
                                }
                                if let runtime = media.runtime {
                                    Text("\(runtime) min")
                                }
                                if let rating = media.rating {
                                    HStack(spacing: 3) {
                                        Image(systemName: "star.fill")
                                            .foregroundColor(.yellow)
                                        Text(String(format: "%.1f", rating))
                                    }
                                }
                                if let resolution = media.resolution {
                                    Text(resolution)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 2)
                                        .background(Color.white.opacity(0.2))
                                        .cornerRadius(4)
                                }
                            }
                            .font(.caption)
                            .foregroundColor(.secondary)

                            if !media.genreList.isEmpty {
                                Text(media.genreList.joined(separator: " \u{2022} "))
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                        }

                        Spacer()
                    }
                    .padding(16)
                }

                // Main content
                VStack(alignment: .leading, spacing: 20) {
                    // Play button row
                    HStack(spacing: 16) {
                        Button(action: { showPlayer = true }) {
                            Label(
                                viewModel.progress != nil ? "Resume" : "Play",
                                systemImage: "play.fill"
                            )
                            .font(.headline.weight(.semibold))
                            .frame(minWidth: 120)
                            .padding(.vertical, 10)
                            .padding(.horizontal, 20)
                            .background(Color.blue)
                            .foregroundColor(.white)
                            .cornerRadius(10)
                        }

                        Button {
                            showAddToPlaylist = true
                        } label: {
                            Label("Playlist", systemImage: "text.badge.plus")
                                .font(.subheadline)
                        }
                        .buttonStyle(.plain)

                        if let progress = viewModel.progress {
                            Text("\(progress.formattedRemaining) remaining")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                    }

                    // Progress bar
                    if let progress = viewModel.progress, progress.position > 0 {
                        ProgressView(value: progress.progressPercentage)
                            .tint(.blue)
                            .frame(maxWidth: .infinity)
                    }

                    // Overview
                    if let overview = media.overview, !overview.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Overview")
                                .font(.headline)

                            Text(overview)
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .lineLimit(5)
                        }
                    }

                    // Technical details
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Details")
                            .font(.headline)

                        LazyVGrid(columns: [
                            GridItem(.flexible()),
                            GridItem(.flexible()),
                            GridItem(.flexible())
                        ], spacing: 12) {
                            if let codec = media.videoCodec {
                                DetailItem(label: "Video", value: codec.uppercased())
                            }
                            if let codec = media.audioCodec {
                                DetailItem(label: "Audio", value: codec.uppercased())
                            }
                            if let duration = media.duration {
                                DetailItem(label: "Duration", value: media.formattedDuration)
                            }
                        }
                    }

                    // Audio tracks
                    if !media.decodedAudioTracks.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Audio Tracks")
                                .font(.headline)

                            ForEach(media.decodedAudioTracks) { track in
                                Text("\u{2022} \(track.displayName)")
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }
                        }
                    }

                    // Subtitles
                    if !media.decodedSubtitleTracks.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Subtitles")
                                .font(.headline)

                            ForEach(media.decodedSubtitleTracks) { track in
                                Text("\u{2022} \(track.displayName)")
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }
                        }
                    }
                }
                .padding(16)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .ignoresSafeArea(edges: .top)
        .fullScreenCover(isPresented: $showPlayer) {
            PlayerView(
                media: media,
                startPosition: viewModel.progress?.position ?? 0
            )
        }
        .sheet(isPresented: $showAddToPlaylist) {
            AddToPlaylistSheet(media: media)
        }
        .task {
            await viewModel.loadProgress()
        }
    }
}

struct DetailItem: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
            Text(value)
                .font(.subheadline)
        }
    }
}

#Preview {
    MediaDetailView(media: Media(
        id: 1,
        title: "Sample Movie",
        originalTitle: nil,
        type: .movie,
        year: 2024,
        overview: "A sample movie description that tells you about the plot.",
        posterPath: nil,
        backdropPath: nil,
        rating: 8.5,
        runtime: 120,
        genres: "Action, Adventure",
        tmdbId: nil,
        imdbId: nil,
        seasonCount: nil,
        episodeCount: nil,
        sourceId: nil,
        filePath: nil,
        fileSize: nil,
        duration: 7200,
        videoCodec: "h264",
        audioCodec: "aac",
        resolution: "1920x1080",
        audioTracks: nil,
        subtitleTracks: nil,
        createdAt: nil,
        updatedAt: nil
    ))
}
