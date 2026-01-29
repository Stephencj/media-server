import SwiftUI

struct ChannelCardView: View {
    let channel: Channel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(channel.icon)
                    .font(.system(size: 50))

                Spacer()

                if let itemCount = channel.itemCount {
                    Text("\(itemCount) items")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            Text(channel.name)
                .font(.headline)
                .lineLimit(1)

            if let description = channel.description, !description.isEmpty {
                Text(description)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(2)
            }

            if !channel.formattedDuration.isEmpty {
                Text("Cycle: \(channel.formattedDuration)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding(20)
        .frame(minWidth: 300, minHeight: 150)
        .background(Color.gray.opacity(0.2))
        .cornerRadius(12)
    }
}

#Preview {
    HStack(spacing: 40) {
        ChannelCardView(channel: Channel(
            id: 1,
            name: "Movie Marathon",
            description: "Classic action movies",
            icon: "🎬",
            createdAt: "2024-01-01",
            updatedAt: "2024-01-01",
            totalDuration: 7200,
            itemCount: 5
        ))

        ChannelCardView(channel: Channel(
            id: 2,
            name: "Comedy Night",
            description: nil,
            icon: "😂",
            createdAt: "2024-01-01",
            updatedAt: "2024-01-01",
            totalDuration: nil,
            itemCount: 10
        ))
    }
    .padding(50)
}
