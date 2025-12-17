package com.mediaserver.tv.ui.browse

import android.graphics.drawable.Drawable
import android.view.ViewGroup
import android.widget.ProgressBar
import androidx.core.content.ContextCompat
import androidx.leanback.widget.ImageCardView
import androidx.leanback.widget.Presenter
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.ContinueWatchingItem

class ContinueWatchingPresenter : Presenter() {

    private var defaultCardImage: Drawable? = null

    override fun onCreateViewHolder(parent: ViewGroup): ViewHolder {
        defaultCardImage = ContextCompat.getDrawable(parent.context, R.drawable.default_poster)

        val cardView = ImageCardView(parent.context).apply {
            isFocusable = true
            isFocusableInTouchMode = true
            setMainImageDimensions(CARD_WIDTH, CARD_HEIGHT)
        }

        return ViewHolder(cardView)
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any) {
        val continueItem = item as ContinueWatchingItem
        val media = continueItem.media
        val progress = continueItem.progress
        val cardView = viewHolder.view as ImageCardView

        cardView.titleText = media.title
        cardView.contentText = "${progress.formattedRemaining} remaining"
        cardView.mainImage = defaultCardImage

        // The info area shows progress - we'll use content text for now
        // In a more complete implementation, you'd add a custom progress bar

        // TODO: Load actual poster
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {
        val cardView = viewHolder.view as ImageCardView
        cardView.badgeImage = null
        cardView.mainImage = null
    }

    companion object {
        private const val CARD_WIDTH = 200
        private const val CARD_HEIGHT = 300
    }
}
