import SwiftUI

/// The guide on a phone: a cable-style grid with channels down the left, time
/// across the top, every program drawn as wide as it is long, and one playhead
/// crossing all of them at the current time.
///
/// Upright, a card above the grid describes the selected program. On its side
/// the phone has the width for the classic cable arrangement instead: artwork
/// and details in a strip along the top, a longer stretch of schedule below.
struct ChannelGuideView: View {
    let channels: [Channel]
    let onSelectChannel: (Channel, Int) -> Void

    @StateObject private var viewModel = ChannelGuideViewModel()
    @Environment(\.verticalSizeClass) private var verticalSizeClass
    @State private var selection: GuideSelection?
    @State private var scrollOffset: CGFloat = 0
    @State private var jumpToNow = 0

    private var isLandscape: Bool { verticalSizeClass == .compact }
    private var metrics: GuideMetrics { isLandscape ? .landscape : .portrait }

    var body: some View {
        Group {
            if viewModel.isLoading && viewModel.entries.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
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
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                VStack(spacing: 0) {
                    if let selected {
                        if isLandscape {
                            landscapeDetail(selected.entry, selected.program)
                        } else {
                            portraitDetail(selected.entry, selected.program)
                        }
                    }
                    grid
                }
            }
        }
        .task {
            await viewModel.loadGuide()
            viewModel.startAutoRefresh()
        }
        .onDisappear { viewModel.stopAutoRefresh() }
    }

    // MARK: - Selection

    /// The program the detail area describes. Falls back to whatever the first
    /// channel is airing, both before anything is tapped and once a selected
    /// program has rolled off the front of the window.
    private var selected: (entry: ChannelGuideEntry, program: GuideProgram)? {
        if let selection,
           let entry = viewModel.entries.first(where: { $0.id == selection.channelId }),
           let program = entry.programs.first(where: { $0.airingId == selection.airingId }) {
            return (entry, program)
        }
        for entry in viewModel.entries {
            if let program = entry.program(airingAt: viewModel.now) {
                return (entry, program)
            }
        }
        return nil
    }

    private func isSelected(_ program: GuideProgram, in entry: ChannelGuideEntry) -> Bool {
        guard let selected else { return false }
        return selected.entry.id == entry.id && selected.program.airingId == program.airingId
    }

    /// First tap describes a program, a second tap on it tunes in.
    private func select(_ program: GuideProgram, in entry: ChannelGuideEntry) {
        if isSelected(program, in: entry) {
            tune(to: entry)
        } else {
            selection = GuideSelection(channelId: entry.id, airingId: program.airingId)
        }
    }

    // MARK: - Detail

    private func portraitDetail(_ entry: ChannelGuideEntry, _ program: GuideProgram) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top, spacing: 12) {
                GuideBackdropView(backdropPath: program.backdropPath, fallback: entry.channel.icon)
                    .frame(width: 112, height: 63)

                VStack(alignment: .leading, spacing: 3) {
                    statusLabel(for: program)
                    Text(displayTitle(of: program))
                        .font(.headline)
                        .lineLimit(1)
                    Text(metaText(for: program))
                        .font(.caption.monospacedDigit())
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }
                Spacer(minLength: 0)
            }

            if program.isAiring(at: viewModel.now) {
                ProgressView(value: program.progress(at: viewModel.now))
                    .tint(.orange)
            }

            Button { tune(to: entry) } label: {
                Text("Watch \(entry.channel.displayNumber) \(entry.channel.name)")
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(1)
                    .foregroundColor(.black)
                    .frame(maxWidth: .infinity)
                    .frame(height: 44)
                    .background(Color.orange, in: RoundedRectangle(cornerRadius: 10))
            }
            .buttonStyle(.plain)
        }
        .padding(12)
        .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 14))
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 10)
    }

    private func landscapeDetail(_ entry: ChannelGuideEntry, _ program: GuideProgram) -> some View {
        HStack(alignment: .center, spacing: 16) {
            GuideBackdropView(backdropPath: program.backdropPath, fallback: entry.channel.icon)
                .frame(width: 142, height: 80)

            VStack(alignment: .leading, spacing: 3) {
                statusLabel(for: program)
                Text(displayTitle(of: program))
                    .font(.title3.weight(.bold))
                    .lineLimit(1)
                Text("\(entry.channel.displayNumber) \(entry.channel.name) · \(metaText(for: program))")
                    .font(.caption.monospacedDigit())
                    .foregroundColor(.secondary)
                    .lineLimit(1)
                if program.isAiring(at: viewModel.now) {
                    ProgressView(value: program.progress(at: viewModel.now))
                        .tint(.orange)
                        .frame(maxWidth: 260)
                }
            }

            Spacer(minLength: 0)

            VStack(alignment: .trailing, spacing: 6) {
                Text(Self.clockWithPeriod.string(from: viewModel.now))
                    .font(.headline.monospacedDigit())
                Button { tune(to: entry) } label: {
                    Text("Watch \(entry.channel.displayNumber)")
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(.black)
                        .padding(.horizontal, 18)
                        .frame(height: 44)
                        .background(Color.orange, in: RoundedRectangle(cornerRadius: 10))
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }

    private func statusLabel(for program: GuideProgram) -> some View {
        Text(statusText(for: program))
            .font(.caption2.weight(.bold))
            .foregroundColor(program.isAiring(at: viewModel.now) ? .orange : .secondary)
            .lineLimit(1)
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

    private func metaText(for program: GuideProgram) -> String {
        var parts = [
            "\(Self.clock.string(from: program.startTime)) – \(Self.clockWithPeriod.string(from: program.endTime))",
            program.formattedDuration
        ]
        // The show name is already the headline for episodes, so the episode
        // title is what's left to say.
        if let showTitle = program.showTitle, !showTitle.isEmpty {
            parts.append(program.title)
        }
        return parts.joined(separator: " · ")
    }

    private func displayTitle(of program: GuideProgram) -> String {
        if let showTitle = program.showTitle, !showTitle.isEmpty { return showTitle }
        return program.title
    }

    // MARK: - Grid

    /// Vertical scroll outermost, horizontal inside, with the channel rail
    /// outside the horizontal one so it holds position while time moves. The
    /// time axis lives inside the horizontal scroll so it tracks time, and is
    /// pushed down by the vertical scroll distance so it also stays in view.
    private var grid: some View {
        ScrollView(.vertical, showsIndicators: false) {
            HStack(alignment: .top, spacing: 0) {
                VStack(spacing: 0) {
                    nowButton
                        .offset(y: scrollOffset)
                        .zIndex(1)
                    ForEach(viewModel.entries) { entry in
                        channelCell(entry.channel)
                    }
                }

                ScrollViewReader { proxy in
                    ScrollView(.horizontal, showsIndicators: false) {
                        VStack(alignment: .leading, spacing: 0) {
                            timeAxis
                                .offset(y: scrollOffset)
                                .zIndex(1)
                            VStack(alignment: .leading, spacing: 0) {
                                ForEach(viewModel.entries) { entry in
                                    lane(for: entry)
                                }
                            }
                            .overlay(alignment: .topLeading) { playhead }
                        }
                        .background(alignment: .topLeading) { nowAnchor }
                    }
                    .onChange(of: jumpToNow) { _ in
                        withAnimation {
                            proxy.scrollTo(Self.nowAnchorId, anchor: UnitPoint(x: 0.2, y: 0))
                        }
                    }
                }
            }
            .background(
                GeometryReader { geo in
                    Color.clear.preference(
                        key: GuideScrollOffsetKey.self,
                        value: -geo.frame(in: .named(Self.scrollSpace)).minY
                    )
                }
            )
        }
        .coordinateSpace(name: Self.scrollSpace)
        .onPreferenceChange(GuideScrollOffsetKey.self) { scrollOffset = max(0, $0) }
        .refreshable { await viewModel.loadGuide() }
    }

    private var nowButton: some View {
        Button { jumpToNow += 1 } label: {
            Text("Now")
                .font(.caption.weight(.semibold))
                .foregroundColor(.orange)
                .frame(width: metrics.railWidth, height: metrics.axisHeight)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .background(Color(.systemBackground))
        .accessibilityLabel("Jump to now")
    }

    private func channelCell(_ channel: Channel) -> some View {
        VStack(alignment: .leading, spacing: 1) {
            Text("\(channel.displayNumber)")
                .font(.subheadline.weight(.bold).monospacedDigit())
            Text(channel.name)
                .font(.caption2)
                .foregroundColor(.secondary)
                .lineLimit(isLandscape ? 1 : 2)
        }
        .padding(.leading, 12)
        .padding(.trailing, 6)
        .frame(width: metrics.railWidth, height: metrics.rowHeight, alignment: .leading)
    }

    private var timeAxis: some View {
        HStack(spacing: 0) {
            ForEach(axisSlots, id: \.self) { slot in
                HStack(spacing: 6) {
                    Rectangle()
                        .fill(Color(.separator))
                        .frame(width: 1)
                    Text(Self.clock.string(from: slot))
                        .font(.caption2.weight(.semibold).monospacedDigit())
                        .foregroundColor(.secondary)
                }
                .frame(width: CGFloat(Self.slotMinutes) * metrics.pointsPerMinute, alignment: .leading)
            }
        }
        .padding(.top, 8)
        .frame(width: laneWidth, height: metrics.axisHeight, alignment: .leading)
        .background(Color(.systemBackground))
        .overlay(alignment: .topLeading) { nowPill }
    }

    private var axisSlots: [Date] {
        let count = viewModel.windowMinutes / Self.slotMinutes
        guard count > 0 else { return [] }
        return (0..<count).map {
            viewModel.windowStart.addingTimeInterval(Double($0 * Self.slotMinutes * 60))
        }
    }

    private func lane(for entry: ChannelGuideEntry) -> some View {
        HStack(spacing: 0) {
            if entry.programs.isEmpty {
                Text("No content scheduled")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.horizontal, 8)
                    .frame(width: laneWidth - 2, height: metrics.rowHeight - 2, alignment: .leading)
                    .background(Color(.systemGray6), in: RoundedRectangle(cornerRadius: 6))
            } else {
                ForEach(entry.programs, id: \.airingId) { program in
                    programBlock(program, in: entry)
                }
            }
        }
        .frame(height: metrics.rowHeight, alignment: .top)
    }

    private var laneWidth: CGFloat {
        CGFloat(viewModel.windowMinutes) * metrics.pointsPerMinute
    }

    /// Programs are contiguous, so laying them out in an HStack at their clipped
    /// widths reproduces the time axis exactly — no absolute positioning needed.
    private func width(of program: GuideProgram) -> CGFloat {
        let from = max(program.startTime, viewModel.windowStart)
        let to = min(program.endTime, viewModel.windowEnd)
        let minutes = to.timeIntervalSince(from) / 60
        return max(0, CGFloat(minutes) * metrics.pointsPerMinute)
    }

    private func programBlock(_ program: GuideProgram, in entry: ChannelGuideEntry) -> some View {
        let blockWidth = width(of: program)
        let isLive = program.isAiring(at: viewModel.now)
        let times = "\(Self.clock.string(from: program.startTime)) – \(Self.clock.string(from: program.endTime))"

        return Button {
            select(program, in: entry)
        } label: {
            VStack(alignment: .leading, spacing: 2) {
                Text(displayTitle(of: program))
                    .font(.footnote.weight(.semibold))
                    .foregroundColor(.primary)
                    .lineLimit(1)
                Text(times)
                    .font(.caption2.monospacedDigit())
                    .foregroundColor(.secondary)
                    .lineLimit(1)
            }
            .padding(.horizontal, 8)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
            .background(
                RoundedRectangle(cornerRadius: 6)
                    .fill(isLive ? Color(.systemGray4) : Color(.systemGray6))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 6)
                    .strokeBorder(isSelected(program, in: entry) ? Color.orange : Color.clear, lineWidth: 2)
            )
            .clipShape(RoundedRectangle(cornerRadius: 6))
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .frame(width: max(blockWidth - 2, 0), height: metrics.rowHeight - 2)
        .padding(.trailing, min(2, blockWidth))
        .accessibilityLabel("\(displayTitle(of: program)), \(times)")
    }

    // MARK: - Playhead

    private var nowInWindow: Bool {
        viewModel.now >= viewModel.windowStart && viewModel.now <= viewModel.windowEnd
    }

    private var nowX: CGFloat {
        let minutes = viewModel.now.timeIntervalSince(viewModel.windowStart) / 60
        return min(laneWidth, max(0, CGFloat(minutes) * metrics.pointsPerMinute))
    }

    /// The line every lane is read against, with the time already gone tinted
    /// behind it so "how far in" is visible on every channel at once.
    private var playhead: some View {
        ZStack(alignment: .topLeading) {
            Rectangle()
                .fill(Color.orange.opacity(0.1))
                .frame(width: nowX)
            Rectangle()
                .fill(Color.orange)
                .frame(width: 2)
                .offset(x: nowX - 1)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .opacity(nowInWindow ? 1 : 0)
        .allowsHitTesting(false)
    }

    private var nowPill: some View {
        Text(Self.clock.string(from: viewModel.now))
            .font(.caption2.weight(.bold).monospacedDigit())
            .foregroundColor(.black)
            .frame(width: Self.pillWidth, height: 20)
            .background(Color.orange, in: Capsule())
            .offset(x: max(0, nowX - Self.pillWidth / 2))
            .opacity(nowInWindow ? 1 : 0)
            .allowsHitTesting(false)
    }

    /// A zero-height marker at the playhead's position. `.offset` doesn't move
    /// a view's layout frame, so the playhead itself can't be a scroll target.
    private var nowAnchor: some View {
        HStack(spacing: 0) {
            Color.clear.frame(width: nowX, height: 0)
            Color.clear.frame(width: 1, height: 0).id(Self.nowAnchorId)
        }
    }

    // MARK: - Actions

    private func tune(to entry: ChannelGuideEntry) {
        guard let index = channels.firstIndex(where: { $0.id == entry.channel.id }) else { return }
        onSelectChannel(entry.channel, index)
    }

    private static let slotMinutes = 30
    private static let pillWidth: CGFloat = 46
    private static let scrollSpace = "guideScroll"
    private static let nowAnchorId = "guideNow"

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
}

private struct GuideSelection: Equatable {
    let channelId: Int64
    let airingId: String
}

/// Grid geometry per orientation. Landscape trades row height for a longer
/// stretch of schedule, since height is what a phone on its side is short of.
private struct GuideMetrics {
    let pointsPerMinute: CGFloat
    let railWidth: CGFloat
    let rowHeight: CGFloat
    let axisHeight: CGFloat

    static let portrait = GuideMetrics(pointsPerMinute: 3.6, railWidth: 64, rowHeight: 62, axisHeight: 34)
    static let landscape = GuideMetrics(pointsPerMinute: 4.4, railWidth: 76, rowHeight: 48, axisHeight: 30)
}

private struct GuideScrollOffsetKey: PreferenceKey {
    static var defaultValue: CGFloat = 0
    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

/// Backdrop art for the detail area, falling back to the channel emoji for
/// items the scanner never matched (extras carry no artwork at all).
private struct GuideBackdropView: View {
    let backdropPath: String?
    let fallback: String

    var body: some View {
        Group {
            if let backdropPath, !backdropPath.isEmpty,
               let url = URL(string: "https://image.tmdb.org/t/p/w300\(backdropPath)") {
                AsyncImage(url: url) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    placeholder
                }
            } else {
                placeholder
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    private var placeholder: some View {
        ZStack {
            Color(.systemGray5)
            Text(fallback).font(.system(size: 28))
        }
    }
}
