package com.mediaserver.tv.ui.detail

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.mediaserver.tv.data.models.Extra
import com.mediaserver.tv.data.models.MediaType
import com.mediaserver.tv.data.models.WatchProgress
import com.mediaserver.tv.data.repository.MediaRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/** Group of extras of one kind, in display order. */
data class ExtraGroup(
    val title: String,
    val extras: List<Extra>
)

@HiltViewModel
class DetailViewModel @Inject constructor(
    private val mediaRepository: MediaRepository
) : ViewModel() {

    private val _progress = MutableStateFlow<WatchProgress?>(null)
    val progress: StateFlow<WatchProgress?> = _progress.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _commentaries = MutableStateFlow<List<Extra>>(emptyList())
    val commentaries: StateFlow<List<Extra>> = _commentaries.asStateFlow()

    private val _extraGroups = MutableStateFlow<List<ExtraGroup>>(emptyList())
    val extraGroups: StateFlow<List<ExtraGroup>> = _extraGroups.asStateFlow()

    companion object {
        private val groupOrder = listOf(
            "deleted_scene" to "Deleted Scenes",
            "featurette" to "Featurettes",
            "behind_the_scenes" to "Behind the Scenes",
            "interview" to "Interviews",
            "gag_reel" to "Gag Reels",
            "music_video" to "Music Videos",
            "other" to "More"
        )
    }

    fun loadProgress(mediaId: Long, mediaType: String) {
        viewModelScope.launch {
            _isLoading.value = true
            val result = mediaRepository.getProgress(mediaId, mediaType)
            _progress.value = result.getOrNull()
            _isLoading.value = false
        }
    }

    fun loadExtras(mediaId: Long, mediaType: MediaType) {
        viewModelScope.launch {
            val result = when (mediaType) {
                MediaType.MOVIE -> mediaRepository.getMovieExtras(mediaId)
                MediaType.TV_SHOW -> mediaRepository.getShowExtras(mediaId)
                else -> return@launch
            }
            val all = result.getOrNull() ?: return@launch

            // Separate commentaries
            _commentaries.value = all.filter { it.isCommentary }
                .sortedWith(compareBy({ it.seasonNumber ?: 0 }, { it.episodeNumber ?: 0 }))

            // Group other extras
            val knownCategories = groupOrder.map { it.first }.toSet()
            _extraGroups.value = groupOrder.mapNotNull { (category, title) ->
                val matching = all.filter { extra ->
                    !extra.isCommentary && (
                        extra.category == category ||
                        (category == "other" && extra.category !in knownCategories)
                    )
                }
                if (matching.isEmpty()) null else ExtraGroup(title, matching)
            }
        }
    }
}
