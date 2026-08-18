package com.kitakami.hibiki.elibrary.data.repository

import com.kitakami.hibiki.elibrary.data.model.ApiResponse
import com.kitakami.hibiki.elibrary.data.model.BatchDeleteRequest
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.data.model.BookListData
import com.kitakami.hibiki.elibrary.data.model.BookTagRequest
import com.kitakami.hibiki.elibrary.data.model.ChunkInitRequest
import com.kitakami.hibiki.elibrary.data.model.ChunkUploadResult
import com.kitakami.hibiki.elibrary.data.model.ReadingProgress
import com.kitakami.hibiki.elibrary.data.model.SaveProgressRequest
import com.kitakami.hibiki.elibrary.data.model.SettingsRequest
import com.kitakami.hibiki.elibrary.data.model.StatsOverview
import com.kitakami.hibiki.elibrary.data.model.SystemStatus
import com.kitakami.hibiki.elibrary.data.model.Tag
import com.kitakami.hibiki.elibrary.data.model.TagCreateRequest
import com.kitakami.hibiki.elibrary.data.model.UpdateBookRequest
import com.kitakami.hibiki.elibrary.data.model.UpdateCheckResult
import com.kitakami.hibiki.elibrary.data.model.UpdateStatus
import com.kitakami.hibiki.elibrary.data.remote.ApiService
import com.kitakami.hibiki.elibrary.data.remote.ApiException
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.ResponseBody
import retrofit2.Response
import java.io.File
import java.io.RandomAccessFile

/**
 * Repository over [ApiService]. Suspend functions throw on failure; callers
 * convert with [com.kitakami.hibiki.elibrary.data.remote.toApiMessage].
 * Responses with a null payload (e.g. `Success(c, nil)`) surface an [ApiException].
 */
class ElibraryRepository(private val api: ApiService) {

    /**
     * Max bytes fetched per Range request during a chunked download. Kept below
     * the Cloudflare tunnel's single-transfer limit (matches the upload chunk
     * size from design doc D6). Lower it if the tunnel limit is tighter.
     */
    private val DOWNLOAD_CHUNK_SIZE = 20L * 1024 * 1024

    // ---- helpers -----------------------------------------------------------

    private fun <T> requireData(resp: ApiResponse<T>): T =
        resp.data ?: throw ApiException(resp.msg.ifBlank { "服务端返回空数据" }, resp.code)

    private fun partFile(name: String, file: File): MultipartBody.Part {
        val mediaType = when (file.extension.lowercase()) {
            "pdf" -> "application/pdf".toMediaType()
            else -> "application/epub+zip".toMediaType()
        }
        return MultipartBody.Part.createFormData(
            name,
            file.name,
            file.asRequestBody(mediaType),
        )
    }

    // ---- system / health ---------------------------------------------------

    suspend fun health(): Boolean = api.health().data?.status == "ok"

    suspend fun systemStatus(): SystemStatus = requireData(api.systemStatus())

    // ---- books -------------------------------------------------------------

    suspend fun listBooks(
        page: Int = 1,
        pageSize: Int = 20,
        keyword: String = "",
        tag: String = "",
        bookStatus: String = "",
        sortField: String = "",
        sortOrder: String = "",
    ): BookListData = requireData(
        api.listBooks(page, pageSize, keyword, tag, bookStatus, sortField, sortOrder),
    )

    suspend fun getBook(id: Long): Book = requireData(api.getBook(id))

    suspend fun updateBook(id: Long, request: UpdateBookRequest) {
        api.updateBook(id, request)
    }

    suspend fun updateBookReadStatus(id: Long, status: String) {
        api.updateBook(id, UpdateBookRequest(read_status = status))
    }

    suspend fun reprocess(id: Long) {
        api.reprocessBook(id)
    }

    suspend fun deleteBook(id: Long) {
        api.deleteBook(id)
    }

    suspend fun batchDelete(ids: List<Long>): Int {
        if (ids.isEmpty()) return 0
        return requireData(api.batchDelete(BatchDeleteRequest(ids))).deleted
    }

    /** Downloads the whole book file (export/share) into [target] and returns it. */
    suspend fun downloadBookToFile(id: Long, target: File): File =
        chunkedDownload(id, target) { range -> api.downloadBook(id, range) }

    /** Downloads the raw book stream (reader) into [target] and returns it. */
    suspend fun downloadReadToFile(id: Long, target: File): File =
        chunkedDownload(id, target) { range -> api.readBook(id, range) }

