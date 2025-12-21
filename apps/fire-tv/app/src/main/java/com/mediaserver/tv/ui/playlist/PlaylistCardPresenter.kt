package com.mediaserver.tv.ui.playlist

import android.view.LayoutInflater
import android.view.ViewGroup
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.leanback.widget.ImageCardView
import androidx.leanback.widget.Presenter
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.Playlist

class PlaylistCardPresenter : Presenter() {

    override fun onCreateViewHolder(parent: ViewGroup): ViewHolder {
        val cardView = ImageCardView(parent.context).apply {
            isFocusable = true
            isFocusableInTouchMode = true
            setBackgroundColor(ContextCompat.getColor(context, R.color.card_background))
        }
        return ViewHolder(cardView)
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any?) {
        val playlist = item as? Playlist ?: return
        val cardView = viewHolder.view as ImageCardView

        cardView.titleText = playlist.name
        cardView.contentText = "${playlist.itemCount} item${if (playlist.itemCount != 1) "s" else ""}"

        cardView.setMainImageDimensions(CARD_WIDTH, CARD_HEIGHT)

        // Set a default playlist icon
        cardView.mainImageView?.setImageResource(R.drawable.ic_playlist)
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {
        val cardView = viewHolder.view as ImageCardView
        cardView.badgeImage = null
        cardView.mainImage = null
    }

    companion object {
        private const val CARD_WIDTH = 250
        private const val CARD_HEIGHT = 200
    }
}
