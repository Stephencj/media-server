import SwiftUI

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
            .frame(height: 600)
    }

    private var posterPlaceholder: some View {
        RoundedRectangle(cornerRadius: 10)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 200, height: 300)
            .overlay {
                Image(systemName: media.type == .movie ? "film" : "tv")
                    .font(.system(size: 60))
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
                        AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w1280\(backdropPath)")) { phase in
                            switch phase {
                            case .success(let image):
                                image
                                    .resizable()
                                    .aspectRatio(contentMode: .fill)
                                    .frame(height: 600)
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
                        .frame(height: 600)
                    } else {
                        backdropPlaceholder
                    }

                    // Content overlay
                    HStack(alignment: .bottom, spacing: 40) {
                        // Poster
                        if let posterPath = media.posterPath {
                            AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w342\(posterPath)")) { phase in
                                switch phase {
                                case .success(let image):
                                    image
                                        .resizable()
                                        .aspectRatio(contentMode: .fill)
                                        .frame(width: 200, height: 300)
                                        .clipShape(RoundedRectangle(cornerRadius: 10))
                                case .failure, .empty:
                                    posterPlaceholder
                                @unknown default:
                                    posterPlaceholder
                                }
                            }
                            .frame(width: 200, height: 300)
                        } else {
                            posterPlaceholder
                        }

                        // Info
                        VStack(alignment: .leading, spacing: 20) {
                            Text(media.title)
                                .font(.largeTitle)
                                .fontWeight(.bold)

                            HStack(spacing: 20) {
                                if let year = media.year {
                                    Text(String(year))
                                }
                                if let runtime = media.runtime {
                                    Text("\(runtime) min")
                                }
                                if let rating = media.rating {
                                    HStack(spacing: 5) {
                                        Image(systemName: "star.fill")
                                            .foregroundColor(.yellow)
                                        Text(String(format: "%.1f", rating))
                                    }
                                }
                                if let resolution = media.resolution {
                                    Text(resolution)
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 4)
                                        .background(Color.white.opacity(0.2))
                                        .cornerRadius(4)
                                }
                            }
                            .foregroundColor(.secondary)

                            if !media.genreList.isEmpty {
                                Text(media.genreList.joined(separator: " • "))
                                    .foregroundColor(.secondary)
                            }
                        }

                        Spacer()
                    }
                    .padding(50)
                }

                // Main content
                VStack(alignment: .leading, spacing: 40) {
                    // Play button row
                    HStack(spacing: 30) {
                        Button(action: { showPlayer = true }) {
                            Label(
                                viewModel.progress != nil ? "Resume" : "Play",
                                systemImage: "play.fill"
                            )
                            .font(.title3)
                            .fontWeight(.semibold)
                            .frame(minWidth: 200)
                        }

                        Button {
                            showAddToPlaylist = true
                        } label: {
                            Label("Add to Playlist", systemImage: "text.badge.plus")
                                .font(.title3)
                        }
                        .buttonStyle(.plain)

                        if let progress = viewModel.progress {
                            Text("\(progress.formattedRemaining) remaining")
                                .foregroundColor(.secondary)
                        }
                    }

                    // Progress bar
                    if let progress = viewModel.progress, progress.position > 0 {
                        ProgressView(value: progress.progressPercentage)
                            .tint(.blue)
                            .frame(maxWidth: 400)
                    }

                    // Overview
                    if let overview = media.overview, !overview.isEmpty {
                        VStack(alignment: .leading, spacing: 10) {
                            Text("Overview")
                                .font(.headline)

                            Text(overview)
                                .foregroundColor(.secondary)
                                .lineLimit(5)
                        }
                    }

                    // Technical details
                    VStack(alignment: .leading, spacing: 20) {
                        Text("Details")
                            .font(.headline)

                        LazyVGrid(columns: [
                            GridItem(.flexible()),
                            GridItem(.flexible()),
                            GridItem(.flexible())
                        ], spacing: 20) {
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
                        VStack(alignment: .leading, spacing: 10) {
                            Text("Audio Tracks")
                                .font(.headline)

                            ForEach(media.decodedAudioTracks) { track in
                                Text("• \(track.displayName)")
                                    .foregroundColor(.secondary)
                            }
                        }
                    }

                    // Subtitles
                    if !media.decodedSubtitleTracks.isEmpty {
                        VStack(alignment: .leading, spacing: 10) {
                            Text("Subtitles")
                                .font(.headline)

                            ForEach(media.decodedSubtitleTracks) { track in
                                Text("• \(track.displayName)")
                                    .foregroundColor(.secondary)
                            }
                        }
                    }
                }
                .padding(50)
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
        VStack(alignment: .leading, spacing: 5) {
            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
            Text(value)
                .font(.body)
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
