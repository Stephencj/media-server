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
        VStack(alignment: .leading, spacing: 20) {
            if !viewModel.commentaries.isEmpty {
                section("Commentaries", note: "The whole film with a different audio track") {
                    ForEach(viewModel.commentaries) { extra in
                        row(title: extra.title, detail: extra.formattedDuration) { playing = extra.toMedia() }
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

            if !viewModel.copies.isEmpty {
                section("Other Copies", note: "More files of this film in your library") {
                    ForEach(viewModel.copies) { copy in
                        row(title: MovieBonusViewModel.fileName(of: copy),
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
