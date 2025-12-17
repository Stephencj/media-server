package com.mediaserver.tv.ui.player

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.mediaserver.tv.data.repository.MediaRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class PlayerViewModel @Inject constructor(
    private val mediaRepository: MediaRepository
) : ViewModel() {

    fun saveProgress(
        mediaId: Long,
        position: Int,
        duration: Int,
        mediaType: String,
        completed: Boolean = false
    ) {
        viewModelScope.launch {
            mediaRepository.updateProgress(
                mediaId = mediaId,
                position = position,
                duration = duration,
                mediaType = mediaType,
                completed = completed
            )
        }
    }

    fun markAsCompleted(
        mediaId: Long,
        position: Int,
        duration: Int,
        mediaType: String
    ) {
        viewModelScope.launch {
            mediaRepository.updateProgress(
                mediaId = mediaId,
                position = position,
                duration = duration,
                mediaType = mediaType,
                completed = true
            )
        }
    }
}
