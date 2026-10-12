import SwiftUI

/// The part of a film's page for everything kept with it: commentary tracks,
/// bonus clips by kind, and other copies of the film. Shows nothing for a
/// film that has none.
struct MovieBonusSection: View {
    @StateObject private var viewModel: MovieBonusViewModel
    @State private var playing: Media?

    init(media: Media) {
        _viewModel = StateObject(wrappedValue: MovieBonusViewModel(mediaId: media.id))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 36) {
            if !viewModel.commentaries.isEmpty {
                shelf("Commentaries", note: "The whole film with a different audio track") {
                    ForEach(viewModel.commentaries) { extra in
                        card(title: extra.title, detail: extra.formattedDuration) { playing = extra.toMedia() }
                    }
                }
            }

            ForEach(viewModel.groups) { group in
                shelf(group.title, note: nil) {
                    ForEach(group.extras) { extra in
                        card(title: extra.title, detail: extra.formattedDuration) { playing = extra.toMedia() }
                    }
                }
            }

            if !viewModel.copies.isEmpty {
                shelf("Other Copies", note: "More files of this film in your library") {
                    ForEach(viewModel.copies) { copy in
                        card(title: MovieBonusViewModel.fileName(of: copy),
                             detail: MovieBonusViewModel.describe(copy: copy)) { playing = copy }
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

    /// One row of cards that scrolls sideways, so a film with many extras
    /// doesn't push the rest of the page out of reach.
    private func shelf<Cards: View>(_ title: String, note: String?, @ViewBuilder cards: () -> Cards) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 16) {
                Text(title)
                    .font(.headline)
                if let note {
                    Text(note)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 30) {
                    cards()
                }
                .padding(.vertical, 20)
                .padding(.horizontal, 4)
            }
            .focusSection()
        }
    }

    private func card(title: String, detail: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            VStack(alignment: .leading, spacing: 8) {
                Image(systemName: "play.circle.fill")
                    .font(.title2)
                Text(title)
                    .font(.callout.weight(.medium))
                    .lineLimit(2)
                    .multilineTextAlignment(.leading)
                if !detail.isEmpty {
                    Text(detail)
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
