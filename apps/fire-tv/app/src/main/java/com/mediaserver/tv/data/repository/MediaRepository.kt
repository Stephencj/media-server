package com.mediaserver.tv.data.repository

import com.mediaserver.tv.data.api.MediaServerApi
import com.mediaserver.tv.data.models.ContinueWatchingItem
import com.mediaserver.tv.data.models.Media
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class MediaRepository @Inject constructor(
    private val api: MediaServerApi
) {
    suspend fun getMovies(limit: Int = 50, offset: Int = 0): Result<List<Media>> = runCatching {
        api.getMovies(limit, offset).items
    }

    suspend fun getShows(limit: Int = 50, offset: Int = 0): Result<List<Media>> = runCatching {
        api.getShows(limit, offset).items
    }

    suspend fun getRecent(limit: Int = 20): Result<List<Media>> = runCatching {
        api.getRecent(limit).items
    }

    suspend fun getMedia(id: Long): Result<Media> = runCatching {
        api.getMedia(id)
    }

    suspend fun getContinueWatching(limit: Int = 10): Result<List<ContinueWatchingItem>> = runCatching {
        api.getContinueWatching(limit).items
    }

    suspend fun triggerScan(): Result<Unit> = runCatching {
        api.triggerScan()
    }
}
