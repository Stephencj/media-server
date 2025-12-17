package com.mediaserver.tv.ui.browse

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.core.content.ContextCompat
import androidx.fragment.app.viewModels
import androidx.leanback.app.BrowseSupportFragment
import androidx.leanback.widget.*
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.ContinueWatchingItem
import com.mediaserver.tv.data.models.Media
import com.mediaserver.tv.ui.detail.DetailActivity
import com.mediaserver.tv.ui.settings.SettingsActivity
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch

@AndroidEntryPoint
class MainFragment : BrowseSupportFragment() {

    private val viewModel: MainViewModel by viewModels()
    private lateinit var rowsAdapter: ArrayObjectAdapter

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setupUI()
        setupEventListeners()
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        observeViewModel()
        viewModel.loadContent()
    }

    private fun setupUI() {
        title = "Media Server"
        headersState = HEADERS_ENABLED
        isHeadersTransitionOnBackEnabled = true

        brandColor = ContextCompat.getColor(requireContext(), R.color.brand_color)
        searchAffordanceColor = ContextCompat.getColor(requireContext(), R.color.search_color)

        // Create rows adapter
        rowsAdapter = ArrayObjectAdapter(ListRowPresenter())
        adapter = rowsAdapter
    }

    private fun setupEventListeners() {
        // Item click
        onItemViewClickedListener = OnItemViewClickedListener { _, item, _, _ ->
            when (item) {
                is Media -> {
                    val intent = Intent(requireContext(), DetailActivity::class.java).apply {
                        putExtra(DetailActivity.EXTRA_MEDIA, item)
                    }
                    startActivity(intent)
                }
                is ContinueWatchingItem -> {
                    val intent = Intent(requireContext(), DetailActivity::class.java).apply {
                        putExtra(DetailActivity.EXTRA_MEDIA, item.media)
                        putExtra(DetailActivity.EXTRA_START_POSITION, item.progress.position)
                    }
                    startActivity(intent)
                }
            }
        }

        // Item selected (for preview)
        onItemViewSelectedListener = OnItemViewSelectedListener { _, item, _, _ ->
            // Could update background image here
        }

        // Search click
        setOnSearchClickedListener {
            // TODO: Implement search
        }
    }

    private fun observeViewModel() {
        viewLifecycleOwner.lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                launch {
                    viewModel.uiState.collect { state ->
                        updateUI(state)
                    }
                }
            }
        }
    }

    private fun updateUI(state: MainUiState) {
        rowsAdapter.clear()

        val cardPresenter = CardPresenter()

        // Continue Watching row
        if (state.continueWatching.isNotEmpty()) {
            val continueAdapter = ArrayObjectAdapter(ContinueWatchingPresenter())
            state.continueWatching.forEach { continueAdapter.add(it) }
            val continueHeader = HeaderItem(0, "Continue Watching")
            rowsAdapter.add(ListRow(continueHeader, continueAdapter))
        }

        // Recently Added row
        if (state.recentlyAdded.isNotEmpty()) {
            val recentAdapter = ArrayObjectAdapter(cardPresenter)
            state.recentlyAdded.forEach { recentAdapter.add(it) }
            val recentHeader = HeaderItem(1, "Recently Added")
            rowsAdapter.add(ListRow(recentHeader, recentAdapter))
        }

        // Movies row
        if (state.movies.isNotEmpty()) {
            val moviesAdapter = ArrayObjectAdapter(cardPresenter)
            state.movies.forEach { moviesAdapter.add(it) }
            val moviesHeader = HeaderItem(2, "Movies")
            rowsAdapter.add(ListRow(moviesHeader, moviesAdapter))
        }

        // TV Shows row
        if (state.tvShows.isNotEmpty()) {
            val showsAdapter = ArrayObjectAdapter(cardPresenter)
            state.tvShows.forEach { showsAdapter.add(it) }
            val showsHeader = HeaderItem(3, "TV Shows")
            rowsAdapter.add(ListRow(showsHeader, showsAdapter))
        }

        // Settings row
        val settingsAdapter = ArrayObjectAdapter(GridItemPresenter())
        settingsAdapter.add(SettingsItem("Settings", R.drawable.ic_settings))
        settingsAdapter.add(SettingsItem("Scan Library", R.drawable.ic_refresh))
        val settingsHeader = HeaderItem(4, "Settings")
        rowsAdapter.add(ListRow(settingsHeader, settingsAdapter))
    }

    override fun onResume() {
        super.onResume()
        viewModel.loadContent()
    }
}

data class SettingsItem(val title: String, val iconRes: Int)
