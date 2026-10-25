import SwiftUI

/// A cable-box style guide: channels down the left, time across the top, and
/// every program drawn as wide as it is long.
///
/// Layout is two nested scroll views — vertical outermost, horizontal inside —
/// with the channel rail sitting outside the horizontal one so it stays put
/// while time scrolls. tvOS moves both to reveal whatever has focus, so there
/// is no manual scrolling to do.
struct ChannelGuideView: View {
    let channels: [Channel]
    let onSelectChannel: (Channel, Int) -> Void

    @StateObject private var viewModel = ChannelGuideViewModel()
    @FocusState private var focusedAiring: String?
    @State private var focused: (channel: Channel, program: GuideProgram)?

    /// The time being browsed: nil while looking at what is on now, or the
    /// start of the programme moved along to. Moving up or down keeps to it,
    /// so the next row is entered at the same point in the evening and not
    /// wherever happens to sit underneath.
    @State private var browseTime: Date?

    // Geometry. 30 minutes reads as 300pt, wide enough that a sitcom slot is
    // still legible at ten feet.
    private let pointsPerMinute: CGFloat = 10
    private let railWidth: CGFloat = 300
    private let rowHeight: CGFloat = 92
    private let axisHeight: CGFloat = 52
    private let slotMinutes = 30
    private let pillWidth: CGFloat = 104
    private let pillHeight: CGFloat = 36

    var body: some View {
        VStack(spacing: 0) {
            detailBanner

            if viewModel.isLoading && viewModel.entries.isEmpty {
                Spacer()
                ProgressView("Loading guide...")
                Spacer()
            } else if viewModel.entries.isEmpty {
                Spacer()
                ContentUnavailableView(
                    "No Channels",
                    systemImage: "tv.inset.filled",
                    description: Text("Create channels in the web interface to watch here")
                )
                Spacer()
            } else {
                grid
            }
        }
        .task {
            await viewModel.loadGuide()
            viewModel.startAutoRefresh()
        }
        .onDisappear { viewModel.stopAutoRefresh() }
    }

    // MARK: - Detail banner