    /**
     * Chunked download via HTTP Range requests. Each request transfers at most
     * [DOWNLOAD_CHUNK_SIZE] bytes, keeping every single response under the
     * tunnel's per-transfer limit. Skips when [target] is already complete and
     * resumes from its current length when it holds a partial prefix (chunks are
     * written at absolute offsets, so an interrupted run leaves a valid prefix).
     */
    private suspend fun chunkedDownload(
        id: Long,
        target: File,
        fetchRange: suspend (String) -> Response<ResponseBody>,
    ): File {
        // Probe: ask for the first byte to learn the total size from Content-Range.
        val probe = fetchRange("bytes=0-0")
        if (probe.code() == 200) {
            // Server ignored Range → legacy single-shot whole-file download.
            writeWholeBody(probe.body(), target)
            return target
        }
        if (probe.code() != 206) {
            probe.closeQuietly()
            throw ApiException("分片下载不可用（HTTP ${probe.code()}）", probe.code())
        }
        val total = contentRangeTotal(probe)
        probe.closeQuietly()
        if (total == null) throw ApiException("分片下载失败：无法获取文件大小", -1)

        if (target.exists() && target.length() == total) return target
        if (target.exists() && target.length() > total) target.delete() // stale/corrupt
        target.parentFile?.mkdirs()

        RandomAccessFile(target, "rw").use { raf ->
            var offset = if (target.exists()) target.length() else 0L
            while (offset < total) {
                val end = minOf(offset + DOWNLOAD_CHUNK_SIZE - 1, total - 1)
                val resp = fetchRange("bytes=$offset-$end")
                if (resp.code() != 206) {
                    resp.closeQuietly()
                    throw ApiException("分片下载失败（HTTP ${resp.code()}）", resp.code())
                }
                val body = resp.body() ?: run {
                    resp.closeQuietly()
                    throw ApiException("分片下载失败：响应无内容", -1)
                }
                body.use { b ->
                    raf.seek(offset)
                    raf.write(b.bytes())
                }
                offset = end + 1
            }
        }
        return target
    }

    private fun writeWholeBody(body: ResponseBody?, target: File) {
        if (body == null) throw ApiException("分片下载不可用：响应无内容", -1)
        target.parentFile?.mkdirs()
        body.use { b ->
            target.outputStream().use { out -> b.byteStream().copyTo(out) }
        }
    }

    /** Parses the total size out of `Content-Range: bytes 0-0/12430934`. */
    private fun contentRangeTotal(resp: Response<ResponseBody>): Long? {
        val header = resp.headers()["Content-Range"] ?: return null
        val slash = header.lastIndexOf('/')
        if (slash < 0) return null
        return header.substring(slash + 1).trim().toLongOrNull()
    }

    private fun Response<ResponseBody>.closeQuietly() {
        try {
            body()?.close()
        } catch (_: Exception) {
        }
    }

    // ---- progress ----------------------------------------------------------

    suspend fun getProgress(id: Long): ReadingProgress? = api.getProgress(id).data

    suspend fun saveProgress(body: SaveProgressRequest): ReadingProgress =
        requireData(api.saveProgress(body))

    // ---- upload ------------------------------------------------------------

    /** Straight upload for small files; returns the created book. */
    suspend fun uploadBook(file: File): Book =
        requireData(api.uploadBook(partFile("file", file)))

    /** Initializes a chunked upload session and returns its upload id. */
    suspend fun initChunkUpload(
        file: File,
        chunkSize: Int,
        totalChunks: Int,
    ): Long {
        val resp = api.initChunkUpload(
            ChunkInitRequest(
                file_name = file.name,
                file_size = file.length(),
                total_chunks = totalChunks,
                chunk_size = chunkSize,
            ),
        )
        return requireData(resp).upload_id
    }

    /**
     * Uploads a single chunk. Returns `true` when this was the final chunk and
     * the server assembled the book (so the caller stops the loop).
     */
    suspend fun uploadChunk(
        uploadId: Long,
        chunkIndex: Int,
        totalChunks: Int,
        chunk: ByteArray,
        fileName: String,
    ): ChunkUploadResult {
        val body = chunk.toRequestBody("application/octet-stream".toMediaType())
        val part = MultipartBody.Part.createFormData("file", fileName, body)
        val resp = api.uploadChunk(
            uploadId = uploadId.toString().toRequestBody("text/plain".toMediaType()),
            chunkIndex = chunkIndex.toString().toRequestBody("text/plain".toMediaType()),
            totalChunks = totalChunks.toString().toRequestBody("text/plain".toMediaType()),
            file = part,
        )
        return requireData(resp)
    }

    // ---- tags --------------------------------------------------------------

    suspend fun listTags(keyword: String = ""): List<Tag> =
        requireData(api.listTags(keyword))

    suspend fun createTag(name: String): Tag =
        requireData(api.createTag(TagCreateRequest(name)))

    suspend fun updateTag(id: Long, name: String) {
        api.updateTag(id, TagCreateRequest(name))
    }

    suspend fun deleteTag(id: Long) {
        api.deleteTag(id)
    }

    suspend fun addBookTag(bookId: Long, tagId: Long) {
        api.addBookTag(BookTagRequest(bookId, tagId))
    }

    suspend fun removeBookTag(bookId: Long, tagId: Long) {
        api.removeBookTag(BookTagRequest(bookId, tagId))
    }

    // ---- settings ----------------------------------------------------------

    suspend fun listSettings(): Map<String, String> = requireData(api.listSettings())

    suspend fun updateSettings(settings: Map<String, String>) {
        api.updateSettings(SettingsRequest(settings))
    }

    // ---- stats / update ----------------------------------------------------

    suspend fun statsOverview(): StatsOverview = requireData(api.statsOverview())

    suspend fun checkUpdate(repo: String = ""): UpdateCheckResult =
        requireData(api.checkUpdate(repo))

    suspend fun startUpdate(downloadUrl: String) {
        api.startUpdate(com.kitakami.hibiki.elibrary.data.model.StartUpdateRequest(downloadUrl))
    }

    suspend fun updateStatus(): UpdateStatus? = api.getUpdateStatus().data
}
