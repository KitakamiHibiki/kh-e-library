package com.kitakami.hibiki.elibrary.ui.bookshelf

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.data.model.Tag
import com.kitakami.hibiki.elibrary.data.model.UpdateBookRequest
import com.kitakami.hibiki.elibrary.data.remote.toApiMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File
import kotlin.math.ceil

/** Special tag value meaning "books not in any shelf" — mirrors backend NoTagFilter. */
const val NO_TAG_FILTER = "__none__"

/** Transient message carrying a string resource + optional format args. */
data class UiMessage(val resId: Int, val args: List<Any> = emptyList())

/** Upload progress surfaced to the UI. */
sealed interface UploadState {
    data object Idle : UploadState
    data class Uploading(val percent: Int) : UploadState
    data class Failed(val message: String) : UploadState
}

class BookShelfViewModel(private val container: AppContainer) : ViewModel() {

    private val repo = container.repository

    /** Files above this size go through chunked upload (matches design doc D6). */
    private val UPLOAD_THRESHOLD = 20L * 1024 * 1024
    private val CHUNK_SIZE = 20 * 1024 * 1024

    val pageSize = 20

    var books by mutableStateOf<List<Book>>(emptyList())
        private set

    var total by mutableStateOf(0)
        private set

    var page by mutableStateOf(1)
        private set

    var keyword by mutableStateOf("")
        private set

    var loading by mutableStateOf(false)
        private set

    var loadingMore by mutableStateOf(false)
        private set

    var error by mutableStateOf<String?>(null)
        private set

    // ---- filters / sort ----------------------------------------------------
    var sortField by mutableStateOf("updated_at")
        private set
    var sortOrder by mutableStateOf("desc")
        private set
    /** Book status filter: "" = all, else unread|reading|finished. */
    var bookStatus by mutableStateOf("")
        private set
    /** Tag filter: "" = all books, [NO_TAG_FILTER] = not in shelf, else a tag name. */
    var tag by mutableStateOf("")
        private set

    // ---- tags (drawer) -----------------------------------------------------
    var tags by mutableStateOf<List<Tag>>(emptyList())
        private set
    var tagsLoading by mutableStateOf(false)
        private set

    // ---- selection (batch delete) ------------------------------------------
    var selectionMode by mutableStateOf(false)
        private set
    var selectedIds by mutableStateOf<Set<Long>>(emptySet())
        private set

    var busy by mutableStateOf(false)
        private set

    /** Transient user-facing message; the screen shows it as a Snackbar/toast. */
    var message by mutableStateOf<UiMessage?>(null)
        private set

    // ---- upload --------------------------------------------------------------
    var uploadState by mutableStateOf<UploadState>(UploadState.Idle)
        private set

    private var searchJob: Job? = null

    init {
        loadTags()
        load(reset = true)
    }

    fun consumeMessage() {
        message = null
    }

    /** Small files upload directly; larger ones go through chunked upload. */
    fun upload(file: File) {
        viewModelScope.launch {
            try {
                if (file.length() <= UPLOAD_THRESHOLD) {
                    uploadState = UploadState.Uploading(100)
                    withContext(Dispatchers.IO) { repo.uploadBook(file) }
                } else {
                    val totalChunks = ceil(file.length().toDouble() / CHUNK_SIZE).toInt()
                    val uploadId = withContext(Dispatchers.IO) {
                        repo.initChunkUpload(file, CHUNK_SIZE, totalChunks)
                    }
                    for (i in 0 until totalChunks) {
                        val chunk = withContext(Dispatchers.IO) { readChunk(file, i, totalChunks) }
                        uploadState = UploadState.Uploading((i + 1) * 100 / totalChunks)
                        withContext(Dispatchers.IO) {
                            repo.uploadChunk(uploadId, i, totalChunks, chunk, file.name)
                        }
                    }
                }
                uploadState = UploadState.Idle
                message = UiMessage(R.string.upload_done)
                load(reset = true)
            } catch (e: Exception) {
                uploadState = UploadState.Idle
                fail(e)
            }
        }
    }

    fun dismissUploadError() {
        if (uploadState is UploadState.Failed) uploadState = UploadState.Idle
    }

    private fun readChunk(file: File, index: Int, totalChunks: Int): ByteArray {
        val from = index.toLong() * CHUNK_SIZE
        val to = if (index == totalChunks - 1) file.length() else (index + 1).toLong() * CHUNK_SIZE
        val size = (to - from).toInt()
        val bytes = ByteArray(size)
        file.inputStream().use { input ->
            input.skip(from)
            var read = 0
            while (read < size) {
                val n = input.read(bytes, read, size - read)
                if (n <= 0) break
                read += n
            }
        }
        return bytes
    }

