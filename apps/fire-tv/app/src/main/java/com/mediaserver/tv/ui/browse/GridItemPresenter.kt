package com.mediaserver.tv.ui.browse

import android.graphics.Color
import android.view.Gravity
import android.view.ViewGroup
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.leanback.widget.Presenter
import com.mediaserver.tv.R

class GridItemPresenter : Presenter() {

    override fun onCreateViewHolder(parent: ViewGroup): ViewHolder {
        val view = TextView(parent.context).apply {
            layoutParams = ViewGroup.LayoutParams(GRID_ITEM_WIDTH, GRID_ITEM_HEIGHT)
            isFocusable = true
            isFocusableInTouchMode = true
            setBackgroundColor(ContextCompat.getColor(context, R.color.card_background))
            setTextColor(Color.WHITE)
            gravity = Gravity.CENTER
            textSize = 18f
        }
        return ViewHolder(view)
    }

    override fun onBindViewHolder(viewHolder: ViewHolder, item: Any) {
        val settingsItem = item as SettingsItem
        (viewHolder.view as TextView).text = settingsItem.title
    }

    override fun onUnbindViewHolder(viewHolder: ViewHolder) {
        // Nothing to unbind
    }

    companion object {
        private const val GRID_ITEM_WIDTH = 200
        private const val GRID_ITEM_HEIGHT = 200
    }
}
