package com.mediaserver.tv.data.repository

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import com.mediaserver.tv.Secrets
import com.mediaserver.tv.data.api.AuthResponse
import com.mediaserver.tv.data.api.ExchangeKeyRequest
import com.mediaserver.tv.data.api.MediaServerApi
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Provider
import javax.inject.Singleton

@Singleton
class AuthRepository @Inject constructor(
    @ApplicationContext private val context: Context,
    private val apiProvider: Provider<MediaServerApi>
) {
    private val prefs: SharedPreferences by lazy {
        context.getSharedPreferences("auth", Context.MODE_PRIVATE)
    }

    var token: String?
        get() = prefs.getString("token", null)
        private set(value) = prefs.edit { putString("token", value) }

    private var tokenExpiration: Long
        get() = prefs.getLong("token_expiration", 0L)
        set(value) = prefs.edit { putLong("token_expiration", value) }

    val serverUrl: String = Secrets.SERVER_URL

    private val _isAuthenticated = MutableStateFlow(hasValidToken())
    val isAuthenticated: StateFlow<Boolean> = _isAuthenticated.asStateFlow()

    private val _lastAuthError = MutableStateFlow<String?>(null)
    val lastAuthError: StateFlow<String?> = _lastAuthError.asStateFlow()

    fun logout() {
        prefs.edit {
            remove("token")
            remove("token_expiration")
            remove("user_id")
            remove("username")
        }
        _isAuthenticated.value = false
    }

    private fun hasValidToken(): Boolean {
        val t = token ?: return false
        val exp = tokenExpiration
        return t.isNotEmpty() && exp > 0 && System.currentTimeMillis() / 1000 < exp
    }

    /**
     * Silently exchanges the embedded device key for a JWT, retrying on
     * transient network errors. Surfaces the failure cause via
     * [lastAuthError] so the connect-failure UI can show what happened.
     */
    suspend fun ensureAuthenticated() {
        if (hasValidToken()) {
            _isAuthenticated.value = true
            return
        }

        _lastAuthError.value = null
        val backoffsMs = longArrayOf(500L, 2000L, 5000L)
        for ((attempt, delayMs) in backoffsMs.withIndex()) {
            try {
                val response = apiProvider.get().exchangeDeviceKey(ExchangeKeyRequest(Secrets.API_KEY))
                saveAuth(response)
                _isAuthenticated.value = true
                return
            } catch (e: retrofit2.HttpException) {
                if (e.code() == 401) {
                    _lastAuthError.value = "Invalid device key — rebuild the app with a valid Secrets.kt."
                    return
                }
                if (attempt == backoffsMs.lastIndex) {
                    _lastAuthError.value = "Server error ${e.code()} from ${Secrets.SERVER_URL}"
                    return
                }
                delay(delayMs)
            } catch (e: Exception) {
                if (attempt == backoffsMs.lastIndex) {
                    _lastAuthError.value = "Cannot reach ${Secrets.SERVER_URL}: ${e.message}"
                    return
                }
                delay(delayMs)
            }
        }
    }

    private fun saveAuth(response: AuthResponse) {
        prefs.edit {
            putString("token", response.token)
            putLong("token_expiration", response.expiresAt)
            putLong("user_id", response.user.id)
            putString("username", response.user.username)
        }
    }
}
