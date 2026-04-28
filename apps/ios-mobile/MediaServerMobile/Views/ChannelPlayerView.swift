import SwiftUI
import AVKit
import UIKit

struct ChannelPlayerView: View {
    let channels: [Channel]
    let startIndex: Int

    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = ChannelPlayerViewModel()
    @State private var currentIndex: Int
    @State private var dragOffset: CGFloat = 0
    @State private var pendingTarget: Channel? = nil

    init(channels: [Channel], startIndex: Int) {
        self.channels = channels
        self.startIndex = startIndex
        _currentIndex = State(initialValue: startIndex)
    }

    private var currentChannel: Channel? {
        channels.indices.contains(currentIndex) ? channels[currentIndex] : nil
    }

    var body: some View {
        ZStack {
            // Video Player — uses the same AVPlayerViewController wrapper
            // as VOD playback so we get PiP, AirPlay, and Apple's controls.
            if viewModel.player != nil {
                AVPlayerView(player: viewModel.player)
                    .ignoresSafeArea()
                    .opacity(viewModel.isLoading ? 0.4 : 1.0)
                    .animation(.easeInOut(duration: 0.2), value: viewModel.isLoading)
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

            // Channel-switch preview during a vertical drag.
            if let target = pendingTarget {
                VStack(spacing: 12) {
                    Image(systemName: dragOffset < 0 ? "chevron.up" : "chevron.down")
                        .font(.system(size: 36, weight: .bold))
                    if !target.icon.isEmpty {
                        Text(target.icon).font(.system(size: 56))
                    }
                    Text(target.name).font(.title2.weight(.semibold))
                }
                .foregroundStyle(.white)
                .padding(24)
                .background(.black.opacity(0.55), in: RoundedRectangle(cornerRadius: 18))
                .opacity(min(1.0, Double(abs(dragOffset)) / 80.0))
                .transition(.opacity)
                .allowsHitTesting(false)
            }

            // Top close + bottom now-playing card overlay
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

                if let nowPlaying = viewModel.nowPlaying {
                    NowPlayingCard(
                        channelName: viewModel.channel?.name,
                        nowPlayingTitle: nowPlaying.title,
                        upNextTitle: viewModel.upNext.first?.title
                    )
                    .padding(.horizontal, 16)
                    .padding(.bottom, 24)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                }
            }
            .animation(.easeInOut(duration: 0.25), value: viewModel.nowPlaying?.mediaId)
        }
        .simultaneousGesture(
            DragGesture(minimumDistance: 30)
                .onChanged { value in
                    guard channels.count > 1, !viewModel.isLoading else { return }
                    dragOffset = value.translation.height
                    if dragOffset < -30, currentIndex + 1 < channels.count {
                        pendingTarget = channels[currentIndex + 1]
                    } else if dragOffset > 30, currentIndex > 0 {
                        pendingTarget = channels[currentIndex - 1]
                    } else {
                        pendingTarget = nil
                    }
                }
                .onEnded { _ in commitSwipe() }
        )
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { notification in
            if let player = viewModel.player,
               notification.object as? AVPlayerItem === player.currentItem {
                viewModel.handleVideoEnd()
            }
        }
        .task(id: currentChannel?.id) {
            if let channel = currentChannel {
                await viewModel.loadChannel(channelId: channel.id)
            }
        }
        .onDisappear {
            viewModel.cleanup()
        }
    }

    private func commitSwipe() {
        defer {
            withAnimation(.spring(response: 0.35, dampingFraction: 0.75)) {
                dragOffset = 0
            }
        }
        guard channels.count > 1, !viewModel.isLoading else { return }

        if dragOffset <= -80, currentIndex + 1 < channels.count {
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
            currentIndex += 1
        } else if dragOffset >= 80, currentIndex > 0 {
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
            currentIndex -= 1
        }

        // Keep the preview visible briefly as confirmation, then clear.
        let target = pendingTarget
        Task { @MainActor in
            try? await Task.sleep(nanoseconds: 800_000_000)
            if pendingTarget == target {
                withAnimation(.easeOut(duration: 0.25)) {
                    pendingTarget = nil
                }
            }
        }
    }
}

private struct NowPlayingCard: View {
    let channelName: String?
    let nowPlayingTitle: String
    let upNextTitle: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let channelName {
                Text(channelName.uppercased())
                    .font(.caption2.weight(.bold))
                    .foregroundColor(.white.opacity(0.7))
                    .lineLimit(1)
            }
            Text(nowPlayingTitle)
                .font(.headline)
                .foregroundColor(.white)
                .lineLimit(1)
            if let upNextTitle {
                Text("Up next: \(upNextTitle)")
                    .font(.caption)
                    .foregroundColor(.white.opacity(0.75))
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(.black.opacity(0.45), in: RoundedRectangle(cornerRadius: 14))
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .stroke(.white.opacity(0.08), lineWidth: 1)
        )
        .shadow(radius: 8)
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
