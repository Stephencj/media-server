# Apple TV Playlist Feature Documentation

## Overview

The playlist feature allows users to create, manage, and play custom playlists of movies and TV shows on Apple TV using the Siri Remote.

## Architecture

### Files Structure

```
MediaPlayer/
├── Models/
│   └── Playlist.swift          # Playlist and PlaylistItem models
├── ViewModels/
│   └── PlaylistViewModel.swift # PlaylistViewModel and PlaylistDetailViewModel
├── Views/
│   └── PlaylistView.swift      # PlaylistView, PlaylistDetailView, PlaylistPlayerView
└── Services/
    └── APIClient.swift         # Extended with playlist API endpoints
```

### Models

#### Playlist
- `id: Int64` - Unique identifier
- `userId: Int64` - Owner user ID
- `name: String` - Playlist name
- `description: String?` - Optional description
- `itemCount: Int` - Number of items
- `createdAt/updatedAt: String` - Timestamps

#### PlaylistItem
- `id: Int64` - Item ID in playlist
- `playlistId: Int64` - Parent playlist
- `mediaId: Int64` - Media reference
- `mediaType: MediaType` - movie/tvshow
- `position: Int` - Order position
- `title, year, posterPath, duration, etc.` - Display info

## User Interactions

### Navigation
1. **Access Playlists**: Navigate to "Playlists" tab in MainTabView
2. **Create Playlist**: Press the "+" button in toolbar
3. **View Playlist**: Select a playlist card to see contents
4. **Delete Playlist**: Long-press for context menu, select "Delete"

### Siri Remote Gestures
- **Swipe**: Navigate between playlists/items
- **Click**: Select playlist or play item
- **Long Press**: Open context menu (delete, remove)
- **Menu Button**: Go back

### Playback Controls
- **Play All**: Starts sequential playback from first item
- **Shuffle**: Randomizes order and starts playback
- **Click on Item**: Plays from that item
- **Up Next**: Shows after video ends with 10s countdown

## Focus Handling

The app uses `@FocusState` for proper remote navigation:

```swift
@FocusState private var focusedField: Field?

ForEach(playlists) { playlist in
    PlaylistCard(playlist: playlist)
        .focusable()
        .onTapGesture { /* handle selection */ }
}
```

## API Integration

### Endpoints Used

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/playlists | Get all user playlists |
| GET | /api/playlists/:id | Get playlist with items |
| POST | /api/playlists | Create new playlist |
| PUT | /api/playlists/:id | Update playlist |
| DELETE | /api/playlists/:id | Delete playlist |
| POST | /api/playlists/:id/items/:mediaId | Add item |
| DELETE | /api/playlists/:id/items/:mediaId | Remove item |
| PUT | /api/playlists/:id/reorder | Reorder items |

### Error Handling

```swift
do {
    let response = try await api.getPlaylists()
    playlists = response.items
} catch APIError.unauthorized {
    // Redirect to login
} catch {
    errorMessage = error.localizedDescription
}
```

## Testing Checklist

- [ ] Create new playlist with name and description
- [ ] Add movies/shows to playlist from library
- [ ] Reorder items with drag and drop
- [ ] Remove items from playlist
- [ ] Delete entire playlist
- [ ] Play All functionality
- [ ] Shuffle functionality
- [ ] Up Next overlay appears after video
- [ ] Auto-advance to next item
- [ ] Cancel auto-play
- [ ] Remote navigation works properly
- [ ] Context menu appears on long-press
- [ ] Empty state displays correctly

## Known Limitations

1. Reordering uses SwiftUI List's built-in move, which may not feel native on tvOS
2. Video player integration is placeholder - integrate with actual PlayerView
3. Poster images require network access

## Future Enhancements

- Add search to find playlists
- Share playlists between users
- Import/export playlists
- Smart playlists based on genres/years