    fun onKeywordChange(value: String) {
        keyword = value
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(300) // debounce
            load(reset = true)
        }
    }

    fun load(reset: Boolean = false) {
        val target = if (reset) 1 else page + 1
        viewModelScope.launch {
            if (reset) {
                loading = true
                error = null
            } else {
                loadingMore = true
            }
            try {
                val data = repo.listBooks(
                    page = target,
                    pageSize = pageSize,
                    keyword = keyword.trim(),
                    tag = tag,
                    bookStatus = bookStatus,
                    sortField = sortField,
                    sortOrder = sortOrder,
                )
                books = if (reset) data.list else books + data.list
                total = data.total
                page = target
            } catch (e: Exception) {
                if (reset) error = e.toApiMessage()
            } finally {
                loading = false
                loadingMore = false
            }
        }
    }

    /** Triggers the next page when the grid scrolls near the end. */
    fun loadMoreIfNeeded(lastVisibleIndex: Int) {
        if (loading || loadingMore) return
        if (books.size >= total) return
        if (lastVisibleIndex >= books.size - 4) load(reset = false)
    }

    fun retry() = load(reset = true)

    // ---- filters -----------------------------------------------------------

    fun setSort(field: String, order: String) {
        if (sortField == field && sortOrder == order) return
        sortField = field
        sortOrder = order
        load(reset = true)
    }

    fun filterByStatus(status: String) {
        if (bookStatus == status) return
        bookStatus = status
        load(reset = true)
    }

    fun selectTag(newTag: String) {
        if (tag == newTag) return
        tag = newTag
        load(reset = true)
    }

    fun loadTags() {
        viewModelScope.launch {
            tagsLoading = true
            try {
                tags = repo.listTags()
            } catch (_: Exception) {
                // Drawer is secondary; ignore failures silently.
            } finally {
                tagsLoading = false
            }
        }
    }

    fun createShelf(name: String) {
        viewModelScope.launch {
            try {
                repo.createTag(name.trim())
                message = UiMessage(R.string.tags_done)
                loadTags()
            } catch (e: Exception) {
                fail(e)
            }
        }
    }

    /** Applies tag-association changes for one book and refreshes it in place. */
    fun updateBookTags(bookId: Long, addIds: List<Long>, removeIds: List<Long>) {
        viewModelScope.launch {
            try {
                addIds.forEach { repo.addBookTag(bookId, it) }
                removeIds.forEach { repo.removeBookTag(bookId, it) }
                val fresh = repo.getBook(bookId)
                books = books.map { if (it.id == bookId) fresh else it }
                message = UiMessage(R.string.tags_done)
                loadTags()
            } catch (e: Exception) {
                fail(e)
            }
        }
    }

    // ---- selection / batch delete ------------------------------------------

    fun updateSelectionMode(on: Boolean) {
        selectionMode = on
        if (!on) selectedIds = emptySet()
    }

    fun toggleSelection(id: Long) {
        selectedIds = if (id in selectedIds) selectedIds - id else selectedIds + id
    }

    fun deleteSelected() {
        val ids = selectedIds.toList()
        if (ids.isEmpty()) return
        viewModelScope.launch {
            busy = true
            try {
                val deleted = repo.batchDelete(ids)
                message = UiMessage(R.string.bookshelf_batch_delete_done, listOf(deleted))
                books = books.filterNot { it.id in ids }
                selectedIds = emptySet()
                selectionMode = false
                total = (total - deleted).coerceAtLeast(0)
            } catch (e: Exception) {
                fail(e)
            } finally {
                busy = false
            }
        }
    }

    // ---- card menu actions -------------------------------------------------

    fun setReadStatus(bookId: Long, status: String) {
        viewModelScope.launch {
            try {
                repo.updateBook(bookId, UpdateBookRequest(read_status = status))
                books = books.map { if (it.id == bookId) it.copy(read_status = status) else it }
                message = UiMessage(R.string.menu_status_updated)
            } catch (e: Exception) {
                fail(e)
            }
        }
    }

    fun reprocess(bookId: Long) {
        viewModelScope.launch {
            try {
                repo.reprocess(bookId)
                books = books.map { if (it.id == bookId) it.copy(book_status = "processing") else it }
                message = UiMessage(R.string.menu_status_updated)
            } catch (e: Exception) {
                fail(e)
            }
        }
    }

    fun deleteBook(bookId: Long) {
        viewModelScope.launch {
            try {
                repo.deleteBook(bookId)
                books = books.filterNot { it.id == bookId }
                total = (total - 1).coerceAtLeast(0)
            } catch (e: Exception) {
                fail(e)
            }
        }
    }

    private fun fail(e: Exception) {
        message = UiMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
    }
}
