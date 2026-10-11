import SwiftUI

/// The queue of movies whose match looks wrong. Each row sets what the file
/// says beside what it is shown as and what it would be changed to, so the
/// decision can be made from the couch without opening anything.
struct MatchReviewView: View {
    @StateObject private var viewModel = MatchReviewViewModel()
    @State private var fixing: MatchReviewItem?

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 24) {
                header

                ForEach(viewModel.items) { item in
                    MatchReviewRow(
                        item: item,
                        isBusy: viewModel.busy.contains(item.mediaId) || viewModel.isAcceptingAll,
                        onAccept: { Task { await viewModel.accept(item) } },
                        onOther: { fixing = item },
                        onKeep: { Task { await viewModel.keep(item) } }
                    )
                }
            }
            .padding(50)
        }
        .navigationTitle("Check Matches")
        .navigationDestination(item: $fixing) { item in
            FixMatchView(mediaId: item.mediaId, fileName: item.fileName, fileFacts: item.fileFacts) {
                viewModel.remove(mediaId: item.mediaId)
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.items.isEmpty {
                ProgressView()
            }
        }
        .task { await viewModel.load() }
        .onDisappear { viewModel.stopPolling() }
    }

    private var header: some View {
        HStack(alignment: .center, spacing: 24) {
            VStack(alignment: .leading, spacing: 10) {
                Text(viewModel.statusText)
                    .font(.headline)
                    .foregroundStyle(.secondary)

                if viewModel.isChecking, let status = viewModel.status, status.total > 0 {
                    ProgressView(value: Double(status.checked), total: Double(status.total))
                        .tint(.orange)
                        .frame(maxWidth: 600)
                }

                if let error = viewModel.errorMessage {
                    Text(error)
                        .font(.subheadline)
                        .foregroundStyle(.red)
                }
            }

            Spacer(minLength: 0)

            Button(viewModel.isChecking ? "Checking…" : "Check Library") {
                Task { await viewModel.startCheck() }
            }
            .disabled(viewModel.isChecking)

            if viewModel.items.count > 1 {
                Button("Accept All") {
                    Task { await viewModel.acceptAll() }
                }
                .disabled(viewModel.isAcceptingAll)
            }
        }
        .focusSection()
    }
}

private struct MatchReviewRow: View {
    let item: MatchReviewItem
    let isBusy: Bool
    let onAccept: () -> Void
    let onOther: () -> Void
    let onKeep: () -> Void

    var body: some View {
        HStack(alignment: .center, spacing: 32) {
            VStack(alignment: .leading, spacing: 6) {
                label("YOUR FILE", accent: false)
                Text(item.fileName)
                    .font(.headline)
                    .lineLimit(2)
                Text(item.fileFacts)
                    .font(.subheadline)
                    .monospacedDigit()
                    .foregroundStyle(.secondary)
            }
            .frame(width: 400, alignment: .leading)

            side(title: "SHOWN AS", film: item.current, fallback: "Not matched", detail: nil, accent: false)
                .frame(width: 360, alignment: .leading)

            side(title: "SUGGESTED", film: item.suggested, fallback: "Use the file name", detail: item.reason, accent: true)
                .frame(maxWidth: .infinity, alignment: .leading)

            HStack(spacing: 16) {
                Button("Accept", action: onAccept)
                Button("Other", action: onOther)
                Button("Keep", action: onKeep)
                    // There is nothing to keep when the movie has no match.
                    .disabled(item.current == nil)
            }
            .disabled(isBusy)
        }
        .padding(28)
        .background(Color.white.opacity(0.06), in: RoundedRectangle(cornerRadius: 18))
        .focusSection()
    }

    private func label(_ text: String, accent: Bool) -> some View {
        Text(text)
            .font(.caption.weight(.bold))
            .foregroundStyle(accent ? Color.orange : Color.secondary)
    }

    private func side(title: String, film: MatchFilm?, fallback: String, detail: String?, accent: Bool) -> some View {
        HStack(alignment: .top, spacing: 16) {
            MatchPosterView(posterPath: film?.posterPath)
                .frame(width: 80, height: 120)

            VStack(alignment: .leading, spacing: 6) {
                label(title, accent: accent)
                Text(film?.title ?? fallback)
                    .font(.headline)
                    .lineLimit(2)
                if let film {
                    Text(facts(for: film))
                        .font(.subheadline)
                        .monospacedDigit()
                        .foregroundStyle(.secondary)
                }
                if let detail, !detail.isEmpty {
                    Text(detail)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(3)
                }
            }
        }
    }

