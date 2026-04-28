import SwiftUI

struct ChannelGuideView: View {
    let channels: [Channel]
    let onSelectChannel: (Channel, Int) -> Void

    @StateObject private var viewModel = ChannelGuideViewModel()

    var body: some View {
        Group {
            if viewModel.isLoading && viewModel.entries.isEmpty {
                ProgressView()
            } else if viewModel.entries.isEmpty {
                VStack(spacing: 12) {
                    Image(systemName: "tv.inset.filled")
                        .font(.system(size: 40))
                        .foregroundColor(.secondary)
                    Text("No Channels")
                        .font(.headline)
                    Text("Create channels in the web interface to watch here")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
            } else {
                ScrollView {
                    LazyVStack(spacing: 8) {
                        ForEach(viewModel.entries) { entry in
                            ChannelGuideRowView(entry: entry) {
                                if let index = channels.firstIndex(where: { $0.id == entry.channel.id }) {
                                    onSelectChannel(entry.channel, index)
                                }
                            }
                        }
                    }
                    .padding(16)
                }
            }
        }
        .task {
            await viewModel.loadGuide()
            viewModel.startAutoRefresh()
        }
        .onDisappear {
            viewModel.stopAutoRefresh()
        }
    }
}

#Preview {
    ChannelGuideView(
        channels: [
            Channel(
                id: 1,
                name: "Movie Marathon",
                description: "Classic action movies",
                icon: "\u{1F3AC}",
                createdAt: "2024-01-01",
                updatedAt: "2024-01-01",
                totalDuration: 7200,
                itemCount: 5
            )
        ],
        onSelectChannel: { _, _ in }
    )
}
