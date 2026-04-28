import SwiftUI
import AVKit

struct TVShowDetailView: View {
    let show: Media

    @StateObject private var viewModel: TVShowDetailViewModel
    @State private var selectedEpisode: Episode?

    init(show: Media) {
        self.show = show
        _viewModel = StateObject(wrappedValue: TVShowDetailViewModel(showId: show.id))
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                // Hero section with backdrop
                ZStack(alignment: .bottomLeading) {
                    // Backdrop image or placeholder
                    if let backdropPath = show.backdropPath {
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
                        if let posterPath = show.posterPath {
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
                            Text(show.title)
                                .font(.title2)
                                .fontWeight(.bold)

                            HStack(spacing: 10) {
                                if let year = show.year {
                                    Text(String(year))
                                }
                                if let rating = show.rating {
                                    HStack(spacing: 3) {
                                        Image(systemName: "star.fill")
                                            .foregroundColor(.yellow)
                                        Text(String(format: "%.1f", rating))
                                    }
                                }
                                if let seasonCount = show.seasonCount {
                                    Text("\(seasonCount) Season\(seasonCount == 1 ? "" : "s")")
                                }
                            }
                            .font(.caption)
                            .foregroundColor(.secondary)

                            if !show.genreList.isEmpty {
                                Text(show.genreList.joined(separator: " \u{2022} "))
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }

                            HStack(spacing: 12) {
                                Button {
                                    Task {
                                        if let episode = await viewModel.playRandomEpisode() {
                                            selectedEpisode = episode
                                        }
                                    }
                                } label: {
                                    Label("Random Episode", systemImage: "shuffle")
                                        .font(.subheadline.weight(.semibold))
                                        .padding(.vertical, 8)
                                        .padding(.horizontal, 16)
                                        .background(Color.blue)
                                        .foregroundColor(.white)
                                        .cornerRadius(8)
                                }
                                .disabled(viewModel.isLoadingRandom)
                            }
                            .padding(.top, 4)
                        }

                        Spacer()
                    }
                    .padding(16)
                }

                // Main content
                VStack(alignment: .leading, spacing: 20) {
                    // Overview
                    if let overview = show.overview, !overview.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Overview")
                                .font(.headline)

                            Text(overview)
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .lineLimit(5)
                        }
                    }

                    // Seasons and Episodes
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Episodes")
                            .font(.headline)

                        // Season selector
                        if !viewModel.seasons.isEmpty {
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 10) {
                                    ForEach(viewModel.seasons) { season in
                                        Button {
                                            viewModel.selectSeason(season.seasonNumber)
                                        } label: {
                                            Text(season.name ?? "Season \(season.seasonNumber)")
                                                .font(.subheadline)
                                                .padding(.horizontal, 14)
                                                .padding(.vertical, 8)
                                                .background(
                                                    viewModel.selectedSeason == season.seasonNumber ?
                                                    Color.blue : Color.gray.opacity(0.2)
                                                )
                                                .foregroundColor(
                                                    viewModel.selectedSeason == season.seasonNumber ?
                                                    .white : .primary
                                                )
                                                .cornerRadius(8)
                                        }
                                        .buttonStyle(.plain)
                                    }
                                }
                            }
                        }

                        // Episode list
                        if viewModel.isLoading {
                            HStack {
                                Spacer()
                                ProgressView()
                                Spacer()
                            }
                            .padding(.vertical, 20)
                        } else if viewModel.currentEpisodes.isEmpty {
                            Text("No episodes available")
                                .foregroundColor(.secondary)
                                .padding(.vertical, 20)
                        } else {
                            LazyVStack(spacing: 12) {
                                ForEach(viewModel.currentEpisodes) { episode in
                                    Button {
                                        selectedEpisode = episode
                                    } label: {
                                        EpisodeRowContent(episode: episode)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }
                    }

                    if let error = viewModel.errorMessage {
                        Text(error)
                            .foregroundColor(.red)
                            .font(.caption)
                    }
                }
                .padding(16)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .ignoresSafeArea(edges: .top)
        .fullScreenCover(item: $selectedEpisode) { episode in
            PlayerView(
                media: convertEpisodeToMedia(episode),
                startPosition: 0
            )
        }
        .task {
            await viewModel.loadSeasons()
        }
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
                Image(systemName: "tv")
                    .font(.system(size: 36))
                    .foregroundColor(.gray)
            }
    }

    // Helper to convert Episode to Media for playback
    private func convertEpisodeToMedia(_ episode: Episode) -> Media {
        Media(
            id: episode.id,
            title: episode.title,
            originalTitle: nil,
            type: .episode,
            year: nil,
            overview: episode.overview,
            posterPath: episode.stillPath,
            backdropPath: show.backdropPath,
            rating: episode.rating,
            runtime: episode.runtime,
            genres: nil,
            tmdbId: nil,
            imdbId: nil,
            seasonCount: nil,
            episodeCount: nil,
            sourceId: nil,
            filePath: episode.filePath,
            fileSize: nil,
            duration: episode.duration,
            videoCodec: nil,
            audioCodec: nil,
            resolution: nil,
            audioTracks: nil,
            subtitleTracks: nil,
            createdAt: nil,
            updatedAt: nil
        )
    }
}

struct EpisodeRowContent: View {
    let episode: Episode

    var body: some View {
        HStack(spacing: 12) {
            // Episode thumbnail
            if let stillPath = episode.stillPath {
                AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w300\(stillPath)")) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: 140, height: 80)
                            .clipShape(RoundedRectangle(cornerRadius: 6))
                    case .failure, .empty:
                        thumbnailPlaceholder
                    @unknown default:
                        thumbnailPlaceholder
                    }
                }
                .frame(width: 140, height: 80)
            } else {
                thumbnailPlaceholder
            }

            // Episode info
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(episode.episodeCode)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                    if let runtime = episode.runtime {
                        Text("\u{2022}")
                            .foregroundColor(.secondary)
                        Text("\(runtime) min")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                Text(episode.title)
                    .font(.subheadline)
                    .fontWeight(.semibold)

                if let overview = episode.overview, !overview.isEmpty {
                    Text(overview)
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .lineLimit(2)
                }
            }

            Spacer()

            // Play indicator
            Image(systemName: "play.fill")
                .font(.body)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 8)
        .padding(.horizontal, 12)
        .background(Color(.systemGray6))
        .cornerRadius(10)
    }

    private var thumbnailPlaceholder: some View {
        RoundedRectangle(cornerRadius: 6)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 140, height: 80)
            .overlay {
                Image(systemName: "tv")
                    .font(.system(size: 24))
                    .foregroundColor(.gray)
            }
    }
}

#Preview {
    TVShowDetailView(show: Media(
        id: 1,
        title: "Sample TV Show",
        originalTitle: nil,
        type: .tvshow,
        year: 2024,
        overview: "A great TV show about interesting things.",
        posterPath: nil,
        backdropPath: nil,
        rating: 8.5,
        runtime: nil,
        genres: "Drama, Sci-Fi",
        tmdbId: nil,
        imdbId: nil,
        seasonCount: 3,
        episodeCount: 30,
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
    ))
}
