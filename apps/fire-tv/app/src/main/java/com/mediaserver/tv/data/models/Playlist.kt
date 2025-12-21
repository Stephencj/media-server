package com.mediaserver.tv.data.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

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

@Serializable
data class PlaylistItem(
    val id: Long,
    @SerialName("playlist_id") val playlistId: Long,
    @SerialName("media_id") val mediaId: Long,
    @SerialName("media_type") val mediaType: MediaType,
    val position: Int,
    @SerialName("added_at") val addedAt: String,
    val title: String,
    val year: Int? = null,
    @SerialName("poster_path") val posterPath: String? = null,
    val duration: Int? = null,
    val overview: String? = null,
    val rating: Double? = null,
    val resolution: String? = null
) {
    val formattedDuration: String
        get() {
            val d = duration ?: return ""
            val hours = d / 3600
            val minutes = (d % 3600) / 60
            return if (hours > 0) "${hours}h ${minutes}m" else "${minutes}m"
        }

    val posterUrl: String?
        get() = posterPath?.let { "https://image.tmdb.org/t/p/w185$it" }
}

@Serializable
data class PlaylistsResponse(
    val items: List<Playlist>
)

@Serializable
data class PlaylistWithItems(
    val playlist: Playlist,
    val items: List<PlaylistItem>
)

@Serializable
data class CreatePlaylistRequest(
    val name: String,
    val description: String? = null
)

@Serializable
data class ReorderPlaylistRequest(
    @SerialName("item_ids") val itemIds: List<Long>
)
