import SwiftUI

/// Extras on a show's page, by kind, each kind folded away until opened:
/// a show can have a few hundred. Shows nothing when there are none.
struct ShowBonusSection: View {
    let title: String
    let groups: [ShowExtraGroup]
    let play: (ShowExtra) -> Void

    var body: some View {
        if !groups.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                Text(title)
                    .font(.headline)
                VStack(spacing: 0) {
                    ForEach(groups) { group in
                        ShowBonusGroup(group: group, play: play)
                        if group != groups.last {
                            Divider().padding(.leading, 12)
                        }
                    }
                }
                .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

private struct ShowBonusGroup: View {
    let group: ShowExtraGroup
    let play: (ShowExtra) -> Void
    @State private var open = false

    var body: some View {
        VStack(spacing: 0) {
            Button {
                withAnimation(.easeInOut(duration: 0.2)) { open.toggle() }
            } label: {
                HStack {
                    Text(group.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundColor(.primary)
                    Text("\(group.extras.count)")
                        .font(.caption.monospacedDigit())
                        .foregroundColor(.secondary)
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundColor(.secondary)
                        .rotationEffect(.degrees(open ? 90 : 0))
                }
                .padding(.horizontal, 12)
                .frame(minHeight: 48)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityIdentifier("extrasGroup-\(group.title)")

            if open {
                ForEach(group.extras) { extra in
                    ShowBonusRow(extra: extra) { play(extra) }
                }
            }
        }
    }
}

struct ShowBonusRow: View {
    let extra: ShowExtra
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: "play.circle.fill")
                    .font(.title2)
                    .foregroundColor(.blue)
                VStack(alignment: .leading, spacing: 2) {
                    Text(extra.displayTitle)
                        .font(.subheadline.weight(.medium))
                        .foregroundColor(.primary)
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)
                    if !extra.detail.isEmpty {
                        Text(extra.detail)
                            .font(.caption.monospacedDigit())
                            .foregroundColor(.secondary)
                    }
                }
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 52)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
