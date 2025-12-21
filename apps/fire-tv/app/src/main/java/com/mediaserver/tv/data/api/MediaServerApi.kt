package com.mediaserver.tv.data.api

import com.mediaserver.tv.data.models.*
import retrofit2.http.*

interface MediaServerApi {

    // Auth
    @POST("api/auth/login")
    suspend fun login(@Body request: LoginRequest): AuthResponse

    @POST("api/auth/register")
    suspend fun register(@Body request: RegisterRequest): AuthResponse

    // Library
    @GET("api/library/movies")
    suspend fun getMovies(
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0
    ): PaginatedResponse<Media>

    @GET("api/library/shows")
    suspend fun getShows(
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0
    ): PaginatedResponse<Media>

    @GET("api/library/recent")
    suspend fun getRecent(@Query("limit") limit: Int = 20): ItemsResponse<Media>

    @POST("api/library/scan")
    suspend fun triggerScan(): ScanResponse

    // Media
    @GET("api/media/{id}")
    suspend fun getMedia(@Path("id") id: Long): Media

    // Progress
    @GET("api/progress/{mediaId}")
    suspend fun getProgress(
        @Path("mediaId") mediaId: Long,
        @Query("type") type: String = "movie"
    ): WatchProgress

    @POST("api/progress/{mediaId}")
    suspend fun updateProgress(
        @Path("mediaId") mediaId: Long,
        @Body request: UpdateProgressRequest
    ): WatchProgress

    @GET("api/continue-watching")
    suspend fun getContinueWatching(@Query("limit") limit: Int = 10): ContinueWatchingResponse

    // Playlists
    @GET("api/playlists")
    suspend fun getPlaylists(): PlaylistsResponse

    @GET("api/playlists/{playlistId}")
    suspend fun getPlaylist(@Path("playlistId") playlistId: Long): PlaylistWithItems

    @POST("api/playlists")
    suspend fun createPlaylist(@Body request: CreatePlaylistRequest): Playlist

    @PUT("api/playlists/{playlistId}")
    suspend fun updatePlaylist(
        @Path("playlistId") playlistId: Long,
        @Body request: CreatePlaylistRequest
    ): MessageResponse

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

@kotlinx.serialization.Serializable
data class LoginRequest(
    val username: String,
    val password: String
)

@kotlinx.serialization.Serializable
data class RegisterRequest(
    val username: String,
    val email: String,
    val password: String
)

@kotlinx.serialization.Serializable
data class AuthResponse(
    val token: String,
    @kotlinx.serialization.SerialName("expires_at") val expiresAt: Long,
    val user: User
)

@kotlinx.serialization.Serializable
data class User(
    val id: Long,
    val username: String,
    val email: String
)

@kotlinx.serialization.Serializable
data class WatchProgress(
    val id: Long,
    @kotlinx.serialization.SerialName("user_id") val userId: Long,
    @kotlinx.serialization.SerialName("media_id") val mediaId: Long,
    @kotlinx.serialization.SerialName("media_type") val mediaType: String,
    val position: Int,
    val duration: Int,
    val completed: Boolean,
    @kotlinx.serialization.SerialName("updated_at") val updatedAt: String
)

@kotlinx.serialization.Serializable
data class UpdateProgressRequest(
    val position: Int,
    val duration: Int,
    @kotlinx.serialization.SerialName("media_type") val mediaType: String,
    val completed: Boolean = false
)
