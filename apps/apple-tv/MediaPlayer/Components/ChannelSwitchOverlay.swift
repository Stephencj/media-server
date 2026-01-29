import SwiftUI

enum ChannelSwitchDirection {
    case up
    case down

    var icon: String {
        switch self {
        case .up: return "chevron.up"
        case .down: return "chevron.down"
        }
    }
}

struct ChannelSwitchOverlay: View {
    let channel: Channel
    let nowPlaying: ChannelScheduleItem?
    let direction: ChannelSwitchDirection

    var body: some View {
        HStack(spacing: 20) {
            // Direction indicator
            Image(systemName: direction.icon)
                .font(.system(size: 24, weight: .bold))
                .foregroundColor(.white.opacity(0.7))

            // Channel icon
            Text(channel.icon)
                .font(.system(size: 48))

            VStack(alignment: .leading, spacing: 6) {
                Text(channel.name)
                    .font(.title2)
                    .fontWeight(.semibold)
                    .foregroundColor(.white)

                if let nowPlaying = nowPlaying {
                    HStack(spacing: 8) {
                        Text("Now:")
                            .font(.subheadline)
                            .foregroundColor(.white.opacity(0.7))

                        if let showTitle = nowPlaying.showTitle {
                            Text("\(showTitle) - \(nowPlaying.title)")
                                .font(.subheadline)
                                .foregroundColor(.white.opacity(0.9))
                                .lineLimit(1)
                        } else {
                            Text(nowPlaying.title)
                                .font(.subheadline)
                                .foregroundColor(.white.opacity(0.9))
                                .lineLimit(1)
                        }
                    }
                }
            }

            // Direction indicator (other side)
            Image(systemName: direction.icon)
                .font(.system(size: 24, weight: .bold))
                .foregroundColor(.white.opacity(0.7))
        }
        .padding(.horizontal, 32)
        .padding(.vertical, 20)
        .background(
            RoundedRectangle(cornerRadius: 16)
                .fill(Color.black.opacity(0.85))
                .shadow(color: .black.opacity(0.3), radius: 20)
        )
    }
}

#Preview {
    ZStack {
        Color.gray

        ChannelSwitchOverlay(
            channel: Channel(
                id: 1,
                name: "Movie Marathon",
                description: "Classic action movies",
                icon: "🎬",
                createdAt: "2024-01-01",
                updatedAt: "2024-01-01",
                totalDuration: 7200,
                itemCount: 5
            ),
            nowPlaying: ChannelScheduleItem(
                id: 1,
                mediaId: 1,
                mediaType: "movie",
                title: "Die Hard",
                duration: 7920,
                cumulativeStart: 0,
                posterPath: nil,
                backdropPath: nil,
                showTitle: nil
            ),
            direction: .up
        )
    }
}
