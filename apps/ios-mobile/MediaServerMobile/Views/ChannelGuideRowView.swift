import SwiftUI

/// One channel's row in the phone guide.
///
/// `now` is passed in rather than read from a clock here so every row in the
/// list advances on the same tick.
struct ChannelGuideRowView: View {
    let entry: ChannelGuideEntry
    let now: Date
    let onSelect: () -> Void

    private var nowPlaying: GuideProgram? { entry.program(airingAt: now) }
    private var upNext: GuideProgram? { entry.program(after: now) }

    var body: some View {
        Button(action: onSelect) {
            HStack(alignment: .top, spacing: 12) {
                VStack(spacing: 2) {
                    Text(entry.channel.icon)
                        .font(.system(size: 22))
                    Text("\(entry.channel.displayNumber)")
                        .font(.caption2.monospacedDigit())
                        .foregroundColor(.secondary)
                }
                .frame(width: 52)

                if let nowPlaying {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(nowPlaying.title)
                            .font(.subheadline.weight(.semibold))
                            .foregroundColor(.primary)
                            .lineLimit(1)

                        Text(subtitle(for: nowPlaying))
                            .font(.caption2)
                            .foregroundColor(.secondary)
                            .lineLimit(1)

                        ProgressView(value: nowPlaying.progress(at: now))
                            .tint(.orange)
                            .scaleEffect(x: 1, y: 0.6, anchor: .center)

                        if let upNext {
                            HStack(spacing: 4) {
                                Text(Self.clock.string(from: upNext.startTime))
                                    .font(.caption2.monospacedDigit().weight(.semibold))
                                    .foregroundColor(.secondary)
                                Text(upNext.title)
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                            }
                        }
                    }
                } else {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(entry.channel.name)
                            .font(.subheadline.weight(.semibold))
                            .foregroundColor(.primary)
                        Text("No content scheduled")
                            .font(.caption2)
                            .foregroundColor(.secondary)
                    }
                }

                Spacer(minLength: 0)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private func subtitle(for program: GuideProgram) -> String {
        var parts: [String] = []
        if let showTitle = program.showTitle, !showTitle.isEmpty {
            parts.append(showTitle)
        }
        parts.append("\(program.remaining(at: now) / 60)m left")
        return parts.joined(separator: " · ")
    }

    private static let clock: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "h:mm"
        return f
    }()
}
