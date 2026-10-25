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
    @State private var showSleepOptions = false
    /// Counts channel changes, so only the latest one's timer takes the
    /// banner down.
    @State private var bannerToken = 0
    @State private var showSpinner = false

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
            Color.black.ignoresSafeArea()

            // The picture and nothing else. Channels are live television:
            // there is no scrubbing, so AVKit's transport bar and its
            // chapters panel have no business appearing on a channel change.
            if let player = viewModel.player {
                ChannelVideoSurface(player: player)
                    .ignoresSafeArea()
            }

            // Only once a channel is taking long enough to wonder about.
            if showSpinner {
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
                                            .accessibilityIdentifier("nowPlaying")
                                    }

                                    Text("Hold the center button for Live and the sleep timer")
                                        .font(.caption)
                                        .foregroundColor(.white.opacity(0.5))
                                        .padding(.top, 2)
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
                        .padding(.horizontal)
                    }

                    ChannelTransportBar(viewModel: viewModel)
                        .padding(.horizontal, 60)
                        .padding(.bottom, 50)
                }
            }

            // Armed sleep timer stays visible regardless of the info overlay.
            // So is how far behind live, once the bar has gone.
            VStack(alignment: .trailing, spacing: 12) {
                if let countdown = viewModel.sleepCountdownText {
                    Label(countdown, systemImage: "moon.fill")
                        .font(.callout.monospacedDigit().weight(.semibold))
                        .foregroundColor(.orange)
                        .padding(.horizontal, 18)
                        .padding(.vertical, 10)
                        .background(Color.black.opacity(0.7), in: Capsule())
                }
                if !showOverlay {
                    BehindLiveTag(viewModel: viewModel)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topTrailing)
            .padding(40)

            // Channel banner: up the instant the channel changes.
            if showSwitchOverlay, let channel = currentChannel {
                VStack {
                    Spacer()
                    HStack {
                        ChannelBanner(channel: channel, nowPlaying: viewModel.nowPlaying)
                        Spacer()
                    }
                }
                .padding(60)
                .transition(.opacity)
            }
        }
        // With no AVKit controller inside, something has to hold focus for
        // the remote's presses to arrive.
        .focusable()
        .task(id: viewModel.isLoading || viewModel.isBuffering) {
            showSpinner = false
            guard viewModel.isLoading || viewModel.isBuffering else { return }
            try? await Task.sleep(nanoseconds: 1_200_000_000)
            if !Task.isCancelled { showSpinner = true }
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
                viewModel.tune(to: channel, neighbours: neighbours(of: currentIndex), immediately: true)
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
            viewModel.togglePause()
            revealControls()
        }
        // Holding the center button: Live and the sleep timer. A button in
        // the auto-hiding bar would fight the focus engine for the remote.
        .onLongPressGesture(minimumDuration: 0.6) {
            showSleepOptions = true
        }
        .onTapGesture {
            if showOverlay && !viewModel.timeshift.isPaused {
                cancelOverlayTimer()
                withAnimation { showOverlay = false }
            } else {
                revealControls()
            }
        }
        .onMoveCommand { direction in
            switch direction {
            case .up:
                switchToNextChannel()
            case .down:
                switchToPreviousChannel()
            case .left:
                viewModel.skip(forward: false)
                revealControls()
            case .right:
                viewModel.skip(forward: true)
                revealControls()
            default:
                break
            }
        }
        .onChange(of: viewModel.timeshift.isPaused) { paused in
            // The bar stays up for as long as the channel is paused.
            if !paused { startOverlayTimer() }
        }
        .onChange(of: viewModel.sleepFireCount) { _ in
            // Playback is already paused; leave the player.
            dismiss()
        }
        .sheet(isPresented: $showSleepOptions) {
            SleepTimerSheet(viewModel: viewModel)
        }
    }

    private func switchToNextChannel() {
        guard channels.count > 1 else { return }
        switchToChannel(at: (currentIndex + 1) % channels.count)
    }

    private func switchToPreviousChannel() {
        guard channels.count > 1 else { return }
        switchToChannel(at: currentIndex == 0 ? channels.count - 1 : currentIndex - 1)
    }

    /// The channels one press away in either direction.
    private func neighbours(of index: Int) -> [Channel] {
        guard channels.count > 1 else { return [] }
        let next = (index + 1) % channels.count
        let previous = (index - 1 + channels.count) % channels.count
        return next == previous ? [channels[next]] : [channels[next], channels[previous]]
    }

    private func switchToChannel(at index: Int) {
        currentIndex = index

        // Banner and cut first, with no animation to wait through; the
        // picture follows as soon as it can.
        showOverlay = false
        showSwitchOverlay = true
        viewModel.tune(to: channels[index], neighbours: neighbours(of: index))

        bannerToken += 1
        let token = bannerToken
        Task {
            try? await Task.sleep(nanoseconds: 3_500_000_000)
            guard token == bannerToken else { return }
            withAnimation { showSwitchOverlay = false }
        }
    }

    /// The info and the bar, up for five seconds, or for as long as the
    /// channel is paused.
    private func revealControls() {
        showSwitchOverlay = false
        showOverlay = true
        if viewModel.timeshift.isPaused {
            cancelOverlayTimer()
        } else {
            startOverlayTimer()
        }
    }

    private func startOverlayTimer() {
        cancelOverlayTimer()
        overlayTimer = Timer.scheduledTimer(withTimeInterval: 5, repeats: false) { _ in
            Task { @MainActor in
                guard !viewModel.timeshift.isPaused else { return }
                withAnimation { showOverlay = false }
            }
        }
    }

    private func cancelOverlayTimer() {
        overlayTimer?.invalidate()
        overlayTimer = nil
    }
}

