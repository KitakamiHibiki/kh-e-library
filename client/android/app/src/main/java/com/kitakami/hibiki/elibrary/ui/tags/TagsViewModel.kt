package com.kitakami.hibiki.elibrary.ui.tags

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Tag
import com.kitakami.hibiki.elibrary.data.remote.toApiMessage
import kotlinx.coroutines.launch

data class TagsMessage(val resId: Int, val args: List<Any> = emptyList())

class TagsViewModel(private val container: AppContainer) : ViewModel() {

    private val repo = container.repository

    var tags by mutableStateOf<List<Tag>>(emptyList())
        private set

    var loading by mutableStateOf(false)
        private set

    var message by mutableStateOf<TagsMessage?>(null)
        private set

    fun consumeMessage() {
        message = null
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            try {
                tags = repo.listTags()
            } catch (e: Exception) {
                message = TagsMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            } finally {
                loading = false
            }
        }
    }

    fun create(name: String) {
        viewModelScope.launch {
            try {
                repo.createTag(name.trim())
                message = TagsMessage(R.string.tags_done)
                load()
            } catch (e: Exception) {
                message = TagsMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }

    fun rename(id: Long, name: String) {
        viewModelScope.launch {
            try {
                repo.updateTag(id, name.trim())
                message = TagsMessage(R.string.tags_done)
                load()
            } catch (e: Exception) {
                message = TagsMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }

    fun delete(id: Long) {
        viewModelScope.launch {
            try {
                repo.deleteTag(id)
                load()
            } catch (e: Exception) {
                message = TagsMessage(R.string.bookshelf_load_failed, listOf(e.toApiMessage()))
            }
        }
    }
}