    private var detailBanner: some View {
        HStack(alignment: .top, spacing: 28) {
            if let focused {
                GuideBackdropView(backdropPath: focused.program.backdropPath, fallback: focused.channel.icon)

                VStack(alignment: .leading, spacing: 6) {
                    Text(statusText(for: focused.program))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(focused.program.isAiring(at: viewModel.now) ? .orange : .secondary)

                    Text(focused.program.title)
                        .font(.title2.weight(.semibold))
                        .lineLimit(1)

                    if let showTitle = focused.program.showTitle, !showTitle.isEmpty {
                        Text(showTitle)
                            .font(.subheadline)
                            .foregroundStyle(.blue)
                            .lineLimit(1)
                    }

                    Text(metaText(for: focused.channel, program: focused.program))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .monospacedDigit()

                    if focused.program.isAiring(at: viewModel.now) {
                        ProgressView(value: focused.program.progress(at: viewModel.now))
                            .tint(.orange)
                            .frame(maxWidth: 420)
                            .padding(.top, 4)
                    }
                }
                Spacer(minLength: 0)
            } else {
                Text("Channel Guide")
                    .font(.title2.weight(.semibold))
                Spacer(minLength: 0)
            }

            // The wall clock the playhead is tracking, where a cable box
            // keeps it.
            VStack(alignment: .trailing, spacing: 4) {
                Text(Self.clockWithPeriod.string(from: viewModel.now))
                    .font(.title2.weight(.semibold))
                    .monospacedDigit()
                Text(Self.day.string(from: viewModel.now))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(height: 190, alignment: .top)
        .padding(.horizontal, 50)
        .padding(.top, 20)
    }

    private func statusText(for program: GuideProgram) -> String {
        let now = viewModel.now
        if program.isAiring(at: now) {
            return "ON NOW · \(program.remaining(at: now) / 60) MIN LEFT"
        }
        if program.endTime <= now { return "ALREADY AIRED" }
        let minutes = max(0, Int(program.startTime.timeIntervalSince(now) / 60))
        return minutes >= 60 ? "STARTS IN \(minutes / 60)H \(minutes % 60)M" : "STARTS IN \(minutes) MIN"
    }

    private func metaText(for channel: Channel, program: GuideProgram) -> String {
        "\(channel.displayNumber) \(channel.name)  ·  \(Self.clock.string(from: program.startTime)) – \(Self.clock.string(from: program.endTime))  ·  \(program.formattedDuration)"
    }

    // MARK: - Grid

    private var grid: some View {
        ScrollView(.vertical, showsIndicators: false) {
            HStack(alignment: .top, spacing: 0) {
                // Channel rail — outside the horizontal scroll, so it holds
                // position while time moves.
                VStack(spacing: 0) {
                    Color.clear.frame(width: railWidth, height: axisHeight)
                    ForEach(viewModel.entries) { entry in
                        channelCell(entry.channel)
                    }
                }

                ScrollView(.horizontal, showsIndicators: false) {
                    VStack(alignment: .leading, spacing: 0) {
                        timeAxis
                        ForEach(viewModel.entries) { entry in
                            lane(for: entry)
                        }
                    }
                    .overlay(alignment: .topLeading) { playhead }
                }
            }
            .padding(.leading, 50)
        }
    }

    private func channelCell(_ channel: Channel) -> some View {
        HStack(spacing: 14) {
            Text("\(channel.displayNumber)")
                .font(.system(.body, design: .monospaced))
                .foregroundStyle(.secondary)
                .frame(width: 58, alignment: .leading)

            Text(channel.icon)
                .font(.system(size: 30))

            Text(channel.name)
                .font(.headline)
                .lineLimit(1)

            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .frame(width: railWidth, height: rowHeight, alignment: .leading)
        .background(focused?.channel.id == channel.id ? Color.white.opacity(0.12) : Color.clear)
    }

    private var timeAxis: some View {
        HStack(spacing: 0) {
            ForEach(axisSlots, id: \.self) { slot in
                Text(Self.clock.string(from: slot))
                    .font(.system(.callout, design: .monospaced))
                    .foregroundStyle(.secondary)
                    .frame(width: CGFloat(slotMinutes) * pointsPerMinute, alignment: .leading)
                    .padding(.leading, 12)
            }
        }
        .frame(height: axisHeight, alignment: .leading)
    }

    private var axisSlots: [Date] {
        let count = viewModel.windowMinutes / slotMinutes
        guard count > 0 else { return [] }
        return (0..<count).map {
            viewModel.windowStart.addingTimeInterval(Double($0 * slotMinutes * 60))
        }
    }

    private func lane(for entry: ChannelGuideEntry) -> some View {
        HStack(spacing: 0) {
            if entry.programs.isEmpty {
                Text("No content scheduled")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 16)
                    .frame(width: laneWidth, height: rowHeight - 12, alignment: .leading)
                    .background(Color.gray.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
            } else {
                ForEach(entry.programs, id: \.airingId) { program in
                    programBlock(program, in: entry)
                }
            }
        }
        .frame(height: rowHeight, alignment: .leading)
        // Each row is entered as a whole, at the programme `landing` picks.
        // Left to itself the focus engine goes to whichever block is nearest
        // on screen, and since blocks are as wide as programmes are long that
        // is as likely to be next up as what is on.
        .focusSection()
        .defaultFocus($focusedAiring, landing(in: entry)?.airingId, priority: .userInitiated)
    }

    /// Where focus lands on entering a channel's row: what it is airing at
    /// the time being browsed, which is now unless the viewer has moved along
    /// the schedule.
    private func landing(in entry: ChannelGuideEntry) -> GuideProgram? {
        if let browseTime, let program = entry.program(airingAt: browseTime) {
            return program
        }
        return entry.program(airingAt: viewModel.now) ?? entry.programs.first
    }

    private var laneWidth: CGFloat {
        CGFloat(viewModel.windowMinutes) * pointsPerMinute
    }

    /// Programs are contiguous, so laying them out in an HStack at their clipped
    /// widths reproduces the time axis exactly — no absolute positioning needed.
    private func width(of program: GuideProgram) -> CGFloat {
        let from = max(program.startTime, viewModel.windowStart)
        let to = min(program.endTime, viewModel.windowEnd)
        let minutes = to.timeIntervalSince(from) / 60
        return max(0, CGFloat(minutes) * pointsPerMinute)
    }

    private func programBlock(_ program: GuideProgram, in entry: ChannelGuideEntry) -> some View {
        let blockWidth = width(of: program)
        let isLive = program.isAiring(at: viewModel.now)
        let isPast = program.endTime <= viewModel.now
        let isFocused = focusedAiring == program.airingId

        return Button {
            tune(to: entry)
        } label: {
            VStack(alignment: .leading, spacing: 3) {
                Text(program.showTitle.map { "\($0) · \(program.title)" } ?? program.title)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(1)
                Text(subtitle(for: program))
                    .font(.system(.caption2, design: .monospaced))
                    .foregroundStyle(isFocused ? Color.black.opacity(0.65) : Color.secondary)
                    .lineLimit(1)
            }
            .foregroundStyle(isFocused ? Color.black : Color.primary)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
            .padding(.horizontal, 14)
            .background(blockBackground(isLive: isLive, isFocused: isFocused))
        }
        .buttonStyle(.plain)
        .frame(width: max(blockWidth - 6, 0), height: rowHeight - 14)
        .opacity(isPast ? 0.45 : 1)
        .padding(.trailing, 6)
        .focused($focusedAiring, equals: program.airingId)
        .onChange(of: focusedAiring) { newValue in
            if newValue == program.airingId {
                // Only a move along a row changes the time being browsed.
                // A move between rows arrives at that time and must not
                // reset it to wherever the new programme happens to start,
                // or each row down would drift a little earlier.
                if focused == nil || focused?.channel.id == entry.channel.id {
                    browseTime = program.isAiring(at: viewModel.now) ? nil : program.startTime
                }
                focused = (entry.channel, program)
            }
        }
    }

    private func subtitle(for program: GuideProgram) -> String {
        "\(Self.clock.string(from: program.startTime)) – \(Self.clock.string(from: program.endTime))"
    }

    @ViewBuilder
    private func blockBackground(isLive: Bool, isFocused: Bool) -> some View {
        if isFocused {
            RoundedRectangle(cornerRadius: 10).fill(Color.white)
        } else if isLive {
            RoundedRectangle(cornerRadius: 10).fill(Color.orange.opacity(0.22))
        } else {
            RoundedRectangle(cornerRadius: 10).fill(Color.white.opacity(0.08))
        }
    }

    /// The line every lane is read against: the current time in a pill on the
    /// axis, the line dropping from it through every channel, and the time
    /// already gone tinted behind it.
    private var playhead: some View {
        let minutes = viewModel.now.timeIntervalSince(viewModel.windowStart) / 60
        let offset = min(laneWidth, max(0, CGFloat(minutes) * pointsPerMinute))
        let inWindow = viewModel.now >= viewModel.windowStart && viewModel.now <= viewModel.windowEnd
        let lanesHeight = CGFloat(viewModel.entries.count) * rowHeight

        return ZStack(alignment: .topLeading) {
            Rectangle()
                .fill(Color.orange.opacity(0.1))
                .frame(width: offset, height: lanesHeight)
                .offset(y: axisHeight)
            Rectangle()
                .fill(Color.orange)
                .frame(width: 3, height: lanesHeight + axisHeight - pillHeight)
                .offset(x: offset - 1.5, y: pillHeight)
            Text(Self.clock.string(from: viewModel.now))
                .font(.system(.caption, design: .monospaced).weight(.bold))
                .foregroundStyle(.black)
                .frame(width: pillWidth, height: pillHeight)
                .background(Color.orange, in: Capsule())
                .offset(x: max(0, offset - pillWidth / 2))
        }
        .opacity(inWindow ? 1 : 0)
        .allowsHitTesting(false)
    }

    // MARK: - Actions

    private func tune(to entry: ChannelGuideEntry) {
        guard let index = channels.firstIndex(where: { $0.id == entry.channel.id }) else { return }
        onSelectChannel(entry.channel, index)
    }

    private static let clock: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "h:mm"
        return f
    }()

    private static let clockWithPeriod: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "h:mm a"
        return f
    }()

    private static let day: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "EEEE, MMM d"
        return f
    }()
}

/// Backdrop art for the detail banner, in the spot a cable box puts its
/// picture window. Falls back to the channel emoji for items the scanner never
/// matched (extras carry no artwork at all).
struct GuideBackdropView: View {
    let backdropPath: String?
    let fallback: String

    var body: some View {
        Group {
            if let backdropPath, !backdropPath.isEmpty,
               let url = URL(string: "https://image.tmdb.org/t/p/w780\(backdropPath)") {
                AsyncImage(url: url) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    placeholder
                }
            } else {
                placeholder
            }
        }
        .frame(width: 302, height: 170)
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }

    private var placeholder: some View {
        ZStack {
            Color.gray.opacity(0.25)
            Text(fallback).font(.system(size: 56))
        }
    }
}
