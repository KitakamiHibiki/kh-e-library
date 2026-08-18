package com.kitakami.hibiki.elibrary.data.remote

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import retrofit2.HttpException
import java.io.IOException

/** Domain error thrown by the repository layer with a user-friendly message. */
class ApiException(message: String, val code: Int = 0) : Exception(message)

@Serializable
private data class ErrorBody(val msg: String? = null, val code: Int? = null)

private val errorJson = Json { ignoreUnknownKeys = true }

/** Extracts a readable message from a Retrofit/network failure. */
fun Throwable.toApiMessage(): String {
    return when (this) {
        is HttpException -> {
            val body = response()?.errorBody()?.string()
            val parsed = body?.let {
                runCatching { errorJson.decodeFromString<ErrorBody>(it) }.getOrNull()
            }
            parsed?.msg?.takeIf { it.isNotBlank() }
                ?: "HTTP ${code()}"
        }
        is IOException -> "网络错误：无法连接到服务端"
        is ApiException -> message ?: "请求失败"
        else -> message ?: "未知错误"
    }
}
