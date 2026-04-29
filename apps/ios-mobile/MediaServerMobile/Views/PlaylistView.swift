import SwiftUI
import AVKit

struct PlaylistView: View {
    @StateObject private var viewModel = PlaylistViewModel()
    @State private var showCreateSheet = false
    @State private var newPlaylistName = ""
    @State private var newPlaylistDescription = ""
    @State private var selectedPlaylist: Playlist?
    @State private var navigateToPlaylist = false

    var body: some View {
        VStack {
            if viewModel.isLoading && viewModel.playlists.isEmpty {
                ProgressView()
            } else if viewModel.playlists.isEmpty {
                VStack(spacing: 12) {
                    Image(systemName: "music.note.list")
                        .font(.system(size: 48))
                        .foregroundColor(.secondary)
                    Text("No Playlists")
                        .font(.title2)
                        .fontWeight(.semibold)
                    Text("Create a playlist to organize your media")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ScrollView {
                    LazyVGrid(columns: [
                        GridItem(.adaptive(minimum: 150), spacing: 16)
                    ], spacing: 16) {
                        ForEach(viewModel.playlists) { playlist in
                            Button {
                                selectedPlaylist = playlist
                                navigateToPlaylist = true
                            } label: {
                                PlaylistCard(playlist: playlist)
                            }
                            .buttonStyle(.plain)
                            .contextMenu {
                                Button(role: .destructive) {
                                    Task { await viewModel.deletePlaylist(playlist) }
                                } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                            }
                        }
                    }
                    .padding(16)
                }
            }

            // Hidden NavigationLink for programmatic navigation
            if let playlist = selectedPlaylist {
                NavigationLink(
                    destination: PlaylistDetailView(playlistId: playlist.id),
                    isActive: $navigateToPlaylist
                ) {
                    EmptyView()
                }
                .hidden()
            }
        }
        .navigationTitle("Playlists")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    showCreateSheet = true
                } label: {
                    Label("New Playlist", systemImage: "plus")
                }
            }
        }
        .sheet(isPresented: $showCreateSheet) {
            CreatePlaylistSheet(
                name: $newPlaylistName,
                description: $newPlaylistDescription,
                onCreate: {
                    Task {
                        if await viewModel.createPlaylist(name: newPlaylistName, description: newPlaylistDescription) {
                            newPlaylistName = ""
                            newPlaylistDescription = ""
                            showCreateSheet = false
                        }
                    }
                },
                onCancel: {
                    newPlaylistName = ""
                    newPlaylistDescription = ""
                    showCreateSheet = false
                }
            )
        }
        .task {
            await viewModel.loadPlaylists()
        }
        .refreshable {
            await viewModel.loadPlaylists()
        }
    }
}

struct PlaylistCard: View {
    let playlist: Playlist

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: "music.note.list")
                    .font(.title2)
                    .foregroundColor(.accentColor)

                Spacer()

                Text("\(playlist.itemCount)")
                    .font(.headline)
                    .foregroundColor(.secondary)
            }

            Text(playlist.name)
                .font(.headline)
                .lineLimit(1)

            if let description = playlist.description, !description.isEmpty {
                Text(description)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineLimit(2)
            }
        }
        .padding(16)
        .frame(minHeight: 100)
        .background(Color.gray.opacity(0.2))
        .cornerRadius(12)
    }
}

struct CreatePlaylistSheet: View {
    @Binding var name: String
    @Binding var description: String
    let onCreate: () -> Void
    let onCancel: () -> Void

    @FocusState private var focusedField: Field?

    enum Field {
        case name, description
    }

    var body: some View {
        NavigationView {
            Form {
                Section {
                    TextField("Playlist Name", text: $name)
                        .focused($focusedField, equals: .name)

                    TextField("Description (optional)", text: $description)
                        .focused($focusedField, equals: .description)
                }
            }
            .navigationTitle("New Playlist")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel", action: onCancel)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Create", action: onCreate)
                        .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .onAppear {
                focusedField = .name
            }
        }
    }
}

