package com.mediaserver.tv.ui.detail

import android.os.Bundle
import androidx.fragment.app.FragmentActivity
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.Media
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class DetailActivity : FragmentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_detail)

        if (savedInstanceState == null) {
            val media = intent.getSerializableExtra(EXTRA_MEDIA) as? Media
            val startPosition = intent.getIntExtra(EXTRA_START_POSITION, 0)

            if (media != null) {
                supportFragmentManager.beginTransaction()
                    .replace(R.id.detail_frame, DetailFragment.newInstance(media, startPosition))
                    .commit()
            } else {
                finish()
            }
        }
    }

    companion object {
        const val EXTRA_MEDIA = "extra_media"
        const val EXTRA_START_POSITION = "extra_start_position"
    }
}
