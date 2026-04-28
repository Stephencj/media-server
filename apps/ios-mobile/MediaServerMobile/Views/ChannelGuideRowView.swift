import SwiftUI

struct ChannelGuideRowView: View {
    let entry: ChannelGuideEntry
    let onSelect: () -> Void

    var body: some View {
        Button(action: onSelect) {
            HStack(spacing: 12) {
                // Channel icon and name (fixed width)
                HStack(spacing: 6) {
                    Text(entry.channel.icon)
                        .font(.system(size: 24))

                    Text(entry.channel.name)
                        .font(.subheadline)
                        .fontWeight(.medium)
                        .lineLimit(1)
                }
                .frame(width: 80, alignment: .leading)

                // Now Playing
                if let nowPlaying = entry.nowPlaying {
                    ProgramBlockView(
                        item: nowPlaying,
                        elapsed: entry.elapsed,
                        isNowPlaying: true
                    )
                } else {
                    Text("No content")
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .frame(minWidth: 120, alignment: .leading)
                }

                // Up Next (show only 1 on iPhone)
                ForEach(entry.upNext.prefix(1)) { item in
                    ProgramBlockView(
                        item: item,
                        elapsed: 0,
                        isNowPlaying: false
                    )
                }

                Spacer()
            }
            .padding(.vertical, 10)
            .padding(.horizontal, 12)
            .background(Color.gray.opacity(0.1))
            .cornerRadius(10)
        }
        .buttonStyle(.plain)
    }
}

struct ProgramBlockView: View {
    let item: ChannelScheduleItem
    let elapsed: Int
    let isNowPlaying: Bool

    var body: some View {
        HStack(spacing: 8) {
            // Poster thumbnail
            if let posterPath = item.posterPath {
                AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w92\(posterPath)")) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    Color.gray.opacity(0.3)
                }
                .frame(width: 36, height: 54)
                .cornerRadius(4)
            } else {
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color.gray.opacity(0.3))
                    .frame(width: 36, height: 54)
            }

            VStack(alignment: .leading, spacing: 2) {
                // Show title for episodes
                if let showTitle = item.showTitle {
                    Text(showTitle)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }

                Text(item.title)
                    .font(.caption)
                    .fontWeight(isNowPlaying ? .semibold : .regular)
                    .lineLimit(2)

                if isNowPlaying {
                    let remaining = item.duration - elapsed
                    Text(formatDuration(remaining) + " left")
                        .font(.caption2)
                        .foregroundColor(.green)
                } else {
                    Text(item.formattedDuration)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }
        }
        .frame(minWidth: 120, maxWidth: 180, alignment: .leading)
        .padding(8)
        .background(isNowPlaying ? Color.blue.opacity(0.15) : Color.gray.opacity(0.1))
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
        icon: "\u{1F3AC}",
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
        .padding(16)
}
