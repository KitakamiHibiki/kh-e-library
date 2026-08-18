package com.kitakami.hibiki.elibrary.data.remote

import com.kitakami.hibiki.elibrary.data.model.ApiResponse
import com.kitakami.hibiki.elibrary.data.model.BatchDeleteRequest
import com.kitakami.hibiki.elibrary.data.model.BatchDeleteResult
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.data.model.BookListData
import com.kitakami.hibiki.elibrary.data.model.BookTagRequest
import com.kitakami.hibiki.elibrary.data.model.ChunkInitRequest
import com.kitakami.hibiki.elibrary.data.model.ChunkInitResult
import com.kitakami.hibiki.elibrary.data.model.ChunkUploadResult
import com.kitakami.hibiki.elibrary.data.model.HealthData
import com.kitakami.hibiki.elibrary.data.model.ReadingProgress
import com.kitakami.hibiki.elibrary.data.model.SaveProgressRequest
import com.kitakami.hibiki.elibrary.data.model.SettingsRequest
import com.kitakami.hibiki.elibrary.data.model.StartUpdateRequest
import com.kitakami.hibiki.elibrary.data.model.StatsOverview
import com.kitakami.hibiki.elibrary.data.model.SystemStatus
import com.kitakami.hibiki.elibrary.data.model.Tag
import com.kitakami.hibiki.elibrary.data.model.TagCreateRequest
import com.kitakami.hibiki.elibrary.data.model.UpdateBookRequest
import com.kitakami.hibiki.elibrary.data.model.UpdateCheckResult
import com.kitakami.hibiki.elibrary.data.model.UpdateStarted
import com.kitakami.hibiki.elibrary.data.model.UpdateStatus
import okhttp3.MultipartBody
import okhttp3.RequestBody
import okhttp3.ResponseBody
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Query
import retrofit2.http.Streaming

/**
 * Backend REST API. All `@GET/@POST` paths are relative to the Retrofit base URL
 * (`http://localhost/api/v1/`); [BaseUrlInterceptor] redirects to the configured
 * server. `/health` lives at the server root, hence the absolute path.
 */
interface ApiService {

    @GET("http://localhost/health")
    suspend fun health(): ApiResponse<HealthData>

    @GET("books/list")
    suspend fun listBooks(
        @Query("page") page: Int = 1,
        @Query("page_size") pageSize: Int = 20,
        @Query("keyword") keyword: String = "",
        @Query("tag") tag: String = "",
        @Query("book_status") bookStatus: String = "",
        @Query("sort_field") sortField: String = "",
        @Query("sort_order") sortOrder: String = "",
    ): ApiResponse<BookListData>

    @GET("books/detail")
    suspend fun getBook(@Query("id") id: Long): ApiResponse<Book>

    @GET("books/progress")
    suspend fun getProgress(@Query("id") id: Long): ApiResponse<ReadingProgress>

    @POST("books/progress/save")
    suspend fun saveProgress(@Body body: SaveProgressRequest): ApiResponse<ReadingProgress>

    @POST("books/update")
    suspend fun updateBook(@Query("id") id: Long, @Body body: UpdateBookRequest): ApiResponse<Unit>

    @POST("books/reprocess")
    suspend fun reprocessBook(@Query("id") id: Long): ApiResponse<Unit>

    @POST("books/delete")
    suspend fun deleteBook(@Query("id") id: Long): ApiResponse<Unit>

    @POST("books/batch_delete")
    suspend fun batchDelete(@Body body: BatchDeleteRequest): ApiResponse<BatchDeleteResult>

    /** Small-file upload (< threshold). Form field name must be `file`. */
    @Multipart
    @POST("books/create")
    suspend fun uploadBook(@Part file: MultipartBody.Part): ApiResponse<Book>

    @POST("books/create/chunk/init")
    suspend fun initChunkUpload(@Body body: ChunkInitRequest): ApiResponse<ChunkInitResult>

    /** One chunk per call; the last chunk returns `completed=true` + the book. */
    @Multipart
    @POST("books/create/chunk")
    suspend fun uploadChunk(
        @Part("upload_id") uploadId: RequestBody,
        @Part("chunk_index") chunkIndex: RequestBody,
        @Part("total_chunks") totalChunks: RequestBody,
        @Part file: MultipartBody.Part,
    ): ApiResponse<ChunkUploadResult>

    @GET("tags/list")
    suspend fun listTags(@Query("keyword") keyword: String = ""): ApiResponse<List<Tag>>

    @POST("tags/create")
    suspend fun createTag(@Body body: TagCreateRequest): ApiResponse<Tag>

    @POST("tags/update")
    suspend fun updateTag(@Query("id") id: Long, @Body body: TagCreateRequest): ApiResponse<Unit>

    @POST("tags/delete")
    suspend fun deleteTag(@Query("id") id: Long): ApiResponse<Unit>

    @POST("books/tags/add")
    suspend fun addBookTag(@Body body: BookTagRequest): ApiResponse<Unit>

    @POST("books/tags/remove")
    suspend fun removeBookTag(@Body body: BookTagRequest): ApiResponse<Unit>

    @GET("settings/list")
    suspend fun listSettings(): ApiResponse<Map<String, String>>

    @POST("settings/update")
    suspend fun updateSettings(@Body body: SettingsRequest): ApiResponse<Unit>

    @GET("stats/overview")
    suspend fun statsOverview(): ApiResponse<StatsOverview>

    @GET("system/status")
    suspend fun systemStatus(): ApiResponse<SystemStatus>

    @GET("system/check-update")
    suspend fun checkUpdate(@Query("repo") repo: String = ""): ApiResponse<UpdateCheckResult>

    @POST("system/start-update")
    suspend fun startUpdate(@Body body: StartUpdateRequest): ApiResponse<UpdateStarted>

    @GET("system/update-status")
    suspend fun getUpdateStatus(): ApiResponse<UpdateStatus>

    /**
     * Full book file stream (supports HTTP Range server-side). Used by readers.
     * Pass a `Range: bytes=start-end` header to fetch one chunk; the server then
     * answers 206 with the partial body.
     */
    @Streaming
    @GET("books/read")
    suspend fun readBook(
        @Query("id") id: Long,
        @Header("Range") range: String? = null,
    ): retrofit2.Response<ResponseBody>

    /** Whole-file download for export/share (supports HTTP Range for chunking). */
    @Streaming
    @GET("books/download")
    suspend fun downloadBook(
        @Query("id") id: Long,
        @Header("Range") range: String? = null,
    ): retrofit2.Response<ResponseBody>
}
