package com.mediaserver.tv.ui.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.mediaserver.tv.data.api.MediaServerApi
import com.mediaserver.tv.data.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val api: MediaServerApi
) : ViewModel() {

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _loginSuccess = MutableStateFlow(false)
    val loginSuccess: StateFlow<Boolean> = _loginSuccess.asStateFlow()

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()

    val serverUrl: String
        get() = authRepository.serverUrl

    fun login(serverUrl: String, username: String, password: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null

            // Update server URL
            authRepository.serverUrl = serverUrl

            val result = authRepository.login(api, username, password)

            result.fold(
                onSuccess = {
                    _loginSuccess.value = true
                },
                onFailure = { exception ->
                    _error.value = exception.message ?: "Login failed"
                }
            )

            _isLoading.value = false
        }
    }

    fun register(serverUrl: String, username: String, email: String, password: String) {
        viewModelScope.launch {
            _isLoading.value = true
            _error.value = null

            // Update server URL
            authRepository.serverUrl = serverUrl

            val result = authRepository.register(api, username, email, password)

            result.fold(
                onSuccess = {
                    _loginSuccess.value = true
                },
                onFailure = { exception ->
                    _error.value = exception.message ?: "Registration failed"
                }
            )

            _isLoading.value = false
        }
    }

    fun clearError() {
        _error.value = null
    }
}
