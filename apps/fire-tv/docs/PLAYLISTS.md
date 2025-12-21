# Fire TV Playlist Feature Documentation

## Overview

The playlist feature allows users to create, manage, and play custom playlists of movies and TV shows on Fire TV using the remote control D-pad navigation.

## Architecture

### Files Structure

```
app/src/main/java/com/mediaserver/tv/
├── data/
│   ├── models/
│   │   ├── Media.kt           # Media and response models
│   │   └── Playlist.kt        # Playlist models
│   ├── api/
│   │   └── MediaServerApi.kt  # Retrofit API interface
│   └── repository/
│       ├── AuthRepository.kt
│       ├── MediaRepository.kt
│       └── PlaylistRepository.kt
└── ui/
    └── playlist/
        ├── PlaylistViewModel.kt      # List and Detail ViewModels
        ├── PlaylistFragment.kt       # Leanback grid of playlists
        ├── PlaylistCardPresenter.kt  # Card rendering
        ├── PlaylistDetailActivity.kt # Detail screen host
        └── PlaylistDetailFragment.kt # Detail with items
```

### Models

#### Playlist
```kotlin
@Serializable
data class Playlist(
    val id: Long,
    @SerialName("user_id") val userId: Long,
    val name: String,
    val description: String? = null,
    @SerialName("item_count") val itemCount: Int = 0,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String
)
```

#### PlaylistItem
```kotlin
@Serializable
data class PlaylistItem(
    val id: Long,
    @SerialName("playlist_id") val playlistId: Long,
    @SerialName("media_id") val mediaId: Long,
    @SerialName("media_type") val mediaType: MediaType,
    val position: Int,
    // ... display fields
)
```

## Leanback Integration

### Fragment Types

1. **PlaylistFragment** - `VerticalGridSupportFragment`
   - Displays grid of playlists
   - Uses `PlaylistCardPresenter` for cards

2. **PlaylistDetailFragment** - `BrowseSupportFragment`
   - Shows playlist actions and items
   - Uses `ListRow` for categories

### Presenters

```kotlin
class PlaylistCardPresenter : Presenter() {
    override fun onCreateViewHolder(parent: ViewGroup): ViewHolder {
        return ViewHolder(ImageCardView(parent.context))
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any?) {
        val playlist = item as? Playlist ?: return
        val cardView = viewHolder.view as ImageCardView
        cardView.titleText = playlist.name
        cardView.contentText = "${playlist.itemCount} items"
    }
}
```

## Remote Control Mapping

| Button | Action |
|--------|--------|
| D-pad Up/Down/Left/Right | Navigate between items |
| Center/Select | Select item, start playback |
| Back | Go back to previous screen |
| Play/Pause | Toggle playback |
| Menu | Context menu (if implemented) |

## Dependency Injection

Using Hilt for dependency injection:

```kotlin
@HiltViewModel
class PlaylistListViewModel @Inject constructor(
    private val playlistRepository: PlaylistRepository
) : ViewModel() {
    // ...
}

@Singleton
class PlaylistRepository @Inject constructor(
    private val api: MediaServerApi
) {
    // ...
}
```

### Module Setup (AppModule.kt)

Add repository providers:

```kotlin
@Provides
@Singleton
fun providePlaylistRepository(api: MediaServerApi): PlaylistRepository {
    return PlaylistRepository(api)
}
```

## API Integration

### Retrofit Interface

```kotlin
interface MediaServerApi {
    @GET("api/playlists")
    suspend fun getPlaylists(): PlaylistsResponse

    @GET("api/playlists/{playlistId}")
    suspend fun getPlaylist(@Path("playlistId") playlistId: Long): PlaylistWithItems

    @POST("api/playlists")
    suspend fun createPlaylist(@Body request: CreatePlaylistRequest): Playlist

    @DELETE("api/playlists/{playlistId}")
    suspend fun deletePlaylist(@Path("playlistId") playlistId: Long): MessageResponse

    @POST("api/playlists/{playlistId}/items/{mediaId}")
    suspend fun addToPlaylist(
        @Path("playlistId") playlistId: Long,
        @Path("mediaId") mediaId: Long,
        @Query("type") type: String
    ): MessageResponse

    @DELETE("api/playlists/{playlistId}/items/{mediaId}")
    suspend fun removeFromPlaylist(
        @Path("playlistId") playlistId: Long,
        @Path("mediaId") mediaId: Long,
        @Query("type") type: String
    ): MessageResponse

    @PUT("api/playlists/{playlistId}/reorder")
    suspend fun reorderPlaylist(
        @Path("playlistId") playlistId: Long,
        @Body request: ReorderPlaylistRequest
    ): MessageResponse
}
```

## Sequential Playback

The `PlaylistDetailViewModel` manages playback state:

```kotlin
fun playAll() {
    if (items.isNotEmpty()) {
        _uiState.update { it.copy(isPlaying = true, currentIndex = 0) }
    }
}

fun playNext(): Boolean {
    val nextIndex = currentIndex + 1
    return if (nextIndex < items.size) {
        _uiState.update { it.copy(currentIndex = nextIndex) }
        true
    } else {
        _uiState.update { it.copy(isPlaying = false) }
        false
    }
}
```

### ExoPlayer Integration

In PlayerActivity, listen for playback completion:

```kotlin
player.addListener(object : Player.Listener {
    override fun onPlaybackStateChanged(state: Int) {
        if (state == Player.STATE_ENDED) {
            // Show "Up Next" overlay
            showUpNextOverlay()
        }
    }
})
```

## Testing Checklist

- [ ] Playlists grid displays correctly
- [ ] D-pad navigation works between cards
- [ ] Select playlist opens detail view
- [ ] Detail shows actions row (Play All, Shuffle)
- [ ] Detail shows items row with media cards
- [ ] Play All starts playback from first item
- [ ] Shuffle randomizes and starts playback
- [ ] Selecting item starts playback from that point
- [ ] Video player launches with correct media
- [ ] Back button returns to playlist
- [ ] Empty playlist shows appropriate message

## Required Resources

Add these drawable resources:

```
res/drawable/
├── ic_playlist.xml      # Playlist icon
├── ic_play.xml          # Play icon
└── ic_shuffle.xml       # Shuffle icon
```

## Layout Files

Add activity declarations to AndroidManifest.xml:

```xml
<activity
    android:name=".ui.playlist.PlaylistDetailActivity"
    android:theme="@style/Theme.Leanback" />
```

## Known Limitations

1. Reordering items via remote is not implemented (would need custom gesture handling)
2. Long-press context menu not yet implemented
3. Video player integration is referenced but should use existing PlayerActivity

## Future Enhancements

- Add long-press to delete playlist
- Implement item reordering with D-pad
- Add create playlist dialog
- Support for offline playlists
- Continue watching integration
