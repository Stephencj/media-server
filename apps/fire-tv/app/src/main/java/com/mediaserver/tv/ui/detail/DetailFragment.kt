package com.mediaserver.tv.ui.detail

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.core.content.ContextCompat
import androidx.fragment.app.viewModels
import androidx.leanback.app.DetailsSupportFragment
import androidx.leanback.app.DetailsSupportFragmentBackgroundController
import androidx.leanback.widget.*
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.Media
import com.mediaserver.tv.data.models.MediaType
import com.mediaserver.tv.ui.player.PlayerActivity
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch

@AndroidEntryPoint
class DetailFragment : DetailsSupportFragment() {

    private val viewModel: DetailViewModel by viewModels()
    private lateinit var detailsBackground: DetailsSupportFragmentBackgroundController
    private lateinit var presenterSelector: ClassPresenterSelector
    private lateinit var rowsAdapter: ArrayObjectAdapter

    private var media: Media? = null
    private var startPosition: Int = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        media = arguments?.getSerializable(ARG_MEDIA) as? Media
        startPosition = arguments?.getInt(ARG_START_POSITION, 0) ?: 0

        if (media == null) {
            requireActivity().finish()
            return
        }

        detailsBackground = DetailsSupportFragmentBackgroundController(this)

        setupAdapter()
        setupDetailsRow()
        setupEventListeners()
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        media?.let { viewModel.loadProgress(it.id, it.type.name.lowercase()) }

        observeViewModel()
    }

    private fun setupAdapter() {
        presenterSelector = ClassPresenterSelector()
        presenterSelector.addClassPresenter(DetailsOverviewRow::class.java, FullWidthDetailsOverviewRowPresenter(DetailsDescriptionPresenter()))
        presenterSelector.addClassPresenter(ListRow::class.java, ListRowPresenter())

        rowsAdapter = ArrayObjectAdapter(presenterSelector)
        adapter = rowsAdapter
    }

    private fun setupDetailsRow() {
        val media = media ?: return

        val detailsRow = DetailsOverviewRow(media)

        // Placeholder image
        detailsRow.imageDrawable = ContextCompat.getDrawable(requireContext(), R.drawable.default_poster)

        // Actions
        val actionAdapter = ArrayObjectAdapter()
        actionAdapter.add(Action(ACTION_PLAY, "Play", null))
        if (startPosition > 0) {
            actionAdapter.add(Action(ACTION_RESUME, "Resume", null))
        }
        actionAdapter.add(Action(ACTION_ADD_WATCHLIST, "Add to Watchlist", null))

        detailsRow.actionsAdapter = actionAdapter

        rowsAdapter.add(detailsRow)

        // Related items row (placeholder)
        val relatedHeader = HeaderItem(0, "More Like This")
        val relatedAdapter = ArrayObjectAdapter(CardPresenter())
        // TODO: Add related items
        rowsAdapter.add(ListRow(relatedHeader, relatedAdapter))
    }

    private fun setupEventListeners() {
        onItemViewClickedListener = OnItemViewClickedListener { _, item, _, _ ->
            when (item) {
                is Action -> handleAction(item.id)
                is Media -> {
                    val intent = Intent(requireContext(), DetailActivity::class.java).apply {
                        putExtra(DetailActivity.EXTRA_MEDIA, item)
                    }
                    startActivity(intent)
                }
            }
        }
    }

    private fun handleAction(actionId: Long) {
        val media = media ?: return

        when (actionId) {
            ACTION_PLAY -> playMedia(0)
            ACTION_RESUME -> playMedia(viewModel.progress.value?.position ?: startPosition)
            ACTION_ADD_WATCHLIST -> {
                // TODO: Add to watchlist
            }
        }
    }

    private fun playMedia(position: Int) {
        val media = media ?: return

        val intent = Intent(requireContext(), PlayerActivity::class.java).apply {
            putExtra(PlayerActivity.EXTRA_MEDIA, media)
            putExtra(PlayerActivity.EXTRA_START_POSITION, position)
        }
        startActivity(intent)
    }

    private fun observeViewModel() {
        viewLifecycleOwner.lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                viewModel.progress.collect { progress ->
                    if (progress != null && progress.position > 0) {
                        updateResumeAction(progress.position)
                    }
                }
            }
        }
    }

    private fun updateResumeAction(position: Int) {
        val detailsRow = rowsAdapter.get(0) as? DetailsOverviewRow ?: return
        val actionAdapter = detailsRow.actionsAdapter as? ArrayObjectAdapter ?: return

        // Check if resume action already exists
        for (i in 0 until actionAdapter.size()) {
            val action = actionAdapter.get(i) as? Action
            if (action?.id == ACTION_RESUME) {
                return // Already has resume
            }
        }

        // Add resume action after play
        actionAdapter.add(1, Action(ACTION_RESUME, "Resume", formatTime(position)))
    }

    private fun formatTime(seconds: Int): String {
        val hours = seconds / 3600
        val minutes = (seconds % 3600) / 60
        return if (hours > 0) {
            "${hours}h ${minutes}m"
        } else {
            "${minutes}m"
        }
    }

    override fun onResume() {
        super.onResume()
        media?.let { viewModel.loadProgress(it.id, it.type.name.lowercase()) }
    }

    companion object {
        private const val ARG_MEDIA = "arg_media"
        private const val ARG_START_POSITION = "arg_start_position"

        private const val ACTION_PLAY = 1L
        private const val ACTION_RESUME = 2L
        private const val ACTION_ADD_WATCHLIST = 3L

        fun newInstance(media: Media, startPosition: Int = 0): DetailFragment {
            return DetailFragment().apply {
                arguments = Bundle().apply {
                    putSerializable(ARG_MEDIA, media)
                    putInt(ARG_START_POSITION, startPosition)
                }
            }
        }
    }
}

class DetailsDescriptionPresenter : AbstractDetailsDescriptionPresenter() {

    override fun onBindDescription(viewHolder: ViewHolder, item: Any) {
        val media = item as Media

        viewHolder.title.text = media.title
        viewHolder.subtitle.text = buildSubtitle(media)
        viewHolder.body.text = media.overview ?: ""
    }

    private fun buildSubtitle(media: Media): String {
        val parts = mutableListOf<String>()

        media.year?.let { parts.add(it.toString()) }
        media.rating?.let { parts.add("★ %.1f".format(it)) }

        if (media.type == MediaType.TV_SHOW) {
            media.seasonCount?.let { parts.add("$it Seasons") }
        } else {
            media.formattedDuration.takeIf { it.isNotEmpty() }?.let { parts.add(it) }
        }

        media.resolution?.let { parts.add(it) }
        media.videoCodec?.uppercase()?.let { parts.add(it) }

        return parts.joinToString(" • ")
    }
}

// Reusing CardPresenter from browse package
private class CardPresenter : Presenter() {
    override fun onCreateViewHolder(parent: android.view.ViewGroup): ViewHolder {
        val cardView = androidx.leanback.widget.ImageCardView(parent.context).apply {
            isFocusable = true
            isFocusableInTouchMode = true
            setMainImageDimensions(150, 225)
        }
        return ViewHolder(cardView)
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any) {
        val media = item as Media
        val cardView = viewHolder.view as androidx.leanback.widget.ImageCardView
        cardView.titleText = media.title
        cardView.contentText = media.year?.toString() ?: ""
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {}
}
