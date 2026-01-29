import SwiftUI
import AVKit

struct ChannelPlayerView: View {
    let channels: [Channel]
    let startIndex: Int

    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = ChannelPlayerViewModel()

    @State private var currentIndex: Int
    @State private var showOverlay = true
    @State private var overlayTimer: Timer?
    @State private var showSwitchOverlay = false
    @State private var switchDirection: ChannelSwitchDirection = .up

    init(channels: [Channel], startIndex: Int) {
        self.channels = channels
        self.startIndex = startIndex
        self._currentIndex = State(initialValue: startIndex)
    }

    private var currentChannel: Channel? {
        guard currentIndex >= 0 && currentIndex < channels.count else { return nil }
        return channels[currentIndex]
    }

    var body: some View {
        ZStack {
            // Video Player
            if let player = viewModel.player {
                AVPlayerView(player: player)
                    .ignoresSafeArea()
            } else {
                Color.black.ignoresSafeArea()
            }

            // Loading indicator
            if viewModel.isBuffering || viewModel.isLoading {
                ProgressView()
                    .scaleEffect(2)
            }

            // Error message
            if let errorMessage = viewModel.errorMessage {
                VStack(spacing: 20) {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 60))
                        .foregroundColor(.yellow)

                    Text("Error")
                        .font(.title)
                        .foregroundColor(.white)

                    Text(errorMessage)
                        .font(.body)
                        .foregroundColor(.white.opacity(0.8))
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 40)

                    Button("Go Back") {
                        dismiss()
                    }
                    .padding(.top, 20)
                }
                .padding(40)
                .background(Color.black.opacity(0.8))
                .cornerRadius(20)
            }

            // Channel Info Overlay
            if showOverlay && !showSwitchOverlay {
                VStack {
                    // Top bar with channel info
                    HStack(alignment: .top) {
                        if let channel = viewModel.channel {
                            HStack(spacing: 16) {
                                Text(channel.icon)
                                    .font(.system(size: 40))

                                VStack(alignment: .leading, spacing: 4) {
                                    Text(channel.name)
                                        .font(.headline)
                                        .foregroundColor(.white)

                                    if let nowPlaying = viewModel.nowPlaying {
                                        Text("Now: \(nowPlaying.title)")
                                            .font(.subheadline)
                                            .foregroundColor(.white.opacity(0.8))
                                    }
                                }
                            }
                            .padding()
                            .background(Color.black.opacity(0.7))
                            .cornerRadius(12)
                        }

                        Spacer()
                    }
                    .padding()

                    Spacer()

                    // Up Next panel at the bottom right
                    if !viewModel.upNext.isEmpty {
                        HStack {
                            Spacer()

                            VStack(alignment: .leading, spacing: 12) {
                                Text("Up Next")
                                    .font(.headline)
                                    .foregroundColor(.white)

                                ForEach(viewModel.upNext.prefix(3)) { item in
                                    UpNextItemRow(item: item)
                                }
                            }
                            .padding()
                            .background(Color.black.opacity(0.7))
                            .cornerRadius(12)
                        }
                        .padding()
                    }
                }
            }

            // Channel Switch Overlay
            if showSwitchOverlay, let channel = currentChannel {
                VStack {
                    Spacer()
                    ChannelSwitchOverlay(
                        channel: channel,
                        nowPlaying: viewModel.nowPlaying,
                        direction: switchDirection
                    )
                    Spacer()
                }
                .transition(.opacity)
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { notification in
            // Check if this notification is for our current player
            if let player = viewModel.player,
               notification.object as? AVPlayerItem === player.currentItem {
                viewModel.handleVideoEnd()
            }
        }
        .task {
            if let channel = currentChannel {
                await viewModel.loadChannel(channelId: channel.id)
            }
            startOverlayTimer()
        }
        .onDisappear {
            cancelOverlayTimer()
            viewModel.cleanup()
        }
        .onExitCommand {
            viewModel.pause()
            dismiss()
        }
        .onPlayPauseCommand {
            // Toggle overlay visibility on play/pause button
            showOverlay.toggle()
            if showOverlay {
                startOverlayTimer()
            }
        }
        .onMoveCommand { direction in
            switch direction {
            case .up:
                switchToNextChannel()
            case .down:
                switchToPreviousChannel()
            default:
                break
            }
        }
    }

    private func switchToNextChannel() {
        guard channels.count > 1 else { return }

        let newIndex = (currentIndex + 1) % channels.count
        switchToChannel(at: newIndex, direction: .up)
    }

    private func switchToPreviousChannel() {
        guard channels.count > 1 else { return }

        let newIndex = currentIndex == 0 ? channels.count - 1 : currentIndex - 1
        switchToChannel(at: newIndex, direction: .down)
    }

    private func switchToChannel(at index: Int, direction: ChannelSwitchDirection) {
        currentIndex = index
        switchDirection = direction

        withAnimation {
            showSwitchOverlay = true
        }

        Task {
            await viewModel.switchToChannel(channelId: channels[index].id)
        }

        // Auto-dismiss switch overlay after 2 seconds
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            withAnimation {
                showSwitchOverlay = false
            }
        }
    }

    private func startOverlayTimer() {
        cancelOverlayTimer()
        overlayTimer = Timer.scheduledTimer(withTimeInterval: 5, repeats: false) { _ in
            withAnimation {
                showOverlay = false
            }
        }
    }

    private func cancelOverlayTimer() {
        overlayTimer?.invalidate()
        overlayTimer = nil
    }
}

struct UpNextItemRow: View {
    let item: ChannelScheduleItem

    var body: some View {
        HStack(spacing: 12) {
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
                Text(item.title)
                    .font(.subheadline)
                    .foregroundColor(.white)
                    .lineLimit(2)

                Text(item.formattedDuration)
                    .font(.caption)
                    .foregroundColor(.white.opacity(0.7))
            }
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
                icon: "🎬",
                createdAt: "2024-01-01",
                updatedAt: "2024-01-01",
                totalDuration: 7200,
                itemCount: 5
            )
        ],
        startIndex: 0
    )
}
