package com.mediaserver.tv.data.models

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

enum class MediaType(val value: String) {
    @SerialName("movie") MOVIE("movie"),
    @SerialName("tvshow") TV_SHOW("tvshow"),
    @SerialName("episode") EPISODE("episode")
}

@Serializable
data class Media(
    val id: Long,
    val title: String,
    @SerialName("original_title") val originalTitle: String? = null,
    val type: MediaType,
    val year: Int? = null,
    val overview: String? = null,
    @SerialName("poster_path") val posterPath: String? = null,
    @SerialName("backdrop_path") val backdropPath: String? = null,
    val rating: Double? = null,
    val runtime: Int? = null,
    val genres: String? = null,
    @SerialName("tmdb_id") val tmdbId: Int? = null,
    @SerialName("imdb_id") val imdbId: String? = null,
    @SerialName("season_count") val seasonCount: Int? = null,
    @SerialName("episode_count") val episodeCount: Int? = null,
    @SerialName("source_id") val sourceId: Long? = null,
    @SerialName("file_path") val filePath: String? = null,
    @SerialName("file_size") val fileSize: Long? = null,
    val duration: Int? = null,
    @SerialName("video_codec") val videoCodec: String? = null,
    @SerialName("audio_codec") val audioCodec: String? = null,
    val resolution: String? = null,
    @SerialName("audio_tracks") val audioTracks: String? = null,
    @SerialName("subtitle_tracks") val subtitleTracks: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null
) {
    val formattedDuration: String
        get() {
            val d = duration ?: return ""
            val hours = d / 3600
            val minutes = (d % 3600) / 60
            return if (hours > 0) "${hours}h ${minutes}m" else "${minutes}m"
        }

    val posterUrl: String?
        get() = posterPath?.let { "https://image.tmdb.org/t/p/w342$it" }

    val backdropUrl: String?
        get() = backdropPath?.let { "https://image.tmdb.org/t/p/w780$it" }
}

@Serializable
data class ContinueWatchingItem(
    val id: Long,
    @SerialName("user_id") val userId: Long,
    @SerialName("media_id") val mediaId: Long,
    @SerialName("media_type") val mediaType: String,
    val position: Int,
    val duration: Int,
    val completed: Boolean,
    @SerialName("updated_at") val updatedAt: String
)

@Serializable
data class PaginatedResponse<T>(
    val items: List<T>,
    val total: Int? = null,
    val limit: Int,
    val offset: Int
)

@Serializable
data class ItemsResponse<T>(
    val items: List<T>
)

@Serializable
data class ContinueWatchingResponse(
    val items: List<ContinueWatchingItem>
)

@Serializable
data class MessageResponse(
    val message: String
)

@Serializable
data class ScanResponse(
    val message: String,
    val status: String
)
