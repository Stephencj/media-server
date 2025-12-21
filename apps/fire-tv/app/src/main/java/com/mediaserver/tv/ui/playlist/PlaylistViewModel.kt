package com.mediaserver.tv.ui.playlist

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.mediaserver.tv.data.models.Playlist
import com.mediaserver.tv.data.models.PlaylistItem
import com.mediaserver.tv.data.repository.PlaylistRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PlaylistListUiState(
    val isLoading: Boolean = false,
    val playlists: List<Playlist> = emptyList(),
    val error: String? = null
)

@HiltViewModel
class PlaylistListViewModel @Inject constructor(
    private val playlistRepository: PlaylistRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(PlaylistListUiState())
    val uiState: StateFlow<PlaylistListUiState> = _uiState.asStateFlow()

    fun loadPlaylists() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            playlistRepository.getPlaylists()
                .onSuccess { playlists ->
                    _uiState.update { it.copy(isLoading = false, playlists = playlists, error = null) }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(isLoading = false, error = e.message) }
                }
        }
    }

    fun createPlaylist(name: String, description: String?) {
        viewModelScope.launch {
            playlistRepository.createPlaylist(name, description)
                .onSuccess { playlist ->
                    _uiState.update {
                        it.copy(playlists = listOf(playlist) + it.playlists)
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message) }
                }
        }
    }

    fun deletePlaylist(playlist: Playlist) {
        viewModelScope.launch {
            playlistRepository.deletePlaylist(playlist.id)
                .onSuccess {
                    _uiState.update {
                        it.copy(playlists = it.playlists.filter { p -> p.id != playlist.id })
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(error = e.message) }
                }
        }
    }
}

data class PlaylistDetailUiState(
    val isLoading: Boolean = false,
    val playlist: Playlist? = null,
    val items: List<PlaylistItem> = emptyList(),
    val error: String? = null,
    val isPlaying: Boolean = false,
    val currentIndex: Int = 0
)

@HiltViewModel
class PlaylistDetailViewModel @Inject constructor(
    private val playlistRepository: PlaylistRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(PlaylistDetailUiState())
    val uiState: StateFlow<PlaylistDetailUiState> = _uiState.asStateFlow()

    fun loadPlaylist(id: Long) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            playlistRepository.getPlaylist(id)
                .onSuccess { response ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            playlist = response.playlist,
                            items = response.items,
                            error = null
                        )
                    }
                }
                .onFailure { e ->
                    _uiState.update { it.copy(isLoading = false, error = e.message) }
                }
        }
    }

    fun removeItem(item: PlaylistItem) {
        val playlist = _uiState.value.playlist ?: return

        viewModelScope.launch {
            playlistRepository.removeFromPlaylist(
                playlist.id,
                item.mediaId,
                item.mediaType.value
            ).onSuccess {
                _uiState.update {
                    it.copy(items = it.items.filter { i -> i.id != item.id })
                }
            }
        }
    }

    fun moveItem(fromPosition: Int, toPosition: Int) {
        val items = _uiState.value.items.toMutableList()
        val item = items.removeAt(fromPosition)
        items.add(toPosition, item)
        _uiState.update { it.copy(items = items) }

        // Save order to server
        val playlist = _uiState.value.playlist ?: return
        viewModelScope.launch {
            playlistRepository.reorderPlaylist(playlist.id, items.map { it.id })
        }
    }

    fun playAll() {
        if (_uiState.value.items.isNotEmpty()) {
            _uiState.update { it.copy(isPlaying = true, currentIndex = 0) }
        }
    }

    fun shuffle() {
        if (_uiState.value.items.isNotEmpty()) {
            _uiState.update {
                it.copy(
                    items = it.items.shuffled(),
                    isPlaying = true,
                    currentIndex = 0
                )
            }
        }
    }

    fun playItem(index: Int) {
        _uiState.update { it.copy(isPlaying = true, currentIndex = index) }
    }

    fun playNext(): Boolean {
        val nextIndex = _uiState.value.currentIndex + 1
        return if (nextIndex < _uiState.value.items.size) {
            _uiState.update { it.copy(currentIndex = nextIndex) }
            true
        } else {
            _uiState.update { it.copy(isPlaying = false) }
            false
        }
    }

    fun stopPlaying() {
        _uiState.update { it.copy(isPlaying = false) }
    }

    val currentItem: PlaylistItem?
        get() {
            val state = _uiState.value
            return if (state.currentIndex in state.items.indices) {
                state.items[state.currentIndex]
            } else null
        }

    val nextItem: PlaylistItem?
        get() {
            val state = _uiState.value
            val nextIndex = state.currentIndex + 1
            return if (nextIndex in state.items.indices) {
                state.items[nextIndex]
            } else null
        }

    val hasNextItem: Boolean
        get() = _uiState.value.currentIndex < _uiState.value.items.size - 1
}
