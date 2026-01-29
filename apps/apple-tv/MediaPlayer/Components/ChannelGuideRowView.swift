import SwiftUI

struct ChannelGuideRowView: View {
    let entry: ChannelGuideEntry
    let onSelect: () -> Void

    @FocusState private var isFocused: Bool

    var body: some View {
        Button(action: onSelect) {
            HStack(spacing: 24) {
                // Channel icon and name (fixed width)
                HStack(spacing: 12) {
                    Text(entry.channel.icon)
                        .font(.system(size: 36))

                    Text(entry.channel.name)
                        .font(.headline)
                        .lineLimit(1)
                }
                .frame(width: 200, alignment: .leading)

                // Now Playing
                if let nowPlaying = entry.nowPlaying {
                    ProgramBlockView(
                        item: nowPlaying,
                        elapsed: entry.elapsed,
                        isNowPlaying: true
                    )
                } else {
                    Text("No content")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .frame(minWidth: 200, alignment: .leading)
                }

                // Up Next programs (show first 2)
                ForEach(entry.upNext.prefix(2)) { item in
                    ProgramBlockView(
                        item: item,
                        elapsed: 0,
                        isNowPlaying: false
                    )
                }

                Spacer()
            }
            .padding(.vertical, 16)
            .padding(.horizontal, 20)
            .background(isFocused ? Color.white.opacity(0.2) : Color.clear)
            .cornerRadius(12)
        }
        .buttonStyle(.plain)
        .focused($isFocused)
    }
}

struct ProgramBlockView: View {
    let item: ChannelScheduleItem
    let elapsed: Int
    let isNowPlaying: Bool

    var body: some View {
        HStack(spacing: 12) {
            // Poster thumbnail
            if let posterPath = item.posterPath {
                AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w92\(posterPath)")) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    Color.gray.opacity(0.3)
                }
                .frame(width: 50, height: 75)
                .cornerRadius(4)
            } else {
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color.gray.opacity(0.3))
                    .frame(width: 50, height: 75)
            }

            VStack(alignment: .leading, spacing: 4) {
                // Show title for episodes
                if let showTitle = item.showTitle {
                    Text(showTitle)
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }

                Text(item.title)
                    .font(.subheadline)
                    .fontWeight(isNowPlaying ? .semibold : .regular)
                    .lineLimit(2)

                HStack(spacing: 8) {
                    if isNowPlaying {
                        // Show remaining time
                        let remaining = item.duration - elapsed
                        Text(formatDuration(remaining) + " left")
                            .font(.caption)
                            .foregroundColor(.green)
                    } else {
                        Text(item.formattedDuration)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }
        }
        .frame(minWidth: 200, maxWidth: 280, alignment: .leading)
        .padding(12)
        .background(isNowPlaying ? Color.blue.opacity(0.2) : Color.gray.opacity(0.15))
        .cornerRadius(8)
    }

    private func formatDuration(_ seconds: Int) -> String {
        let hours = seconds / 3600
        let minutes = (seconds % 3600) / 60
        if hours > 0 {
            return "\(hours)h \(minutes)m"
        }
        return "\(minutes)m"
    }
}

#Preview {
    let channel = Channel(
        id: 1,
        name: "Movie Marathon",
        description: "Classic action movies",
        icon: "🎬",
        createdAt: "2024-01-01",
        updatedAt: "2024-01-01",
        totalDuration: 7200,
        itemCount: 5
    )

    let nowPlaying = ChannelScheduleItem(
        id: 1,
        mediaId: 1,
        mediaType: "movie",
        title: "Die Hard",
        duration: 7920,
        cumulativeStart: 0,
        posterPath: nil,
        backdropPath: nil,
        showTitle: nil
    )

    let upNext = ChannelScheduleItem(
        id: 2,
        mediaId: 2,
        mediaType: "movie",
        title: "Terminator 2",
        duration: 8640,
        cumulativeStart: 7920,
        posterPath: nil,
        backdropPath: nil,
        showTitle: nil
    )

    let entry = ChannelGuideEntry(
        id: 1,
        channel: channel,
        nowPlaying: nowPlaying,
        upNext: [upNext],
        elapsed: 1800
    )

    return ChannelGuideRowView(entry: entry, onSelect: {})
        .padding(50)
        .background(Color.black)
}
