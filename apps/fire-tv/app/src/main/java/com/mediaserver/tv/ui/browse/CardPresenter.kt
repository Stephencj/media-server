package com.mediaserver.tv.ui.browse

import android.graphics.drawable.Drawable
import android.view.ViewGroup
import androidx.core.content.ContextCompat
import androidx.leanback.widget.ImageCardView
import androidx.leanback.widget.Presenter
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.Media
import com.mediaserver.tv.data.models.MediaType

class CardPresenter : Presenter() {

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
        val media = item as Media
        val cardView = viewHolder.view as ImageCardView

        cardView.titleText = media.title
        cardView.contentText = buildContentText(media)
        cardView.mainImage = defaultCardImage

        // Set badge for TV shows
        if (media.type == MediaType.TV_SHOW) {
            cardView.badgeImage = ContextCompat.getDrawable(cardView.context, R.drawable.ic_tv_badge)
        } else {
            cardView.badgeImage = null
        }

        // TODO: Load actual poster image using Coil
        // if (media.posterPath != null) {
        //     val imageUrl = "${serverUrl}${media.posterPath}"
        //     cardView.mainImageView.load(imageUrl)
        // }
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {
        val cardView = viewHolder.view as ImageCardView
        cardView.badgeImage = null
        cardView.mainImage = null
    }

    private fun buildContentText(media: Media): String {
        val parts = mutableListOf<String>()

        media.year?.let { parts.add(it.toString()) }
        media.rating?.let { parts.add("★ %.1f".format(it)) }

        if (media.type == MediaType.TV_SHOW) {
            media.seasonCount?.let { parts.add("$it Seasons") }
        } else {
            media.formattedDuration.takeIf { it.isNotEmpty() }?.let { parts.add(it) }
        }

        return parts.joinToString(" • ")
    }

    companion object {
        private const val CARD_WIDTH = 200
        private const val CARD_HEIGHT = 300
    }
}
