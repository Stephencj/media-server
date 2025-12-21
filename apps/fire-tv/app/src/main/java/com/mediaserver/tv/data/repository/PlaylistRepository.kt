package com.mediaserver.tv.data.repository

import com.mediaserver.tv.data.api.MediaServerApi
import com.mediaserver.tv.data.models.*
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class PlaylistRepository @Inject constructor(
    private val api: MediaServerApi
) {
    suspend fun getPlaylists(): Result<List<Playlist>> = runCatching {
        api.getPlaylists().items
    }

    suspend fun getPlaylist(id: Long): Result<PlaylistWithItems> = runCatching {
        api.getPlaylist(id)
    }

    suspend fun createPlaylist(name: String, description: String?): Result<Playlist> = runCatching {
        api.createPlaylist(CreatePlaylistRequest(name, description))
    }

    suspend fun updatePlaylist(id: Long, name: String, description: String?): Result<Unit> = runCatching {
        api.updatePlaylist(id, CreatePlaylistRequest(name, description))
    }

    suspend fun deletePlaylist(id: Long): Result<Unit> = runCatching {
        api.deletePlaylist(id)
    }

    suspend fun addToPlaylist(playlistId: Long, mediaId: Long, mediaType: String): Result<Unit> = runCatching {
        api.addToPlaylist(playlistId, mediaId, mediaType)
    }

    suspend fun removeFromPlaylist(playlistId: Long, mediaId: Long, mediaType: String): Result<Unit> = runCatching {
        api.removeFromPlaylist(playlistId, mediaId, mediaType)
    }

    suspend fun reorderPlaylist(playlistId: Long, itemIds: List<Long>): Result<Unit> = runCatching {
        api.reorderPlaylist(playlistId, ReorderPlaylistRequest(itemIds))
    }
}
