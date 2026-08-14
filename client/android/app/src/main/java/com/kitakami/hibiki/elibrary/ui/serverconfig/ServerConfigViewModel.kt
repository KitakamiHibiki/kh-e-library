package com.kitakami.hibiki.elibrary.ui.serverconfig

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.data.remote.toHttpUrlOrNull
import kotlinx.coroutines.launch

sealed interface TestOutcome {
    data object Idle : TestOutcome
    data object Ok : TestOutcome
    data class Failed(val reason: String) : TestOutcome
}

class ServerConfigViewModel(private val container: AppContainer) : ViewModel() {

    var url by mutableStateOf(container.baseUrl.value)
        private set

    var testing by mutableStateOf(false)
        private set

    var saving by mutableStateOf(false)
        private set

    var testOutcome by mutableStateOf<TestOutcome>(TestOutcome.Idle)
        private set

    fun onUrlChange(value: String) {
        url = value
        testOutcome = TestOutcome.Idle
    }

    fun isUrlValid(): Boolean = url.trim().toHttpUrlOrNull() != null

    fun test() {
        val candidate = url.trim()
        if (candidate.toHttpUrlOrNull() == null) {
            testOutcome = TestOutcome.Failed("地址无效，需以 http:// 或 https:// 开头")
            return
        }
        viewModelScope.launch {
            testing = true
            testOutcome = TestOutcome.Idle
            container.testConnection(candidate).fold(
                onSuccess = { testOutcome = TestOutcome.Ok },
                onFailure = { e ->
                    testOutcome = TestOutcome.Failed(e.message ?: "无法连接服务端")
                },
            )
            testing = false
        }
    }

    fun save(onDone: () -> Unit) {
        viewModelScope.launch {
            saving = true
            container.saveBaseUrl(url.trim())
            saving = false
            onDone()
        }
    }
}
