package com.mediaserver.tv.ui.playlist

import android.os.Bundle
import android.view.View
import androidx.fragment.app.viewModels
import androidx.leanback.app.VerticalGridSupportFragment
import androidx.leanback.widget.*
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.mediaserver.tv.data.models.Playlist
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch

@AndroidEntryPoint
class PlaylistFragment : VerticalGridSupportFragment(), OnItemViewClickedListener {

    private val viewModel: PlaylistListViewModel by viewModels()
    private lateinit var adapter: ArrayObjectAdapter

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        title = "Playlists"
        setupAdapter()
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        onItemViewClickedListener = this

        viewLifecycleOwner.lifecycleScope.launch {
            viewLifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.uiState.collect { state ->
                    adapter.clear()
                    state.playlists.forEach { playlist ->
                        adapter.add(playlist)
                    }
                }
            }
        }

        viewModel.loadPlaylists()
    }

    private fun setupAdapter() {
        val presenter = VerticalGridPresenter()
        presenter.numberOfColumns = 4
        gridPresenter = presenter

        adapter = ArrayObjectAdapter(PlaylistCardPresenter())
        setAdapter(adapter)
    }

    override fun onItemClicked(
        itemViewHolder: Presenter.ViewHolder?,
        item: Any?,
        rowViewHolder: RowPresenter.ViewHolder?,
        row: Row?
    ) {
        if (item is Playlist) {
            // Navigate to playlist detail
            val intent = PlaylistDetailActivity.createIntent(requireContext(), item.id, item.name)
            startActivity(intent)
        }
    }

    companion object {
        fun newInstance() = PlaylistFragment()
    }
}
