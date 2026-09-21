import SwiftUI

/// The guide on a phone. A time axis doesn't survive at this width, so each
/// channel gets one row carrying what the axis was for: what's on, how far
/// through it is, and what follows.
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
                        .multilineTextAlignment(.center)
                }
                .padding(32)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(viewModel.entries) { entry in
                            ChannelGuideRowView(
                                entry: entry,
                                now: viewModel.now,
                                onSelect: { tune(to: entry) }
                            )
                            Divider().padding(.leading, 76)
                        }
                    }
                }
                .refreshable { await viewModel.loadGuide() }
            }
        }
        .task {
            await viewModel.loadGuide()
            viewModel.startAutoRefresh()
        }
        .onDisappear { viewModel.stopAutoRefresh() }
    }

    private func tune(to entry: ChannelGuideEntry) {
        guard let index = channels.firstIndex(where: { $0.id == entry.channel.id }) else { return }
        onSelectChannel(entry.channel, index)
    }
}
