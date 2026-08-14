package com.kitakami.hibiki.elibrary.ui.settings

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.StatsOverview
import com.kitakami.hibiki.elibrary.data.model.UpdateCheckResult
import com.kitakami.hibiki.elibrary.data.model.UpdateStatus
import com.kitakami.hibiki.elibrary.data.remote.toApiMessage
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

data class SettingsMessage(val resId: Int, val args: List<Any> = emptyList())

class SettingsViewModel(private val container: AppContainer) : ViewModel() {

    private val repo = container.repository

    var settings by mutableStateOf<Map<String, String>>(emptyMap())
        private set
    var stats by mutableStateOf<StatsOverview?>(null)
        private set
    var currentVersion by mutableStateOf("")
        private set
    var loading by mutableStateOf(false)
        private set
    var saving by mutableStateOf(false)
        private set
    var message by mutableStateOf<SettingsMessage?>(null)
        private set

    // Software update.
    var updateResult by mutableStateOf<UpdateCheckResult?>(null)
        private set
    var checkingUpdate by mutableStateOf(false)
        private set
    var updateStatus by mutableStateOf<UpdateStatus?>(null)
        private set
    var updateRunning by mutableStateOf(false)
        private set

    private var pollJob: Job? = null

    fun consumeMessage() {
        message = null
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            try {
                settings = repo.listSettings()
            } catch (_: Exception) {
                // Runtime settings optional; stats/version still load.
            }
            try {
                stats = repo.statsOverview()
            } catch (_: Exception) {
            }
            try {
                currentVersion = repo.systemStatus().version
            } catch (_: Exception) {
            }
            loading = false
        }
    }

    fun save(changed: Map<String, String>) {
        if (changed.isEmpty()) return
        viewModelScope.launch {
            saving = true
            try {
                repo.updateSettings(changed)
                settings = settings + changed
                message = SettingsMessage(R.string.setting_saved)
            } catch (e: Exception) {
                message = SettingsMessage(R.string.setting_save_failed, listOf(e.toApiMessage()))
            } finally {
                saving = false
            }
        }
    }

    // ---- software update ----------------------------------------------------

    fun checkUpdate() {
        viewModelScope.launch {
            checkingUpdate = true
            updateResult = null
            try {
                updateResult = repo.checkUpdate()
            } catch (e: Exception) {
                message = SettingsMessage(R.string.update_failed, listOf(e.toApiMessage()))
            } finally {
                checkingUpdate = false
            }
        }
    }

    fun startUpdate(downloadUrl: String) {
        viewModelScope.launch {
            try {
                repo.startUpdate(downloadUrl)
                updateRunning = true
                pollUpdateStatus()
            } catch (e: Exception) {
                message = SettingsMessage(R.string.update_start_failed, listOf(e.toApiMessage()))
            }
        }
    }

    private fun pollUpdateStatus() {
        pollJob?.cancel()
        pollJob = viewModelScope.launch {
            while (true) {
                try {
                    val status = repo.updateStatus() ?: return@launch
                    updateStatus = status
                    when (status.state) {
                        "completed", "failed" -> {
                            updateRunning = false
                            return@launch
                        }
                        else -> {}
                    }
                } catch (_: Exception) {
                    // Server may be restarting; keep polling briefly.
                }
                delay(1500)
            }
        }
    }

    override fun onCleared() {
        pollJob?.cancel()
    }
}
