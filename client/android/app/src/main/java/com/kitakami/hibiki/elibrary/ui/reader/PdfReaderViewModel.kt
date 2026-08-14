package com.kitakami.hibiki.elibrary.ui.reader

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.data.model.SaveProgressRequest
import com.kitakami.hibiki.elibrary.data.remote.toApiMessage
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.File

/**
 * Loads the PDF via `GET /books/read`, tracks the current page and auto-saves
 * progress (debounced) to the server. `cfi` stores `page:N`; `progress` = page/total.
 */
class PdfReaderViewModel(
    private val container: AppContainer,
    private val context: Context,
    val bookId: Long,
) : ViewModel() {

    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var file: File? by mutableStateOf(null)
        private set

    var pageCount by mutableStateOf(0)
        private set
    var currentPage by mutableStateOf(0)
        private set
    var restored by mutableStateOf(false)
        private set

    /** True when the reader reached the last page (drives the completion UI). */
    var reachedEnd by mutableStateOf(false)
        private set

    private var saveJob: Job? = null
    private var lastSavedPage = -1

    init {
        load()
    }

    private fun load() {
        viewModelScope.launch {
            loading = true
            try {
                val target = File(context.cacheDir, "reader_${bookId}.pdf")
                if (!target.exists() || target.length() == 0L) {
                    container.repository.downloadReadToFile(bookId, target)
                }
                file = target
                // Restore last page.
                val progress = container.repository.getProgress(bookId)
                val page = progress?.cfi?.removePrefix("page:")?.toIntOrNull()
                if (page != null && page > 0) currentPage = page - 1
                restored = true
            } catch (e: Exception) {
                error = e.toApiMessage()
            } finally {
                loading = false
            }
        }
    }

    fun onPageChanged(page: Int) {
        currentPage = page
        if (pageCount > 0) {
            reachedEnd = page >= pageCount - 1
            scheduleSave()
        }
    }

    fun onPageCountLoaded(count: Int) {
        pageCount = count
    }

    fun goToPage(page: Int) {
        val p = page.coerceIn(1, pageCount.coerceAtLeast(1))
        onPageChanged(p - 1)
    }

    fun markFinished(onDone: () -> Unit) {
        viewModelScope.launch {
            try {
                container.repository.updateBookReadStatus(bookId, "finished")
                onDone()
            } catch (e: Exception) {
                error = e.toApiMessage()
            }
        }
    }

    fun reset(onDone: () -> Unit) {
        viewModelScope.launch {
            try {
                container.repository.saveProgress(
                    SaveProgressRequest(book_id = bookId, progress = 0.0, cfi = "page:1"),
                )
                currentPage = 0
                reachedEnd = false
                onDone()
            } catch (e: Exception) {
                error = e.toApiMessage()
            }
        }
    }

    private fun scheduleSave() {
        if (currentPage == lastSavedPage) return
        lastSavedPage = currentPage
        saveJob?.cancel()
        saveJob = viewModelScope.launch {
            delay(2000) // debounce
            try {
                val total = pageCount.coerceAtLeast(1)
                container.repository.saveProgress(
                    SaveProgressRequest(
                        book_id = bookId,
                        progress = (currentPage + 1).toDouble() / total,
                        cfi = "page:${currentPage + 1}",
                    ),
                )
            } catch (_: Exception) {
            }
        }
    }
}
