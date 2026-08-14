package com.kitakami.hibiki.elibrary.ui.reader

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.data.model.SaveProgressRequest
import com.kitakami.hibiki.elibrary.data.reader.EpubBook
import com.kitakami.hibiki.elibrary.data.reader.EpubChapter
import com.kitakami.hibiki.elibrary.data.reader.EpubParser
import com.kitakami.hibiki.elibrary.data.remote.toApiMessage
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.File

/**
 * Downloads + parses an EPUB, keeps chapter/scroll state, and auto-saves
 * progress (debounced) to the server. Locator format: `ch<index>:<ratio>` where
 * ratio is 0..1 scroll position inside the chapter.
 */
class EpubReaderViewModel(
    private val container: AppContainer,
    private val context: Context,
    val bookId: Long,
) : ViewModel() {

    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var book: EpubBook? by mutableStateOf(null)
        private set

    var chapterIndex by mutableStateOf(0)
        private set
    var ratio by mutableStateOf(0f)
        private set

    var fontSize by mutableStateOf(18)
        private set
    var darkMode by mutableStateOf(false)
        private set

    var finished by mutableStateOf(false)
        private set

    private var saveJob: Job? = null
    private var lastSavedLocator = ""

    init {
        load()
    }

    private fun load() {
        viewModelScope.launch {
            loading = true
            try {
                val epub = File(context.cacheDir, "reader_${bookId}.epub")
                if (!epub.exists() || epub.length() == 0L) {
                    container.repository.downloadReadToFile(bookId, epub)
                }
                val extractDir = File(context.cacheDir, "reader_${bookId}")
                val parsed = EpubParser.parse(epub, extractDir)
                book = parsed

                // Restore last position: cfi = "ch<index>:<ratio>".
                val progress = container.repository.getProgress(bookId)
                val loc = progress?.cfi
                if (loc != null) {
                    val m = Regex("""ch(\d+):([\d.]+)""").find(loc)
                    if (m != null) {
                        val idx = m.groupValues[1].toInt().coerceIn(0, parsed.chapters.lastIndex)
                        val r = m.groupValues[2].toFloatOrNull()?.coerceIn(0f, 1f) ?: 0f
                        chapterIndex = idx
                        ratio = r
                    }
                }
            } catch (e: Exception) {
                error = e.toApiMessage()
            } finally {
                loading = false
            }
        }
    }

    fun currentChapter(): EpubChapter? = book?.chapters?.getOrNull(chapterIndex)

    fun onScrollRatio(newRatio: Float) {
        ratio = newRatio.coerceIn(0f, 1f)
        scheduleSave()
    }

    fun goToChapter(index: Int) {
        val clamped = index.coerceIn(0, (book?.chapters?.size ?: 1) - 1)
        if (clamped == chapterIndex) return
        chapterIndex = clamped
        ratio = 0f
        scheduleSave()
    }

    /** Advances to the next chapter; returns false when already at the last one. */
    fun next(): Boolean {
        val last = (book?.chapters?.size ?: 1) - 1
        if (chapterIndex >= last) {
            finished = true
            scheduleSave()
            return false
        }
        chapterIndex += 1
        ratio = 0f
        scheduleSave()
        return true
    }

    fun prev() {
        if (chapterIndex > 0) {
            chapterIndex -= 1
            ratio = 0f
            scheduleSave()
        }
    }

    fun updateFontSize(size: Int) {
        fontSize = size.coerceIn(12, 32)
    }

    fun toggleDark() {
        darkMode = !darkMode
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

    fun readAgain(onDone: () -> Unit) {
        viewModelScope.launch {
            try {
                container.repository.saveProgress(
                    SaveProgressRequest(book_id = bookId, progress = 0.0, cfi = "ch0:0.0"),
                )
                chapterIndex = 0
                ratio = 0f
                finished = false
                onDone()
            } catch (e: Exception) {
                error = e.toApiMessage()
            }
        }
    }

    fun backToLastPage() {
        finished = false
    }

    private fun locator(): String = "ch$chapterIndex:${"%.3f".format(ratio)}"

    private fun scheduleSave() {
        val loc = locator()
        if (loc == lastSavedLocator) return
        lastSavedLocator = loc
        saveJob?.cancel()
        saveJob = viewModelScope.launch {
            delay(2000) // debounce
            val total = (book?.chapters?.size ?: 1)
            val progress = (chapterIndex + ratio).toDouble() / total
            try {
                container.repository.saveProgress(
                    SaveProgressRequest(book_id = bookId, progress = progress, cfi = locator()),
                )
            } catch (_: Exception) {
            }
        }
    }
}
