package com.kitakami.hibiki.elibrary.ui.bookdetail

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
import kotlinx.coroutines.launch

/** Transient message carrying a string resource + optional format args. */
data class DetailMessage(val resId: Int, val args: List<Any> = emptyList())

class BookDetailViewModel(private val container: AppContainer) : ViewModel() {

    var book by mutableStateOf<Book?>(null)
        private set

    var loading by mutableStateOf(false)
        private set

    var error by mutableStateOf<String?>(null)
        private set

    var tags by mutableStateOf<List<Tag>>(emptyList())
        private set

    var busy by mutableStateOf(false)
        private set

    var message by mutableStateOf<DetailMessage?>(null)
        private set

    var deleted by mutableStateOf(false)
        private set

    fun consumeMessage() {
        message = null
    }

    fun load(id: Long) {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                book = container.repository.getBook(id)
                tags = container.repository.listTags()
            } catch (e: Exception) {
                error = e.toApiMessage()
            } finally {
                loading = false
            }
        }
    }

    fun setReadStatus(status: String) {
        val id = book?.id ?: return
        viewModelScope.launch {
            try {
                container.repository.updateBook(id, UpdateBookRequest(read_status = status))
                book = book?.copy(read_status = status)
                message = DetailMessage(R.string.menu_status_updated)
            } catch (e: Exception) {
                message = DetailMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }

    fun reprocess() {
        val id = book?.id ?: return
        viewModelScope.launch {
            try {
                container.repository.reprocess(id)
                book = book?.copy(book_status = "processing")
                message = DetailMessage(R.string.menu_status_updated)
            } catch (e: Exception) {
                message = DetailMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }

    fun delete() {
        val id = book?.id ?: return
        viewModelScope.launch {
            try {
                container.repository.deleteBook(id)
                deleted = true
            } catch (e: Exception) {
                message = DetailMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }

    fun updateBookTags(addIds: List<Long>, removeIds: List<Long>) {
        val id = book?.id ?: return
        viewModelScope.launch {
            try {
                addIds.forEach { container.repository.addBookTag(id, it) }
                removeIds.forEach { container.repository.removeBookTag(id, it) }
                book = container.repository.getBook(id)
                tags = container.repository.listTags()
                message = DetailMessage(R.string.tags_done)
            } catch (e: Exception) {
                message = DetailMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }
}
