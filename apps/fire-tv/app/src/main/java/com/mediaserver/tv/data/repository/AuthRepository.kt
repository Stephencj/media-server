package com.mediaserver.tv.data.repository

import android.content.Context
import android.content.SharedPreferences
import androidx.core.content.edit
import com.mediaserver.tv.data.api.AuthResponse
import com.mediaserver.tv.data.api.LoginRequest
import com.mediaserver.tv.data.api.MediaServerApi
import com.mediaserver.tv.data.api.RegisterRequest
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthRepository @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private val prefs: SharedPreferences by lazy {
        context.getSharedPreferences("auth", Context.MODE_PRIVATE)
    }

    var token: String?
        get() = prefs.getString("token", null)
        set(value) = prefs.edit { putString("token", value) }

    var serverUrl: String
        get() = prefs.getString("server_url", "http://10.0.2.2:8080") ?: "http://10.0.2.2:8080"
        set(value) = prefs.edit { putString("server_url", value) }

    val isLoggedIn: Boolean
        get() = token != null

    fun logout() {
        prefs.edit {
            remove("token")
        }
    }

    fun saveAuth(response: AuthResponse) {
        prefs.edit {
            putString("token", response.token)
            putLong("user_id", response.user.id)
            putString("username", response.user.username)
        }
    }
}
