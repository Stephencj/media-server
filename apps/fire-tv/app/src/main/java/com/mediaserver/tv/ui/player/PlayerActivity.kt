package com.mediaserver.tv.ui.player

import android.os.Bundle
import android.view.KeyEvent
import android.view.View
import androidx.activity.viewModels
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.hls.HlsMediaSource
import androidx.media3.exoplayer.source.ProgressiveMediaSource
import androidx.media3.ui.PlayerView
import com.mediaserver.tv.R
import com.mediaserver.tv.data.models.Media
import com.mediaserver.tv.data.repository.AuthRepository
import com.mediaserver.tv.data.repository.MediaRepository
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch
import javax.inject.Inject

@AndroidEntryPoint
class PlayerActivity : FragmentActivity() {

    @Inject
    lateinit var mediaRepository: MediaRepository

    @Inject
    lateinit var authRepository: AuthRepository

    private val viewModel: PlayerViewModel by viewModels()

    private var player: ExoPlayer? = null
    private lateinit var playerView: PlayerView

    private var media: Media? = null
    private var startPosition: Int = 0
    private var playWhenReady = true
    private var currentPosition: Long = 0

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_player)

        playerView = findViewById(R.id.player_view)

        media = intent.getSerializableExtra(EXTRA_MEDIA) as? Media
        startPosition = intent.getIntExtra(EXTRA_START_POSITION, 0)

        if (media == null) {
            finish()
            return
        }

        currentPosition = startPosition.toLong() * 1000 // Convert to ms

        observeViewModel()
    }

    override fun onStart() {
        super.onStart()
        initializePlayer()
    }

    override fun onResume() {
        super.onResume()
        hideSystemUi()
        if (player == null) {
            initializePlayer()
        }
    }

    override fun onPause() {
        super.onPause()
        saveProgress()
    }

    override fun onStop() {
        super.onStop()
        releasePlayer()
    }

    private fun initializePlayer() {
        val media = media ?: return

        player = ExoPlayer.Builder(this)
            .build()
            .also { exoPlayer ->
                playerView.player = exoPlayer

                // Create data source with auth header
                val dataSourceFactory = DefaultHttpDataSource.Factory()
                    .setDefaultRequestProperties(
                        mapOf("Authorization" to "Bearer ${authRepository.token}")
                    )

                // Try direct play first, fall back to HLS
                val streamUrl = mediaRepository.getDirectPlayUrl(media.id)
                val mediaItem = MediaItem.fromUri(streamUrl)

                val mediaSource = ProgressiveMediaSource.Factory(dataSourceFactory)
                    .createMediaSource(mediaItem)

                exoPlayer.setMediaSource(mediaSource)
                exoPlayer.playWhenReady = playWhenReady
                exoPlayer.seekTo(currentPosition)
                exoPlayer.prepare()

                exoPlayer.addListener(object : Player.Listener {
                    override fun onPlaybackStateChanged(playbackState: Int) {
                        when (playbackState) {
                            Player.STATE_ENDED -> {
                                viewModel.markAsCompleted(
                                    media.id,
                                    (exoPlayer.currentPosition / 1000).toInt(),
                                    (exoPlayer.duration / 1000).toInt(),
                                    media.type.name.lowercase()
                                )
                                finish()
                            }
                            Player.STATE_READY -> {
                                // Player is ready
                            }
                        }
                    }
                })
            }

        // Start periodic progress saving
        startProgressSaving()
    }

    private fun startProgressSaving() {
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                while (true) {
                    kotlinx.coroutines.delay(10_000) // Save every 10 seconds
                    saveProgress()
                }
            }
        }
    }

    private fun saveProgress() {
        val media = media ?: return
        val exoPlayer = player ?: return

        if (exoPlayer.duration > 0) {
            val position = (exoPlayer.currentPosition / 1000).toInt()
            val duration = (exoPlayer.duration / 1000).toInt()
            val completed = position.toFloat() / duration.toFloat() > 0.95f

            viewModel.saveProgress(
                mediaId = media.id,
                position = position,
                duration = duration,
                mediaType = media.type.name.lowercase(),
                completed = completed
            )
        }
    }

    private fun releasePlayer() {
        player?.let { exoPlayer ->
            playWhenReady = exoPlayer.playWhenReady
            currentPosition = exoPlayer.currentPosition
            exoPlayer.release()
        }
        player = null
    }

    private fun hideSystemUi() {
        window.decorView.systemUiVisibility = (
            View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
            or View.SYSTEM_UI_FLAG_LAYOUT_STABLE
            or View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
            or View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
            or View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
            or View.SYSTEM_UI_FLAG_FULLSCREEN
        )
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent?): Boolean {
        val player = player ?: return super.onKeyDown(keyCode, event)

        when (keyCode) {
            KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE,
            KeyEvent.KEYCODE_DPAD_CENTER -> {
                if (player.isPlaying) {
                    player.pause()
                } else {
                    player.play()
                }
                return true
            }
            KeyEvent.KEYCODE_MEDIA_FAST_FORWARD,
            KeyEvent.KEYCODE_DPAD_RIGHT -> {
                player.seekTo(player.currentPosition + 10_000) // +10 seconds
                return true
            }
            KeyEvent.KEYCODE_MEDIA_REWIND,
            KeyEvent.KEYCODE_DPAD_LEFT -> {
                player.seekTo(maxOf(0, player.currentPosition - 10_000)) // -10 seconds
                return true
            }
            KeyEvent.KEYCODE_BACK -> {
                saveProgress()
                finish()
                return true
            }
        }

        return super.onKeyDown(keyCode, event)
    }

    private fun observeViewModel() {
        // Could observe save status if needed
    }

    companion object {
        const val EXTRA_MEDIA = "extra_media"
        const val EXTRA_START_POSITION = "extra_start_position"
    }
}
