package com.kitakami.hibiki.elibrary.data.model

import kotlinx.serialization.Serializable

/**
 * Response envelope used by every API endpoint:
 * `{"code":200,"data":...,"msg":"ok"}`.
 *
 * `data` is nullable because several endpoints return `Success(c, nil)`
 * (i.e. `"data":null`); `coerceInputValues` in the app's [Json] keeps this
 * tolerant of missing values too.
 */
@Serializable
data class ApiResponse<T>(
    val code: Int,
    val data: T? = null,
    val msg: String = "",
)

@Serializable
data class HealthData(
    val status: String,
)

/** Mirrors the backend `model.Book`. */
@Serializable
data class Book(
    val id: Long,
    val title: String,
    val author: String = "",
    val publisher: String? = null,
    val isbn: String? = null,
    val cover: String = "",
    val file: String = "",
    val file_type: String = "",
    val file_size: Long = 0,
    val file_hash: String = "",
    val description: String? = null,
    val language: String? = null,
    val tags: List<String> = emptyList(),
    val pages: Int = 0,
    val storage_key: String = "local",
    val read_status: String = "unread", // unread | reading | finished
    val book_status: String = "ready",  // processing | failed | ready | deleted
    val created_at: Long = 0,
    val updated_at: Long = 0,
) {
    val isEpub: Boolean get() = file_type.equals("epub", ignoreCase = true)
    val isProcessing: Boolean get() = book_status == "processing"
    val isFailed: Boolean get() = book_status == "failed"
}

@Serializable
data class BookListData(
    val list: List<Book>,
    val total: Int,
    val page: Int,
)

@Serializable
data class Tag(
    val id: Long,
    val name: String,
    val count: Int = 0,
)

@Serializable
data class ReadingProgress(
    val id: Long? = null,
    val book_id: Long,
    val progress: Double = 0.0,
    val cfi: String? = null,
    val chapter_href: String? = null,
    val created_at: Long = 0,
    val updated_at: Long = 0,
)

@Serializable
data class SaveProgressRequest(
    val book_id: Long,
    val progress: Double,
    val cfi: String? = null,
    val chapter_href: String? = null,
)

@Serializable
data class StatsOverview(
    val total_books: Int = 0,
    val unread_count: Int = 0,
    val reading_count: Int = 0,
    val finished_count: Int = 0,
    val recent_reading: List<RecentReading> = emptyList(),
)

@Serializable
data class RecentReading(
    val id: Long,
    val title: String,
    val updated_at: Long,
)

@Serializable
data class SystemStatus(
    val version: String,
    val platform: String = "",
    val executable: String = "",
)

@Serializable
data class UpdateCheckResult(
    val current_version: String,
    val latest_version: String,
    val has_update: Boolean,
    val release_notes: String = "",
    val download_url: String = "",
    val file_name: String = "",
    val file_size: Long = 0,
    val published_at: String = "",
    val release_url: String = "",
)

@Serializable
data class UpdateStatus(
    val state: String = "idle", // idle | downloading | installing | completed | failed
    val message: String = "",
    val progress: Int = 0,
)

@Serializable
data class StartUpdateRequest(val download_url: String)

@Serializable
data class UpdateStarted(val status: String = "")

@Serializable
data class UpdateBookRequest(
    val title: String? = null,
    val author: String? = null,
    val publisher: String? = null,
    val read_status: String? = null,
)

@Serializable
data class BatchDeleteRequest(val ids: List<Long>)

@Serializable
data class BatchDeleteResult(val deleted: Int = 0)

@Serializable
data class ChunkInitRequest(
    val file_name: String,
    val file_size: Long,
    val total_chunks: Int,
    val chunk_size: Int,
)

@Serializable
data class ChunkInitResult(val upload_id: Long)

/** Response of one chunk POST. `completed=true` carries the finalized book. */
@Serializable
data class ChunkUploadResult(
    val completed: Boolean = false,
    val book: Book? = null,
    val chunk_index: Int? = null,
    val received: Boolean? = null,
)

@Serializable
data class BookTagRequest(
    val book_id: Long,
    val tag_id: Long,
)

@Serializable
data class TagCreateRequest(val name: String)

@Serializable
data class SettingsRequest(val settings: Map<String, String>)
