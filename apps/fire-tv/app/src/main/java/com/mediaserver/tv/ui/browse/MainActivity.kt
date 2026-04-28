package com.mediaserver.tv.ui.browse

import android.os.Bundle
import android.view.View
import android.widget.Button
import android.widget.TextView
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.lifecycleScope
import com.mediaserver.tv.R
import com.mediaserver.tv.data.repository.AuthRepository
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : FragmentActivity() {

    @Inject
    lateinit var authRepository: AuthRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        attemptAuth(savedInstanceState)
    }

    private fun attemptAuth(savedInstanceState: Bundle?) {
        lifecycleScope.launch {
            authRepository.ensureAuthenticated()

            if (authRepository.isAuthenticated.value) {
                showMain(savedInstanceState)
            } else {
                showConnectError(savedInstanceState)
            }
        }
    }

    private fun showMain(savedInstanceState: Bundle?) {
        setContentView(R.layout.activity_main)
        if (savedInstanceState == null) {
            supportFragmentManager.beginTransaction()
                .replace(R.id.main_frame, MainFragment())
                .commit()
        }
    }

    private fun showConnectError(savedInstanceState: Bundle?) {
        setContentView(R.layout.activity_connect_error)
        findViewById<TextView>(R.id.error_server_url)?.text = authRepository.serverUrl
        val errorView = findViewById<TextView>(R.id.error_message)
        val message = authRepository.lastAuthError.value
        if (message != null) {
            errorView?.text = message
            errorView?.visibility = View.VISIBLE
        } else {
            errorView?.visibility = View.GONE
        }
        findViewById<Button>(R.id.error_retry_button)?.setOnClickListener {
            attemptAuth(savedInstanceState)
        }
    }
}
