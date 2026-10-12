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

    // The now-playing card behaves like player chrome: it appears when the
    // program or channel changes, is brought back by a tap, and otherwise
    // fades out after 5s -- matching the tvOS info overlay -- so it doesn't
    // sit on top of the video for the whole session.
    @State private var infoVisible = true
    @State private var hideInfoTask: Task<Void, Never>? = nil

    private let infoAutoHideDelay: UInt64 = 5_000_000_000

    @State private var showSleepOptions = false

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
                AVPlayerView(
                    player: viewModel.player,
                    onVerticalDrag: { trackSwipe($0) },
                    onVerticalDragEnded: { commitSwipe() }
                )
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

                    // The armed countdown stays up even while the rest of the
                    // chrome is hidden, so you can always see time running out.
                    if let countdown = viewModel.sleepCountdownText {
                        Label(countdown, systemImage: "moon.fill")
                            .font(.caption.monospacedDigit().weight(.semibold))
                            .foregroundStyle(.orange)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(.black.opacity(0.6), in: Capsule())
                            .padding(.trailing, 8)
                    }

                    if infoVisible {
                        Button {
                            showSleepOptions = true
                        } label: {
                            Image(systemName: viewModel.isSleepTimerArmed ? "moon.fill" : "moon")
                                .font(.system(size: 24))
                                .foregroundStyle(.white.opacity(0.8))
                                .shadow(radius: 4)
                        }
                        .padding(16)
                    }
                }

                Spacer()

                if let nowPlaying = viewModel.nowPlaying, infoVisible {
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
            .animation(.easeInOut(duration: 0.3), value: infoVisible)
        }
        .simultaneousGesture(
            TapGesture().onEnded { showInfoBriefly() }
        )
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { notification in
            if let player = viewModel.player,
               notification.object as? AVPlayerItem === player.currentItem {
                viewModel.handleVideoEnd()
            }
        }
        .onChange(of: viewModel.nowPlaying?.mediaId) { _ in
            showInfoBriefly()
        }
        .onChange(of: viewModel.sleepFireCount) { _ in
            // Timer elapsed: playback is already paused, so leave the player.
            dismiss()
        }
        .confirmationDialog("Sleep Timer", isPresented: $showSleepOptions, titleVisibility: .visible) {
            ForEach([15, 30, 45, 60], id: \.self) { minutes in
                Button("\(minutes) minutes") { viewModel.armSleepTimer(minutes: minutes) }
            }
            if viewModel.secondsUntilEndOfProgram != nil {
                Button("End of program") { viewModel.armSleepTimerForEndOfProgram() }
            }
            if viewModel.isSleepTimerArmed {
                Button("Turn off", role: .destructive) { viewModel.cancelSleepTimer() }
            }
            Button("Cancel", role: .cancel) { }
        } message: {
            Text(viewModel.isSleepTimerArmed
                 ? "Playback stops in \(viewModel.sleepCountdownText ?? "")."
                 : "Playback will stop and return you to the guide.")
        }
        .task(id: currentChannel?.id) {
            showInfoBriefly()
            if let channel = currentChannel {
                await viewModel.loadChannel(channelId: channel.id)
            }
        }
        .onDisappear {
            hideInfoTask?.cancel()
            hideInfoTask = nil
            viewModel.cleanup()
        }
    }

    /// Shows the now-playing card, then fades it out after a few seconds.
    /// Restarts the countdown if it is already on screen.
    private func showInfoBriefly() {
        hideInfoTask?.cancel()
        infoVisible = true
        hideInfoTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: infoAutoHideDelay)
            guard !Task.isCancelled else { return }
            infoVisible = false
        }
    }

    /// Follows a vertical drag on the player: up for the next channel, down
    /// for the previous one. The drag comes from `AVPlayerView`, which can
    /// hear it even while AVKit's controls are showing.
    private func trackSwipe(_ translation: CGFloat) {
        guard channels.count > 1, !viewModel.isLoading else { return }
        dragOffset = translation
        if dragOffset < -30, currentIndex + 1 < channels.count {
            pendingTarget = channels[currentIndex + 1]
        } else if dragOffset > 30, currentIndex > 0 {
            pendingTarget = channels[currentIndex - 1]
        } else {
            pendingTarget = nil
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
                channelNumber: 101,
                createdAt: "2024-01-01",
                updatedAt: "2024-01-01",
                totalDuration: 7200,
                itemCount: 5
            )
        ],
        startIndex: 0
    )
}
