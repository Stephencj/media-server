import SwiftUI

struct AddToPlaylistSheet: View {
    let media: Media

    @State private var playlists: [Playlist] = []
    @State private var isLoading = true
    @State private var error: String?
    @State private var showSuccess = false
    @State private var successMessage = ""
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ZStack {
                List {
                    if isLoading {
                        HStack {
                            Spacer()
                            ProgressView()
                            Spacer()
                        }
                    } else if playlists.isEmpty {
                        ContentUnavailableView(
                            "No Playlists",
                            systemImage: "music.note.list",
                            description: Text("Create a playlist first")
                        )
                    } else {
                        ForEach(playlists) { playlist in
                            Button {
                                addToPlaylist(playlist)
                            } label: {
                                HStack {
                                    Text(playlist.name)
                                        .font(.title3)
                                    Spacer()
                                    Text("\(playlist.itemCount) items")
                                        .foregroundColor(.secondary)
                                }
                                .padding(.vertical, 8)
                            }
                        }
                    }

                    if let error = error {
                        Text(error)
                            .foregroundColor(.red)
                            .font(.caption)
                    }
                }

                if showSuccess {
                    VStack {
                        Spacer()
                        HStack {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundColor(.green)
                            Text(successMessage)
                                .foregroundColor(.white)
                        }
                        .padding()
                        .background(Color.black.opacity(0.8))
                        .cornerRadius(10)
                        .padding(.bottom, 50)
                    }
                    .transition(.move(edge: .bottom))
                }
            }
            .navigationTitle("Add to Playlist")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
        .task { await loadPlaylists() }
    }

    private func loadPlaylists() async {
        isLoading = true
        error = nil
        do {
            let response = try await APIClient.shared.getPlaylists()
            playlists = response.items
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    private func addToPlaylist(_ playlist: Playlist) {
        Task {
            do {
                try await APIClient.shared.addToPlaylist(
                    playlistId: playlist.id,
                    mediaId: media.id,
                    mediaType: media.type.rawValue
                )

                // Show success feedback
                successMessage = "Added to \(playlist.name)"
                withAnimation {
                    showSuccess = true
                }

                // Dismiss after showing success
                try? await Task.sleep(nanoseconds: 1_500_000_000) // 1.5 seconds
                dismiss()
            } catch {
                self.error = "Failed to add to playlist: \(error.localizedDescription)"
            }
        }
    }
}

#Preview {
    AddToPlaylistSheet(media: Media(
        id: 1,
        title: "Sample Movie",
        originalTitle: nil,
        type: .movie,
        year: 2024,
        overview: "A sample movie",
        posterPath: nil,
        backdropPath: nil,
        rating: 8.5,
        runtime: 120,
        genres: "Action",
        tmdbId: nil,
        imdbId: nil,
        seasonCount: nil,
        episodeCount: nil,
        sourceId: nil,
        filePath: nil,
        fileSize: nil,
        duration: 7200,
        videoCodec: "h264",
        audioCodec: "aac",
        resolution: "1920x1080",
        audioTracks: nil,
        subtitleTracks: nil,
        createdAt: nil,
        updatedAt: nil
    ))
}
