package com.mediaserver.tv.ui.detail

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.mediaserver.tv.data.models.WatchProgress
import com.mediaserver.tv.data.repository.MediaRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class DetailViewModel @Inject constructor(
    private val mediaRepository: MediaRepository
) : ViewModel() {

    private val _progress = MutableStateFlow<WatchProgress?>(null)
    val progress: StateFlow<WatchProgress?> = _progress.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    fun loadProgress(mediaId: Long, mediaType: String) {
        viewModelScope.launch {
            _isLoading.value = true
            val result = mediaRepository.getProgress(mediaId, mediaType)
            _progress.value = result.getOrNull()
            _isLoading.value = false
        }
    }
}
