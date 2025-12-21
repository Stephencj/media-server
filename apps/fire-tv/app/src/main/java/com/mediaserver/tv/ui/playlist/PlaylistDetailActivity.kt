package com.mediaserver.tv.ui.playlist

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.fragment.app.FragmentActivity
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class PlaylistDetailActivity : FragmentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (savedInstanceState == null) {
            val playlistId = intent.getLongExtra(EXTRA_PLAYLIST_ID, -1)
            val playlistName = intent.getStringExtra(EXTRA_PLAYLIST_NAME) ?: "Playlist"

            supportFragmentManager.beginTransaction()
                .replace(android.R.id.content, PlaylistDetailFragment.newInstance(playlistId, playlistName))
                .commit()
        }
    }

    companion object {
        private const val EXTRA_PLAYLIST_ID = "playlist_id"
        private const val EXTRA_PLAYLIST_NAME = "playlist_name"

        fun createIntent(context: Context, playlistId: Long, playlistName: String): Intent {
            return Intent(context, PlaylistDetailActivity::class.java).apply {
                putExtra(EXTRA_PLAYLIST_ID, playlistId)
                putExtra(EXTRA_PLAYLIST_NAME, playlistName)
            }
        }
    }
}
