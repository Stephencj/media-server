import SwiftUI

/// Extras on a show's page: a heading, then one shelf of cards per kind,
/// each scrolling sideways so a show with hundreds of extras doesn't push
/// the rest of the page out of reach. Shows nothing when there are none.
struct ShowBonusSection: View {
    let title: String
    let groups: [ShowExtraGroup]
    let play: (ShowExtra) -> Void

    var body: some View {
        if !groups.isEmpty {
            VStack(alignment: .leading, spacing: 24) {
                Text(title)
                    .font(.title3.weight(.semibold))
                ForEach(groups) { group in
                    shelf(group)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.top, 30)
        }
    }

    private func shelf(_ group: ShowExtraGroup) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                Text(group.title)
                    .font(.headline)
                Text("\(group.extras.count)")
                    .font(.caption.monospacedDigit())
                    .foregroundColor(.secondary)
            }
            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(spacing: 30) {
                    ForEach(group.extras) { extra in
                        card(extra)
                    }
                }
                .padding(.vertical, 20)
                .padding(.horizontal, 4)
            }
            .focusSection()
        }
        .accessibilityElement(children: .contain)
        .accessibilityIdentifier("extrasShelf-\(group.title)")
    }

    private func card(_ extra: ShowExtra) -> some View {
        Button { play(extra) } label: {
            VStack(alignment: .leading, spacing: 8) {
                Image(systemName: "play.circle.fill")
                    .font(.title2)
                Text(extra.displayTitle)
                    .font(.callout.weight(.medium))
                    .lineLimit(2)
                    .multilineTextAlignment(.leading)
                if !extra.detail.isEmpty {
                    Text(extra.detail)
                        .font(.caption)
                        .monospacedDigit()
                        .foregroundColor(.secondary)
                }
            }
            .padding(24)
            .frame(width: 360, height: 190, alignment: .topLeading)
        }
        .buttonStyle(.card)
    }
}