    private func facts(for film: MatchFilm) -> String {
        var parts: [String] = []
        if film.year > 0 { parts.append(String(film.year)) }
        if film.runtime > 0 { parts.append(MatchFormat.minutes(film.runtime)) }
        return parts.joined(separator: " · ")
    }
}

/// Choosing a movie's match by hand: the films it could be, each with how its
/// year and running time compare with the file. Selecting one applies it.
struct FixMatchView: View {
    let fileName: String
    let fileFacts: String
    let onChosen: () -> Void

    @StateObject private var viewModel: FixMatchViewModel
    @Environment(\.dismiss) private var dismiss

    init(mediaId: Int64, fileName: String, fileFacts: String, onChosen: @escaping () -> Void) {
        self.fileName = fileName
        self.fileFacts = fileFacts
        self.onChosen = onChosen
        _viewModel = StateObject(wrappedValue: FixMatchViewModel(mediaId: mediaId))
    }

    var body: some View {
        HStack(alignment: .top, spacing: 56) {
            VStack(alignment: .leading, spacing: 24) {
                VStack(alignment: .leading, spacing: 10) {
                    Text("YOUR FILE")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                    Text(fileName)
                        .font(.headline)
                    Text(fileFacts)
                        .font(.subheadline)
                        .monospacedDigit()
                        .foregroundStyle(.secondary)
                }
                .padding(28)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Color.white.opacity(0.06), in: RoundedRectangle(cornerRadius: 18))

                TextField("Search another title", text: $viewModel.query)
                    .onSubmit { Task { await viewModel.search() } }

                if let error = viewModel.errorMessage {
                    Text(error)
                        .font(.subheadline)
                        .foregroundStyle(.red)
                }
            }
            .frame(width: 520)
            .focusSection()

            ScrollView {
                LazyVStack(spacing: 20) {
                    if viewModel.isLoading {
                        ProgressView()
                            .padding(.top, 80)
                    } else if viewModel.candidates.isEmpty {
                        Text("No films found. Try a different title.")
                            .foregroundStyle(.secondary)
                            .padding(.top, 80)
                    } else {
                        ForEach(viewModel.candidates) { candidate in
                            Button {
                                Task {
                                    if await viewModel.choose(candidate) {
                                        onChosen()
                                        dismiss()
                                    }
                                }
                            } label: {
                                candidateRow(candidate)
                            }
                            .buttonStyle(.card)
                            .disabled(viewModel.isChoosing)
                        }
                    }
                }
                .padding(.horizontal, 20)
                .padding(.vertical, 20)
            }
            .focusSection()
        }
        .padding(50)
        .navigationTitle("Which movie is this?")
        .task { await viewModel.search() }
    }

    private func candidateRow(_ candidate: MatchCandidate) -> some View {
        HStack(alignment: .center, spacing: 24) {
            MatchPosterView(posterPath: candidate.posterPath)
                .frame(width: 80, height: 120)

            VStack(alignment: .leading, spacing: 6) {
                Text(candidate.title)
                    .font(.headline)
                    .lineLimit(1)
                Text(candidate.facts)
                    .font(.subheadline)
                    .monospacedDigit()
                    .foregroundStyle(.secondary)
                if !candidate.fit.isEmpty {
                    Text(candidate.fit)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(candidate.isBestFit ? Color.orange : Color.secondary)
                }
            }

            Spacer(minLength: 0)

            if candidate.isBestFit {
                badge("Best fit", background: .orange, foreground: .black)
            } else if candidate.isCurrent {
                badge("Current", background: Color.white.opacity(0.18), foreground: .primary)
            }
        }
        .padding(20)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func badge(_ text: String, background: Color, foreground: Color) -> some View {
        Text(text)
            .font(.caption.weight(.bold))
            .foregroundStyle(foreground)
            .padding(.horizontal, 14)
            .padding(.vertical, 6)
            .background(background, in: Capsule())
    }
}

/// A small poster, so a wrong match is visible at a glance.
struct MatchPosterView: View {
    let posterPath: String?

    var body: some View {
        Group {
            if let posterPath, !posterPath.isEmpty,
               let url = URL(string: "https://image.tmdb.org/t/p/w185\(posterPath)") {
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
            Color.gray.opacity(0.25)
            Image(systemName: "film")
                .foregroundStyle(.secondary)
        }
    }
}
