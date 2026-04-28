import SwiftUI
import AVKit

struct ChannelPlayerView: View {
    let channels: [Channel]
    let startIndex: Int

    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = ChannelPlayerViewModel()

    private var currentChannel: Channel? {
        guard startIndex >= 0 && startIndex < channels.count else { return nil }
        return channels[startIndex]
    }

    var body: some View {
        ZStack {
            // Video Player
            if let player = viewModel.player {
                VideoPlayer(player: player)
                    .ignoresSafeArea()
            } else {
                Color.black.ignoresSafeArea()
            }

            // Loading indicator
            if viewModel.isBuffering || viewModel.isLoading {
                ProgressView()
                    .scaleEffect(1.5)
                    .tint(.white)
            }

            // Error message
            if let errorMessage = viewModel.errorMessage {
                VStack(spacing: 16) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 40))
                        .foregroundColor(.yellow)

                    Text("Error")
                        .font(.title2)
                        .foregroundColor(.white)

                    Text(errorMessage)
                        .font(.body)
                        .foregroundColor(.white.opacity(0.8))
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 20)

                    Button("Go Back") {
                        dismiss()
                    }
                    .buttonStyle(.borderedProminent)
                    .padding(.top, 8)
                }
                .padding(24)
                .background(Color.black.opacity(0.8))
                .cornerRadius(16)
            }

            // Dismiss button
            VStack {
                HStack {
                    Button {
                        viewModel.pause()
                        dismiss()
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 30))
                            .foregroundStyle(.white.opacity(0.8))
                            .shadow(radius: 4)
                    }
                    .padding(16)

                    Spacer()
                }

                Spacer()
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { notification in
            if let player = viewModel.player,
               notification.object as? AVPlayerItem === player.currentItem {
                viewModel.handleVideoEnd()
            }
        }
        .task {
            if let channel = currentChannel {
                await viewModel.loadChannel(channelId: channel.id)
            }
        }
        .onDisappear {
            viewModel.cleanup()
        }
    }
}

#Preview {
    ChannelPlayerView(
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
        startIndex: 0
    )
}
