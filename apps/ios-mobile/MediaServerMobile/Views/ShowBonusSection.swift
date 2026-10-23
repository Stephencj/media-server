import SwiftUI

/// The part of a show's page for extras: commentary tracks (with episode
/// labels) and bonus clips by kind. Shows nothing for a show that has none.
struct ShowBonusSection: View {
    @StateObject private var viewModel: ShowBonusViewModel
    @State private var playing: Media?

    init(showId: Int64) {
        _viewModel = StateObject(wrappedValue: ShowBonusViewModel(showId: showId))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            if !viewModel.commentaries.isEmpty {
                section("Commentaries", note: "The episode with a different audio track") {
                    ForEach(viewModel.commentaries) { extra in
                        row(
                            title: extra.title,
                            detail: [extra.episodeLabel, extra.formattedDuration].compactMap { $0 }.joined(separator: " \u{00B7} ")
                        ) {
                            playing = extra.toMedia()
                        }
                    }
                }
            }

            ForEach(viewModel.groups) { group in
                section(group.title, note: nil) {
                    ForEach(group.extras) { extra in
                        row(title: extra.title, detail: extra.formattedDuration) { playing = extra.toMedia() }
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .fullScreenCover(item: $playing) { media in
            PlayerView(media: media)
        }
        .task { await viewModel.load() }
    }

    private func section<Rows: View>(_ title: String, note: String?, @ViewBuilder rows: () -> Rows) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.headline)
            if let note {
                Text(note)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            VStack(spacing: 0) {
                rows()
            }
            .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))
            .padding(.top, 4)
        }
    }

    private func row(title: String, detail: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: "play.circle.fill")
                    .font(.title2)
                    .foregroundColor(.blue)
                VStack(alignment: .leading, spacing: 2) {
                    Text(title)
                        .font(.subheadline.weight(.medium))
                        .foregroundColor(.primary)
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)
                    if !detail.isEmpty {
                        Text(detail)
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