/// The strip a television shows on a channel change: number, name, and what
/// is on, at the bottom of the screen where it doesn't cover the picture.
private struct ChannelBanner: View {
    let channel: Channel
    let nowPlaying: ChannelScheduleItem?

    var body: some View {
        HStack(alignment: .center, spacing: 28) {
            Text("\(channel.displayNumber)")
                .font(.system(size: 64, weight: .bold, design: .rounded))
                .monospacedDigit()

            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 12) {
                    Text(channel.icon)
                    Text(channel.name)
                        .fontWeight(.semibold)
                }
                .font(.title3)

                Text(programme)
                    .font(.headline)
                    .foregroundColor(.white.opacity(0.75))
                    .lineLimit(1)
            }
        }
        .foregroundColor(.white)
        .padding(.horizontal, 36)
        .padding(.vertical, 24)
        .background(Color.black.opacity(0.78), in: RoundedRectangle(cornerRadius: 20))
    }

    private var programme: String {
        guard let nowPlaying else { return " " }
        if let showTitle = nowPlaying.showTitle, !showTitle.isEmpty {
            return "\(showTitle) · \(nowPlaying.title)"
        }
        return nowPlaying.title
    }
}

/// A bare video layer: the player's picture with none of AVKit's interface.
struct ChannelVideoSurface: UIViewRepresentable {
    let player: AVPlayer

    func makeUIView(context: Context) -> PlayerLayerView {
        let view = PlayerLayerView()
        view.backgroundColor = .black
        view.playerLayer.videoGravity = .resizeAspect
        view.playerLayer.player = player
        return view
    }

    func updateUIView(_ view: PlayerLayerView, context: Context) {
        if view.playerLayer.player !== player {
            view.playerLayer.player = player
        }
    }
}

final class PlayerLayerView: UIView {
    override class var layerClass: AnyClass { AVPlayerLayer.self }
    var playerLayer: AVPlayerLayer { layer as! AVPlayerLayer }
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


/// Sleep timer options for the Apple TV player.
struct SleepTimerSheet: View {
    @ObservedObject var viewModel: ChannelPlayerViewModel
    @Environment(\.dismiss) private var dismiss

    private let presets = [15, 30, 45, 60]

    var body: some View {
        VStack(spacing: 28) {
            if !viewModel.timeshift.isLive(at: Date()) {
                Button {
                    viewModel.jumpToLive()
                    dismiss()
                } label: {
                    optionLabel("Jump to Live",
                                detail: "\(ChannelTimeshift.clock(viewModel.timeshift.behind(at: Date()))) behind")
                }
                .frame(maxWidth: 620)
                .accessibilityIdentifier("jumpToLive")
            }

            VStack(spacing: 6) {
                Text("Sleep Timer")
                    .font(.title2.weight(.semibold))
                Text(viewModel.isSleepTimerArmed
                     ? "Playback stops in \(viewModel.sleepCountdownText ?? "")."
                     : "Playback will stop and return you to the guide.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            VStack(spacing: 14) {
                ForEach(presets, id: \.self) { minutes in
                    Button {
                        viewModel.armSleepTimer(minutes: minutes)
                        dismiss()
                    } label: {
                        optionLabel("\(minutes) minutes", detail: stopTime(afterMinutes: minutes))
                    }
                }

                if let seconds = viewModel.secondsUntilEndOfProgram {
                    Button {
                        viewModel.armSleepTimerForEndOfProgram()
                        dismiss()
                    } label: {
                        optionLabel("End of program",
                                    detail: stopTime(afterMinutes: max(1, seconds / 60)))
                    }
                }

                if viewModel.isSleepTimerArmed {
                    Button {
                        viewModel.cancelSleepTimer()
                        dismiss()
                    } label: {
                        optionLabel("Turn off", detail: "")
                    }
                }
            }
            .frame(maxWidth: 620)
        }
        .padding(60)
    }

    private func optionLabel(_ title: String, detail: String) -> some View {
        HStack {
            Text(title)
            Spacer()
            Text(detail)
                .font(.callout.monospacedDigit())
                .foregroundColor(.secondary)
        }
        .padding(.horizontal, 8)
    }

    /// Shows when playback would actually stop, so the choice is concrete.
    private func stopTime(afterMinutes minutes: Int) -> String {
        let date = Date().addingTimeInterval(Double(minutes) * 60)
        let f = DateFormatter()
        f.dateFormat = "h:mm a"
        return f.string(from: date)
    }
}

/// The scrub line along the bottom, drawn like the movie player's: how far
/// into the programme, how far the channel itself has got, and whether this
/// is live or how far behind it.
struct ChannelTransportBar: View {
    @ObservedObject var viewModel: ChannelPlayerViewModel

