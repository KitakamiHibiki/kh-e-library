package com.kitakami.hibiki.elibrary.data.local

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

private val Context.serverConfigDataStore by preferencesDataStore(name = "server_config")

/**
 * Persists the user-configured server root URL. The rest of the app observes
 * [baseUrlFlow]; the network layer uses it to route every request.
 */
class ServerConfigStore(private val context: Context) {

    private val baseUrlKey = stringPreferencesKey("base_url")

    val baseUrlFlow: Flow<String> = context.serverConfigDataStore.data
        .map { prefs -> prefs[baseUrlKey] ?: "" }

    suspend fun setBaseUrl(url: String) {
        context.serverConfigDataStore.edit { prefs ->
            prefs[baseUrlKey] = url.trim().trimEnd('/')
        }
    }
}
