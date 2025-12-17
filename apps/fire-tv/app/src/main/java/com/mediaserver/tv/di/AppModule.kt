package com.mediaserver.tv.di

import android.content.Context
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.mediaserver.tv.data.api.MediaServerApi
import com.mediaserver.tv.data.repository.AuthRepository
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object AppModule {

    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    @Provides
    @Singleton
    fun provideAuthInterceptor(authRepository: AuthRepository): Interceptor {
        return Interceptor { chain ->
            val original = chain.request()
            val token = authRepository.token

            val request = if (token != null) {
                original.newBuilder()
                    .header("Authorization", "Bearer $token")
                    .header("Content-Type", "application/json")
                    .build()
            } else {
                original.newBuilder()
                    .header("Content-Type", "application/json")
                    .build()
            }

            chain.proceed(request)
        }
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(authInterceptor: Interceptor): OkHttpClient {
        val logging = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }

        return OkHttpClient.Builder()
            .addInterceptor(authInterceptor)
            .addInterceptor(logging)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    @Provides
    @Singleton
    fun provideRetrofit(
        client: OkHttpClient,
        json: Json,
        authRepository: AuthRepository
    ): Retrofit {
        val contentType = "application/json".toMediaType()

        return Retrofit.Builder()
            .baseUrl(authRepository.serverUrl + "/")
            .client(client)
            .addConverterFactory(json.asConverterFactory(contentType))
            .build()
    }

    @Provides
    @Singleton
    fun provideMediaServerApi(retrofit: Retrofit): MediaServerApi {
        return retrofit.create(MediaServerApi::class.java)
    }
}
