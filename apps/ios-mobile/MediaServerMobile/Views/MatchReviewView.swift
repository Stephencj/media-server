import SwiftUI

/// The queue of movies whose match looks wrong. Each card sets what the file
/// says beside what it is shown as and what it would be changed to, so the
/// decision can be made without opening anything.
struct MatchReviewView: View {
    @StateObject private var viewModel = MatchReviewViewModel()
    @State private var fixing: MatchReviewItem?

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 12) {
                header

                ForEach(viewModel.items) { item in
                    MatchReviewCard(
                        item: item,
                        isBusy: viewModel.busy.contains(item.mediaId) || viewModel.isAcceptingAll,
                        onAccept: { Task { await viewModel.accept(item) } },
                        onOther: { fixing = item },
                        onKeep: { Task { await viewModel.keep(item) } }
                    )
                }
            }
            .padding(16)
        }
        .navigationTitle("Check Matches")
        .navigationBarTitleDisplayMode(.inline)
        .overlay {
            if viewModel.isLoading && viewModel.items.isEmpty {
                ProgressView()
            }
        }
        .sheet(item: $fixing) { item in
            NavigationView {
                FixMatchView(mediaId: item.mediaId, fileName: item.fileName, fileFacts: item.fileFacts) {
                    viewModel.remove(mediaId: item.mediaId)
                }
            }
        }
        .task { await viewModel.load() }
        .onDisappear { viewModel.stopPolling() }
        .refreshable { await viewModel.load() }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(viewModel.statusText)
                .font(.subheadline)
                .foregroundColor(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)

            if viewModel.isChecking, let status = viewModel.status, status.total > 0 {
                ProgressView(value: Double(status.checked), total: Double(status.total))
                    .tint(.orange)
            }

            if let error = viewModel.errorMessage {
                Text(error)
                    .font(.footnote)
                    .foregroundColor(.red)
            }

            HStack(spacing: 10) {
                Button {
                    Task { await viewModel.startCheck() }
                } label: {
                    Text(viewModel.isChecking ? "Checking…" : "Check Library")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .disabled(viewModel.isChecking)

                if viewModel.items.count > 1 {
                    Button {
                        Task { await viewModel.acceptAll() }
                    } label: {
                        Text("Accept All")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.orange)
                    .disabled(viewModel.isAcceptingAll)
                }
            }
            .controlSize(.large)
        }
        .padding(.bottom, 4)
    }
}

private struct MatchReviewCard: View {
    let item: MatchReviewItem
    let isBusy: Bool
    let onAccept: () -> Void
    let onOther: () -> Void
    let onKeep: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            VStack(alignment: .leading, spacing: 2) {
                Text(item.fileName)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(2)
                Text(item.fileFacts)
                    .font(.caption.monospacedDigit())
                    .foregroundColor(.secondary)
            }

            HStack(alignment: .top, spacing: 12) {
                side(label: "SHOWN AS", film: item.current, fallback: "Not matched", accent: false)
                side(label: "SUGGESTED", film: item.suggested, fallback: "Use the file name", accent: true)
            }

            Text(item.reason)
                .font(.footnote)
                .foregroundColor(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            HStack(spacing: 8) {
                Button(action: onAccept) {
                    Text("Accept").frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(.orange)

                Button(action: onOther) {
                    Text("Other").frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)

                Button(action: onKeep) {
                    Text("Keep").frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                // There is nothing to keep when the movie has no match.
                .disabled(item.current == nil)
            }
            .controlSize(.large)
            .disabled(isBusy)
        }
        .padding(14)
        .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 14))
    }

    private func side(label: String, film: MatchFilm?, fallback: String, accent: Bool) -> some View {
        HStack(alignment: .top, spacing: 8) {
            MatchPosterView(posterPath: film?.posterPath)
                .frame(width: 46, height: 69)

            VStack(alignment: .leading, spacing: 2) {
                Text(label)
                    .font(.caption2.weight(.bold))
                    .foregroundColor(accent ? .orange : .secondary)
                Text(film?.title ?? fallback)
                    .font(.footnote.weight(.semibold))
                    .lineLimit(2)
                if let film {
                    Text(facts(for: film))
                        .font(.caption2.monospacedDigit())
                        .foregroundColor(.secondary)
                }
            }
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func facts(for film: MatchFilm) -> String {
        var parts: [String] = []
        if film.year > 0 { parts.append(String(film.year)) }
        if film.runtime > 0 { parts.append(MatchFormat.minutes(film.runtime)) }
        return parts.joined(separator: " · ")
    }
}

/// Choosing a movie's match by hand: the films it could be, each with how its
/// year and running time compare with the file.
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
        List {
            Section {
                VStack(alignment: .leading, spacing: 2) {
                    Text(fileName)
                        .font(.subheadline.weight(.semibold))
                    Text(fileFacts)
                        .font(.caption.monospacedDigit())
                        .foregroundColor(.secondary)
                }
            } header: {
                Text("Your file")
            }

            Section {
                if viewModel.isLoading {
                    HStack {
                        Spacer()
                        ProgressView()
                        Spacer()
                    }
                } else if viewModel.candidates.isEmpty {
                    Text("No films found. Try a different title.")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
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
                        .buttonStyle(.plain)
                        .disabled(viewModel.isChoosing)
                    }
                }
            } header: {
                Text("Which movie is this?")
            } footer: {
                if let error = viewModel.errorMessage {
                    Text(error).foregroundColor(.red)
                }
            }
        }
        .navigationTitle("Fix Match")
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $viewModel.query, prompt: "Search another title")
        .onSubmit(of: .search) {
            Task { await viewModel.search() }
        }
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
            }
        }
        .task { await viewModel.search() }
    }

    private func candidateRow(_ candidate: MatchCandidate) -> some View {
        HStack(alignment: .top, spacing: 12) {
            MatchPosterView(posterPath: candidate.posterPath)
                .frame(width: 46, height: 69)

            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    Text(candidate.title)
                        .font(.subheadline.weight(.semibold))
                        .lineLimit(2)
                    if candidate.isBestFit {
                        badge("Best fit", background: .orange, foreground: .black)
                    } else if candidate.isCurrent {
                        badge("Current", background: Color(.systemGray4), foreground: .primary)
                    }
                }
                Text(candidate.facts)
                    .font(.caption.monospacedDigit())
                    .foregroundColor(.secondary)
                if !candidate.fit.isEmpty {
                    Text(candidate.fit)
                        .font(.caption.weight(.medium))
                        .foregroundColor(candidate.isBestFit ? .orange : .secondary)
                }
            }
            Spacer(minLength: 0)
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
    }

    private func badge(_ text: String, background: Color, foreground: Color) -> some View {
        Text(text)
            .font(.caption2.weight(.bold))
            .foregroundColor(foreground)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(background, in: Capsule())
            .fixedSize()
    }
}

/// A small poster, so a wrong match is visible at a glance.
struct MatchPosterView: View {
    let posterPath: String?

    var body: some View {
        Group {
            if let posterPath, !posterPath.isEmpty,
               let url = URL(string: "https://image.tmdb.org/t/p/w92\(posterPath)") {
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
        .clipShape(RoundedRectangle(cornerRadius: 6))
    }

    private var placeholder: some View {
        ZStack {
            Color(.systemGray5)
            Image(systemName: "film")
                .foregroundColor(.secondary)
        }
    }
}
