import SwiftUI

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
                        if let posterPath = show.posterPath {
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
                            Text(show.title)
                                .font(.largeTitle)
                                .fontWeight(.bold)

                            HStack(spacing: 20) {
                                if let year = show.year {
                                    Text(String(year))
                                }
                                if let rating = show.rating {
                                    HStack(spacing: 5) {
                                        Image(systemName: "star.fill")
                                            .foregroundColor(.yellow)
                                        Text(String(format: "%.1f", rating))
                                    }
                                }
                                if let seasonCount = show.seasonCount {
                                    Text("\(seasonCount) Season\(seasonCount == 1 ? "" : "s")")
                                }
                            }
                            .foregroundColor(.secondary)

                            if !show.genreList.isEmpty {
                                Text(show.genreList.joined(separator: " • "))
                                    .foregroundColor(.secondary)
                            }

                            HStack(spacing: 20) {
                                Button {
                                    Task {
                                        if let episode = await viewModel.playRandomEpisode() {
                                            selectedEpisode = episode
                                        }
                                    }
                                } label: {
                                    Label("Random Episode", systemImage: "shuffle")
                                        .font(.title3)
                                        .fontWeight(.semibold)
                                }
                                .disabled(viewModel.isLoadingRandom)
                            }
                            .padding(.top, 10)
                        }

                        Spacer()
                    }
                    .padding(50)
                }

                // Main content
                VStack(alignment: .leading, spacing: 40) {
                    // Overview
                    if let overview = show.overview, !overview.isEmpty {
                        VStack(alignment: .leading, spacing: 10) {
                            Text("Overview")
                                .font(.headline)

                            Text(overview)
                                .foregroundColor(.secondary)
                                .lineLimit(5)
                        }
                    }

                    // Seasons and Episodes
                    VStack(alignment: .leading, spacing: 20) {
                        Text("Episodes")
                            .font(.headline)

                        // Season selector
                        if !viewModel.seasons.isEmpty {
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 15) {
                                    ForEach(viewModel.seasons) { season in
                                        Button {
                                            viewModel.selectSeason(season.seasonNumber)
                                        } label: {
                                            Text(season.name ?? "Season \(season.seasonNumber)")
                                                .font(.title3)
                                                .padding(.horizontal, 20)
                                                .padding(.vertical, 10)
                                                .background(
                                                    viewModel.selectedSeason == season.seasonNumber ?
                                                    Color.white.opacity(0.2) : Color.clear
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
                            .padding(.vertical, 40)
                        } else if viewModel.currentEpisodes.isEmpty {
                            Text("No episodes available")
                                .foregroundColor(.secondary)
                                .padding(.vertical, 40)
                        } else {
                            LazyVStack(spacing: 20) {
                                ForEach(viewModel.currentEpisodes) { episode in
                                    Button {
                                        selectedEpisode = episode
                                    } label: {
                                        EpisodeRowContent(episode: episode)
                                    }
                                    .buttonStyle(.card)
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
                .padding(50)
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
            .frame(height: 600)
    }

    private var posterPlaceholder: some View {
        RoundedRectangle(cornerRadius: 10)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 200, height: 300)
            .overlay {
                Image(systemName: "tv")
                    .font(.system(size: 60))
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
        HStack(spacing: 20) {
            // Episode thumbnail
            if let stillPath = episode.stillPath {
                AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w300\(stillPath)")) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(width: 250, height: 140)
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                    case .failure, .empty:
                        thumbnailPlaceholder
                    @unknown default:
                        thumbnailPlaceholder
                    }
                }
                .frame(width: 250, height: 140)
            } else {
                thumbnailPlaceholder
            }

            // Episode info
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(episode.episodeCode)
                        .font(.caption)
                        .foregroundColor(.secondary)
                    if let runtime = episode.runtime {
                        Text("•")
                            .foregroundColor(.secondary)
                        Text("\(runtime) min")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Text(episode.title)
                    .font(.title3)
                    .fontWeight(.semibold)

                if let overview = episode.overview, !overview.isEmpty {
                    Text(overview)
                        .font(.body)
                        .foregroundColor(.secondary)
                        .lineLimit(2)
                }
            }

            Spacer()

            // Play indicator
            Image(systemName: "play.fill")
                .font(.title2)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 20)
    }

    private var thumbnailPlaceholder: some View {
        RoundedRectangle(cornerRadius: 8)
            .fill(Color.gray.opacity(0.3))
            .frame(width: 250, height: 140)
            .overlay {
                Image(systemName: "tv")
                    .font(.system(size: 40))
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
