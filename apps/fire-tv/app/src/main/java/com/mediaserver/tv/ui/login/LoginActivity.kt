package com.mediaserver.tv.ui.login

import android.content.Intent
import android.os.Bundle
import android.view.View
import android.widget.Toast
import androidx.activity.viewModels
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.mediaserver.tv.databinding.ActivityLoginBinding
import com.mediaserver.tv.ui.browse.MainActivity
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.launch

@AndroidEntryPoint
class LoginActivity : FragmentActivity() {

    private lateinit var binding: ActivityLoginBinding
    private val viewModel: LoginViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityLoginBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setupUI()
        observeViewModel()
    }

    private fun setupUI() {
        binding.serverUrlInput.setText(viewModel.serverUrl)

        binding.loginButton.setOnClickListener {
            attemptLogin()
        }

        binding.registerButton.setOnClickListener {
            attemptRegister()
        }

        // Focus handling for D-pad navigation
        binding.serverUrlInput.setOnFocusChangeListener { _, hasFocus ->
            binding.serverUrlLabel.alpha = if (hasFocus) 1f else 0.7f
        }

        binding.usernameInput.setOnFocusChangeListener { _, hasFocus ->
            binding.usernameLabel.alpha = if (hasFocus) 1f else 0.7f
        }

        binding.passwordInput.setOnFocusChangeListener { _, hasFocus ->
            binding.passwordLabel.alpha = if (hasFocus) 1f else 0.7f
        }
    }

    private fun observeViewModel() {
        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                launch {
                    viewModel.isLoading.collect { isLoading ->
                        binding.progressBar.visibility = if (isLoading) View.VISIBLE else View.GONE
                        binding.loginButton.isEnabled = !isLoading
                        binding.registerButton.isEnabled = !isLoading
                    }
                }

                launch {
                    viewModel.loginSuccess.collect { success ->
                        if (success) {
                            navigateToMain()
                        }
                    }
                }

                launch {
                    viewModel.error.collect { error ->
                        error?.let {
                            Toast.makeText(this@LoginActivity, it, Toast.LENGTH_LONG).show()
                            viewModel.clearError()
                        }
                    }
                }
            }
        }
    }

    private fun attemptLogin() {
        val serverUrl = binding.serverUrlInput.text.toString().trim()
        val username = binding.usernameInput.text.toString().trim()
        val password = binding.passwordInput.text.toString()

        if (serverUrl.isEmpty() || username.isEmpty() || password.isEmpty()) {
            Toast.makeText(this, "Please fill in all fields", Toast.LENGTH_SHORT).show()
            return
        }

        viewModel.login(serverUrl, username, password)
    }

    private fun attemptRegister() {
        val serverUrl = binding.serverUrlInput.text.toString().trim()
        val username = binding.usernameInput.text.toString().trim()
        val email = binding.emailInput.text.toString().trim()
        val password = binding.passwordInput.text.toString()

        if (serverUrl.isEmpty() || username.isEmpty() || email.isEmpty() || password.isEmpty()) {
            Toast.makeText(this, "Please fill in all fields", Toast.LENGTH_SHORT).show()
            return
        }

        viewModel.register(serverUrl, username, email, password)
    }

    private fun navigateToMain() {
        startActivity(Intent(this, MainActivity::class.java))
        finish()
    }
}
