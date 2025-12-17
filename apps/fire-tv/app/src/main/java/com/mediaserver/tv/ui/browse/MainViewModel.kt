package com.mediaserver.tv.ui.browse

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.mediaserver.tv.data.models.ContinueWatchingItem
import com.mediaserver.tv.data.models.Media
import com.mediaserver.tv.data.models.MediaType
import com.mediaserver.tv.data.repository.MediaRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class MainUiState(
    val isLoading: Boolean = false,
    val continueWatching: List<ContinueWatchingItem> = emptyList(),
    val recentlyAdded: List<Media> = emptyList(),
    val movies: List<Media> = emptyList(),
    val tvShows: List<Media> = emptyList(),
    val error: String? = null
)

@HiltViewModel
class MainViewModel @Inject constructor(
    private val mediaRepository: MediaRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(MainUiState())
    val uiState: StateFlow<MainUiState> = _uiState.asStateFlow()

    fun loadContent() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            val continueWatchingDeferred = async { mediaRepository.getContinueWatching(10) }
            val recentDeferred = async { mediaRepository.getRecent(15) }
            val moviesDeferred = async { mediaRepository.getMovies(15, 0) }
            val showsDeferred = async { mediaRepository.getShows(15, 0) }

            val continueWatching = continueWatchingDeferred.await().getOrDefault(emptyList())
            val recent = recentDeferred.await().getOrDefault(emptyList())
            val movies = moviesDeferred.await().getOrDefault(emptyList())
            val shows = showsDeferred.await().getOrDefault(emptyList())

            _uiState.update {
                it.copy(
                    isLoading = false,
                    continueWatching = continueWatching,
                    recentlyAdded = recent,
                    movies = movies.filter { m -> m.type == MediaType.MOVIE },
                    tvShows = shows.filter { m -> m.type == MediaType.TV_SHOW }
                )
            }
        }
    }

    fun triggerScan() {
        viewModelScope.launch {
            mediaRepository.triggerScan()
        }
    }
}
