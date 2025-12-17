package com.mediaserver.tv.ui.browse

import android.content.Intent
import android.os.Bundle
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.lifecycleScope
import com.mediaserver.tv.R
import com.mediaserver.tv.data.repository.AuthRepository
import com.mediaserver.tv.ui.login.LoginActivity
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : FragmentActivity() {

    @Inject
    lateinit var authRepository: AuthRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        lifecycleScope.launch {
            val isAuthenticated = authRepository.isAuthenticated.first()

            if (!isAuthenticated) {
                startActivity(Intent(this@MainActivity, LoginActivity::class.java))
                finish()
                return@launch
            }

            setContentView(R.layout.activity_main)

            if (savedInstanceState == null) {
                supportFragmentManager.beginTransaction()
                    .replace(R.id.main_frame, MainFragment())
                    .commit()
            }
        }
    }
}
