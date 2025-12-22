import SwiftUI

struct MediaCardView: View {
    let media: Media
    var progress: WatchProgress? = nil
    var onSave: ((Media) -> Void)? = nil
    var onMarkWatched: ((Media) -> Void)? = nil

    private var placeholderView: some View {
        RoundedRectangle(cornerRadius: 10)
            .fill(
                LinearGradient(
                    colors: [.blue.opacity(0.3), .purple.opacity(0.3)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .overlay {
                Image(systemName: media.type == .movie ? "film" : "tv")
                    .font(.system(size: 50))
                    .foregroundColor(.white.opacity(0.5))
            }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Poster
            ZStack(alignment: .bottomLeading) {
                if let posterPath = media.posterPath {
                    AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w342\(posterPath)")) { phase in
                        switch phase {
                        case .success(let image):
                            image
                                .resizable()
                                .aspectRatio(contentMode: .fill)
                        case .failure, .empty:
                            placeholderView
                        @unknown default:
                            placeholderView
                        }
                    }
                    .aspectRatio(2/3, contentMode: .fit)
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                } else {
                    placeholderView
                        .aspectRatio(2/3, contentMode: .fit)
                }

                // Progress bar overlay
                if let progress = progress, progress.position > 0, !progress.completed {
                    VStack {
                        Spacer()
                        GeometryReader { geo in
                            Rectangle()
                                .fill(Color.red)
                                .frame(width: geo.size.width * progress.progressPercentage, height: 4)
                        }
                        .frame(height: 4)
                    }
                }

                // Type badge
                if media.type == .tvshow {
                    Text("TV")
                        .font(.caption2)
                        .fontWeight(.bold)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.blue)
                        .cornerRadius(4)
                        .padding(8)
                }
            }

            // Title
            Text(media.title)
                .font(.callout)
                .fontWeight(.medium)
                .lineLimit(2)
                .multilineTextAlignment(.leading)

            // Year and rating
            HStack(spacing: 10) {
                if let year = media.year {
                    Text(String(year))
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                if let rating = media.rating, rating > 0 {
                    HStack(spacing: 2) {
                        Image(systemName: "star.fill")
                            .font(.caption2)
                            .foregroundColor(.yellow)
                        Text(String(format: "%.1f", rating))
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }
        }
        .frame(width: 250)
        .contextMenu {
            Button {
                onSave?(media)
            } label: {
                Label("Save", systemImage: "bookmark")
            }

            Button {
                onMarkWatched?(media)
            } label: {
                Label("Mark as Watched", systemImage: "checkmark.circle")
            }
        }
    }
}

#Preview {
    HStack(spacing: 40) {
        MediaCardView(media: Media(
            id: 1,
            title: "Sample Movie with a Very Long Title",
            originalTitle: nil,
            type: .movie,
            year: 2024,
            overview: nil,
            posterPath: nil,
            backdropPath: nil,
            rating: 8.5,
            runtime: 120,
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
        ))

        MediaCardView(
            media: Media(
                id: 2,
                title: "TV Show",
                originalTitle: nil,
                type: .tvshow,
                year: 2023,
                overview: nil,
                posterPath: nil,
                backdropPath: nil,
                rating: 9.0,
                runtime: nil,
                genres: nil,
                tmdbId: nil,
                imdbId: nil,
                seasonCount: 3,
                episodeCount: 24,
                sourceId: nil,
                filePath: nil,
                fileSize: nil,
                duration: nil,
                videoCodec: nil,
                audioCodec: nil,
                resolution: nil,
                audioTracks: nil,
                subtitleTracks: nil,
                createdAt: nil,
                updatedAt: nil
            ),
            progress: WatchProgress(
                id: 1,
                userId: 1,
                mediaId: 2,
                mediaType: "tvshow",
                position: 1800,
                duration: 3600,
                completed: false,
                updatedAt: nil
            )
        )
    }
    .padding(50)
}