struct PlaylistDetailView: View {
    let playlistId: Int64
    @StateObject private var viewModel = PlaylistDetailViewModel()
    @State private var showPlayer = false

    var body: some View {
        VStack {
            if viewModel.isLoading && viewModel.playlist == nil {
                ProgressView()
            } else if let playlist = viewModel.playlist {
                VStack(alignment: .leading, spacing: 16) {
                    // Header
                    HStack {
                        VStack(alignment: .leading) {
                            Text(playlist.name)
                                .font(.title)
                                .fontWeight(.bold)

                            if let desc = playlist.description, !desc.isEmpty {
                                Text(desc)
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }

                            Text("\(viewModel.items.count) items")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }

                        Spacer()

                        HStack(spacing: 12) {
                            Button {
                                viewModel.playAll()
                                showPlayer = true
                            } label: {
                                Label("Play All", systemImage: "play.fill")
                            }
                            .disabled(viewModel.items.isEmpty)

                            Button {
                                viewModel.shuffle()
                                showPlayer = true
                            } label: {
                                Label("Shuffle", systemImage: "shuffle")
                            }
                            .disabled(viewModel.items.isEmpty)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.top, 12)

                    // Items List
                    if viewModel.items.isEmpty {
                        VStack(spacing: 12) {
                            Image(systemName: "music.note")
                                .font(.system(size: 48))
                                .foregroundColor(.secondary)
                            Text("Empty Playlist")
                                .font(.title2)
                                .fontWeight(.semibold)
                            Text("Add items from your library")
                                .font(.subheadline)
                                .foregroundColor(.secondary)
                                .multilineTextAlignment(.center)
                        }
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else {
                        List {
                            ForEach(Array(viewModel.items.enumerated()), id: \.element.id) { index, item in
                                PlaylistItemRow(item: item, index: index + 1)
                                    .onTapGesture {
                                        viewModel.playItem(at: index)
                                        showPlayer = true
                                    }
                                    .contextMenu {
                                        Button(role: .destructive) {
                                            Task { await viewModel.removeItem(item) }
                                        } label: {
                                            Label("Remove", systemImage: "trash")
                                        }
                                    }
                            }
                            .onMove(perform: viewModel.moveItem)
                        }
                    }
                }
            }
        }
        .fullScreenCover(isPresented: $showPlayer) {
            if let _ = viewModel.currentItem {
                PlaylistPlayerView(viewModel: viewModel)
            }
        }
        .task {
            await viewModel.loadPlaylist(id: playlistId)
        }
    }
}

struct PlaylistItemRow: View {
    let item: PlaylistItem
    let index: Int

    var body: some View {
        HStack(spacing: 12) {
            Text("\(index)")
                .font(.headline)
                .foregroundColor(.secondary)
                .frame(width: 24)

            if let posterPath = item.posterPath {
                AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w92\(posterPath)")) { image in
                    image.resizable().aspectRatio(contentMode: .fill)
                } placeholder: {
                    Color.gray.opacity(0.3)
                }
                .frame(width: 44, height: 66)
                .cornerRadius(4)
            } else {
                Color.gray.opacity(0.3)
                    .frame(width: 44, height: 66)
                    .cornerRadius(4)
            }

            VStack(alignment: .leading, spacing: 4) {
                Text(item.title)
                    .font(.headline)
                    .lineLimit(1)

                HStack {
                    if let year = item.year {
                        Text(String(year))
                    }
                    if let resolution = item.resolution {
                        Text(resolution)
                    }
                }
                .font(.subheadline)
                .foregroundColor(.secondary)

                if !item.formattedDuration.isEmpty {
                    Text(item.formattedDuration)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            Spacer()

            Image(systemName: "line.3.horizontal")
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }
}

struct PlaylistPlayerView: View {
    @ObservedObject var viewModel: PlaylistDetailViewModel
    @Environment(\.dismiss) private var dismiss

    @State private var showUpNext = false
    @State private var autoPlayCountdown = 10
    @State private var countdownTimer: Timer?

    var body: some View {
        ZStack {
            // Video Player
            if let playerVM = viewModel.playerViewModel {
                VideoPlayer(player: playerVM.player)
                    .ignoresSafeArea()
                    .onAppear {
                        Task { await playerVM.loadAndPlay() }
                    }

                // Loading indicator
                if playerVM.isBuffering {
                    ProgressView()
                        .scaleEffect(2)
                }
            } else {
                Color.black.ignoresSafeArea()
            }

            // Dismiss button
            VStack {
                HStack {
                    Button {
                        viewModel.playerViewModel?.pause()
                        dismiss()
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.title)
                            .foregroundColor(.white)
                            .shadow(radius: 4)
                    }
                    .padding(16)

                    Spacer()
                }
                Spacer()
            }

            // Up Next overlay
            if showUpNext, let nextItem = viewModel.nextItem {
                VStack {
                    Spacer()

                    HStack {
                        Spacer()

                        VStack(alignment: .leading, spacing: 12) {
                            Text("Up Next in \(autoPlayCountdown)s")
                                .font(.caption)
                                .foregroundColor(.secondary)

                            HStack(spacing: 12) {
                                if let posterPath = nextItem.posterPath {
                                    AsyncImage(url: URL(string: "https://image.tmdb.org/t/p/w92\(posterPath)")) { image in
                                        image.resizable().aspectRatio(contentMode: .fill)
                                    } placeholder: {
                                        Color.gray.opacity(0.3)
                                    }
                                    .frame(width: 44, height: 66)
                                    .cornerRadius(4)
                                }

                                VStack(alignment: .leading, spacing: 4) {
                                    Text(nextItem.title)
                                        .font(.headline)
                                        .foregroundColor(.white)

                                    if let year = nextItem.year {
                                        Text(String(year))
                                            .font(.subheadline)
                                            .foregroundColor(.secondary)
                                    }
                                }
                            }

                            HStack(spacing: 12) {
                                Button("Play Now") {
                                    cancelAutoPlayTimer()
                                    showUpNext = false
                                    if viewModel.playNext() {
                                        // Player will be updated automatically via setupPlayer()
                                    }
                                }
                                .buttonStyle(.borderedProminent)

                                Button("Cancel") {
                                    cancelAutoPlayTimer()
                                    showUpNext = false
                                    dismiss()
                                }
                                .buttonStyle(.bordered)
                            }
                        }
                        .padding()
                        .background(Color.black.opacity(0.9))
                        .cornerRadius(12)
                        .padding()
                    }
                }
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: .AVPlayerItemDidPlayToEndTime)) { notification in
            // Check if this notification is for our current player
            if let playerVM = viewModel.playerViewModel,
               notification.object as? AVPlayerItem === playerVM.player.currentItem {
                handleVideoEnd()
            }
        }
        .onDisappear {
            cancelAutoPlayTimer()
            viewModel.cleanup()
        }
    }

    private func handleVideoEnd() {
        // Mark video as completed
        viewModel.playerViewModel?.markAsCompleted()
        viewModel.handleVideoEnd()

        if viewModel.hasNextItem {
            showUpNext = true
            startAutoPlayTimer()
        } else {
            // Last item in playlist - dismiss player
            dismiss()
        }
    }

    private func startAutoPlayTimer() {
        autoPlayCountdown = 10
        countdownTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { timer in
            if autoPlayCountdown > 0 && showUpNext {
                autoPlayCountdown -= 1
                if autoPlayCountdown == 0 {
                    timer.invalidate()
                    showUpNext = false
                    if viewModel.playNext() {
                        // Player will be updated automatically via setupPlayer()
                    }
                }
            } else {
                timer.invalidate()
            }
        }
    }

    private func cancelAutoPlayTimer() {
        countdownTimer?.invalidate()
        countdownTimer = nil
    }
}

#Preview {
    PlaylistView()
}