    var body: some View {
        TimelineView(.periodic(from: .now, by: 1)) { context in
            if let item = viewModel.nowPlaying, item.duration > 0 {
                content(item: item, now: context.date)
            }
        }
    }

    private func content(item: ChannelScheduleItem, now: Date) -> some View {
        let duration = Double(item.duration)
        let position = viewModel.pendingSkipTarget ?? viewModel.position
        let behind = viewModel.timeshift.behind(at: now)
        // The channel's own place in this programme, as far as skipping
        // forward can go.
        let live = min(duration, viewModel.position + behind)
        let isLive = viewModel.timeshift.isLive(at: now)

        return VStack(spacing: 14) {
            HStack {
                Image(systemName: viewModel.timeshift.isPaused ? "pause.fill" : "play.fill")
                    .font(.title3)
                Spacer()
                if isLive {
                    Text("LIVE")
                        .font(.callout.weight(.heavy))
                        .padding(.horizontal, 14)
                        .padding(.vertical, 6)
                        .background(Color.red, in: Capsule())
                        .accessibilityIdentifier("liveBadge")
                } else {
                    Text("\(ChannelTimeshift.clock(behind)) behind live")
                        .font(.callout.weight(.semibold).monospacedDigit())
                        .padding(.horizontal, 14)
                        .padding(.vertical, 6)
                        .background(Color.white.opacity(0.22), in: Capsule())
                        .accessibilityIdentifier("behindBadge")
                }
            }

            GeometryReader { geo in
                let width = geo.size.width
                let x = width * CGFloat(min(1, max(0, position / duration)))
                let liveX = width * CGFloat(min(1, max(0, live / duration)))
                ZStack(alignment: .leading) {
                    Capsule().fill(Color.white.opacity(0.25))
                    Capsule().fill(Color.white.opacity(0.45)).frame(width: liveX)
                    Capsule().fill(Color.white).frame(width: x)
                    if !isLive {
                        Rectangle().fill(Color.red).frame(width: 4, height: 18).offset(x: liveX - 2)
                    }
                    Circle().fill(Color.white).frame(width: 22, height: 22).offset(x: x - 11)
                }
                .frame(height: 10)
                .frame(maxHeight: .infinity)
            }
            .frame(height: 22)

            HStack {
                Text(ChannelTimeshift.clock(position))
                    .accessibilityIdentifier("elapsedTime")
                Spacer()
                Text("◀ 10s   ▶ 30s")
                    .foregroundColor(.white.opacity(0.5))
                Spacer()
                Text("-\(ChannelTimeshift.clock(duration - position))")
            }
            .font(.callout.monospacedDigit())
        }
        .foregroundColor(.white)
        .padding(.horizontal, 36)
        .padding(.vertical, 24)
        .background(Color.black.opacity(0.7), in: RoundedRectangle(cornerRadius: 20))
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("transportBar")
    }
}

/// How far behind live, in the corner, while the bar is down.
struct BehindLiveTag: View {
    @ObservedObject var viewModel: ChannelPlayerViewModel

    var body: some View {
        TimelineView(.periodic(from: .now, by: 1)) { context in
            if viewModel.nowPlaying != nil, !viewModel.timeshift.isLive(at: context.date) {
                let text = "\(ChannelTimeshift.clock(viewModel.timeshift.behind(at: context.date))) behind live"
                Label(text,
                      systemImage: viewModel.timeshift.isPaused ? "pause.fill" : "clock.arrow.circlepath")
                    .font(.callout.monospacedDigit().weight(.semibold))
                    .foregroundColor(.white)
                    .padding(.horizontal, 18)
                    .padding(.vertical, 10)
                    .background(Color.black.opacity(0.7), in: Capsule())
                    .accessibilityElement(children: .ignore)
                    .accessibilityLabel(text)
                    .accessibilityIdentifier("behindTag")
            }
        }
    }
}
