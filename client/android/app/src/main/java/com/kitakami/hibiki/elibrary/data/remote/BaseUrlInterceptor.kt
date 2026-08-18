package com.kitakami.hibiki.elibrary.data.remote

import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.Interceptor
import okhttp3.Response

/**
 * Rewrites every request so its scheme / host / port come from the user-configured
 * server root URL. Retrofit is built once with a placeholder base URL
 * (`http://localhost/...`); this interceptor redirects the real traffic to the
 * configured server without rebuilding the Retrofit instance.
 */
class BaseUrlInterceptor(
    private val baseUrlProvider: () -> String,
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val baseStr = baseUrlProvider().trim().trimEnd('/')
        val base = if (baseStr.isBlank()) null else baseStr.toHttpUrlOrNull()
        if (base == null) {
            return chain.proceed(chain.request())
        }
        val request = chain.request()
        val newUrl = base.newBuilder()
            .encodedPath(request.url.encodedPath)
            .encodedQuery(request.url.encodedQuery)
            .build()
        return chain.proceed(request.newBuilder().url(newUrl).build())
    }
}

/** Small helper so the raw string can be parsed to an [HttpUrl] safely. */
internal fun String.toHttpUrlOrNull(): HttpUrl? = runCatching { toHttpUrl() }.getOrNull()
