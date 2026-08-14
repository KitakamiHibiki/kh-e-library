package com.kitakami.hibiki.elibrary

import android.content.Context
import com.kitakami.hibiki.elibrary.data.local.ServerConfigStore
import com.kitakami.hibiki.elibrary.data.remote.ApiException
import com.kitakami.hibiki.elibrary.data.remote.ApiService
import com.kitakami.hibiki.elibrary.data.remote.BaseUrlInterceptor
import com.kitakami.hibiki.elibrary.data.remote.toHttpUrlOrNull
import com.kitakami.hibiki.elibrary.data.repository.ElibraryRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.MediaType.Companion.toMediaType
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory

/**
 * Manual dependency container (Hilt planned for a later iteration). Lives in
 * [KHApplication]; composables access it via the application instance.
 */
class AppContainer(private val context: Context) {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
    }

    val serverConfigStore = ServerConfigStore(context)

    /** Mirrors [serverConfigStore.baseUrlFlow] for synchronous reads in interceptors. */
    private val baseUrlFlow = MutableStateFlow("")
    val baseUrl: StateFlow<String> = baseUrlFlow

    private val okHttpClient = OkHttpClient.Builder()
        .addInterceptor(BaseUrlInterceptor { baseUrlFlow.value })
        .build()

    val api: ApiService = Retrofit.Builder()
        .baseUrl("http://localhost/api/v1/")
        .client(okHttpClient)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()
        .create(ApiService::class.java)

    val repository = ElibraryRepository(api)

    init {
        scope.launch {
            serverConfigStore.baseUrlFlow.collectLatest { baseUrlFlow.value = it }
        }
    }

    suspend fun saveBaseUrl(url: String) = serverConfigStore.setBaseUrl(url)

    /** Tests connectivity to a candidate root URL before it is persisted. */
    suspend fun testConnection(url: String): Result<Unit> = withContext(Dispatchers.IO) {
        runCatching {
            val root = url.trim().trimEnd('/').toHttpUrlOrNull()
                ?: throw ApiException("地址无效", -1)
            val healthUrl = root.newBuilder().encodedPath("/health").build()
            val response = OkHttpClient().newCall(
                Request.Builder().url(healthUrl).get().build(),
            ).execute()
            response.use { resp ->
                if (!resp.isSuccessful) throw ApiException("HTTP ${resp.code}", resp.code)
            }
        }
    }
}
