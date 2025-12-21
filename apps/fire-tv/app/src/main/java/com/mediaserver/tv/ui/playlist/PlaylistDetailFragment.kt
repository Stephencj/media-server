package com.mediaserver.tv.ui.playlist

import android.os.Bundle
import android.view.View
import androidx.core.content.ContextCompat
import androidx.fragment.app.viewModels
import androidx.leanback.app.BrowseSupportFragment
import androidx.leanback.widget.*
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.bumptech.glide.Glide
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.PlaylistItem
import com.mediaserver.tv.ui.player.PlayerActivity
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch

@AndroidEntryPoint
class PlaylistDetailFragment : BrowseSupportFragment(), OnItemViewClickedListener {

    private val viewModel: PlaylistDetailViewModel by viewModels()
    private var playlistId: Long = -1
    private var playlistName: String = "Playlist"

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        playlistId = arguments?.getLong(ARG_PLAYLIST_ID, -1) ?: -1
        playlistName = arguments?.getString(ARG_PLAYLIST_NAME) ?: "Playlist"

        title = playlistName
        headersState = HEADERS_DISABLED

        setupUI()
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        onItemViewClickedListener = this

        viewLifecycleOwner.lifecycleScope.launch {
            viewLifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.uiState.collect { state ->
                    updateUI(state)
                }
            }
        }

        if (playlistId > 0) {
            viewModel.loadPlaylist(playlistId)
        }
    }

    private fun setupUI() {
        brandColor = ContextCompat.getColor(requireContext(), R.color.fastlane_background)
        searchAffordanceColor = ContextCompat.getColor(requireContext(), R.color.search_opaque)
    }

    private fun updateUI(state: PlaylistDetailUiState) {
        val rowsAdapter = ArrayObjectAdapter(ListRowPresenter())

        // Actions row
        val actionsAdapter = ArrayObjectAdapter(ActionPresenter())
        actionsAdapter.add(PlaylistAction.PLAY_ALL)
        actionsAdapter.add(PlaylistAction.SHUFFLE)

        val actionsHeader = HeaderItem(0, "Actions")
        rowsAdapter.add(ListRow(actionsHeader, actionsAdapter))

        // Items row
        if (state.items.isNotEmpty()) {
            val itemsAdapter = ArrayObjectAdapter(PlaylistItemPresenter())
            state.items.forEach { item ->
                itemsAdapter.add(item)
            }

            val itemsHeader = HeaderItem(1, "Items (${state.items.size})")
            rowsAdapter.add(ListRow(itemsHeader, itemsAdapter))
        }

        adapter = rowsAdapter
    }

    override fun onItemClicked(
        itemViewHolder: Presenter.ViewHolder?,
        item: Any?,
        rowViewHolder: RowPresenter.ViewHolder?,
        row: Row?
    ) {
        when (item) {
            is PlaylistAction -> {
                when (item) {
                    PlaylistAction.PLAY_ALL -> {
                        viewModel.playAll()
                        startPlayback()
                    }
                    PlaylistAction.SHUFFLE -> {
                        viewModel.shuffle()
                        startPlayback()
                    }
                }
            }
            is PlaylistItem -> {
                val index = viewModel.uiState.value.items.indexOf(item)
                if (index >= 0) {
                    viewModel.playItem(index)
                    startPlayback()
                }
            }
        }
    }

    private fun startPlayback() {
        viewModel.currentItem?.let { item ->
            val intent = PlayerActivity.createIntent(
                requireContext(),
                item.mediaId,
                item.title,
                item.mediaType.value
            )
            startActivity(intent)
        }
    }

    companion object {
        private const val ARG_PLAYLIST_ID = "playlist_id"
        private const val ARG_PLAYLIST_NAME = "playlist_name"

        fun newInstance(playlistId: Long, playlistName: String) = PlaylistDetailFragment().apply {
            arguments = Bundle().apply {
                putLong(ARG_PLAYLIST_ID, playlistId)
                putString(ARG_PLAYLIST_NAME, playlistName)
            }
        }
    }
}

enum class PlaylistAction(val title: String, val icon: Int) {
    PLAY_ALL("Play All", R.drawable.ic_play),
    SHUFFLE("Shuffle", R.drawable.ic_shuffle)
}

class ActionPresenter : Presenter() {
    override fun onCreateViewHolder(parent: android.view.ViewGroup): ViewHolder {
        val view = ImageCardView(parent.context).apply {
            isFocusable = true
            isFocusableInTouchMode = true
        }
        return ViewHolder(view)
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any?) {
        val action = item as? PlaylistAction ?: return
        val cardView = viewHolder.view as ImageCardView

        cardView.titleText = action.title
        cardView.setMainImageDimensions(150, 100)
        cardView.mainImageView?.setImageResource(action.icon)
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {}
}

class PlaylistItemPresenter : Presenter() {
    override fun onCreateViewHolder(parent: android.view.ViewGroup): ViewHolder {
        val cardView = ImageCardView(parent.context).apply {
            isFocusable = true
            isFocusableInTouchMode = true
        }
        return ViewHolder(cardView)
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any?) {
        val playlistItem = item as? PlaylistItem ?: return
        val cardView = viewHolder.view as ImageCardView

        cardView.titleText = playlistItem.title
        cardView.contentText = buildString {
            playlistItem.year?.let { append("$it") }
            if (playlistItem.formattedDuration.isNotEmpty()) {
                if (isNotEmpty()) append(" • ")
                append(playlistItem.formattedDuration)
            }
        }

        cardView.setMainImageDimensions(CARD_WIDTH, CARD_HEIGHT)

        playlistItem.posterUrl?.let { url ->
            Glide.with(cardView.context)
                .load(url)
                .into(cardView.mainImageView!!)
        }
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {
        val cardView = viewHolder.view as ImageCardView
        cardView.mainImage = null
    }

    companion object {
        private const val CARD_WIDTH = 176
        private const val CARD_HEIGHT = 264
    }
}
