package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/service"
)

// BookHandler handles book-related API requests.
type BookHandler struct {
	svc *service.BookService
}

// NewBookHandler creates a new BookHandler.
func NewBookHandler(svc *service.BookService) *BookHandler {
	return &BookHandler{svc: svc}
}

// List handles GET /books/list
func (h *BookHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")
	tag := c.Query("tag")
	bookStatus := c.Query("book_status")
	sortField := c.Query("sort_field")
	sortOrder := c.Query("sort_order")

	books, total, err := h.svc.List(page, pageSize, keyword, tag, bookStatus, sortField, sortOrder)
	if err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	Success(c, gin.H{
		"list":  books,
		"total": total,
		"page":  page,
	})
}

// GetByID handles GET /books/detail?id=
func (h *BookHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	book, err := h.svc.GetByID(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, "书籍不存在")
		return
	}

	Success(c, book)
}

// Upload handles POST /books/create
func (h *BookHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// Check file size (200MB limit)
	const maxUploadSize = 200 << 20
	if file.Size > maxUploadSize {
		Error(c, http.StatusRequestEntityTooLarge, "文件超过大小限制")
		return
	}

	f, err := file.Open()
	if err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	defer f.Close()

	book, err := h.svc.Upload(file.Filename, f)
	if err != nil {
		msg := err.Error()
		code := http.StatusInternalServerError
		if strings.Contains(msg, "unsupported") || strings.Contains(msg, "校验失败") {
			code = http.StatusBadRequest
		} else if strings.Contains(msg, "已存在") {
			code = http.StatusConflict
		}
		Error(c, code, msg)
		return
	}

	SuccessWithStatus(c, http.StatusCreated, book)
}

// InitChunkUpload handles POST /books/create/chunk/init.
// It validates file metadata and returns an upload session ID.
func (h *BookHandler) InitChunkUpload(c *gin.Context) {
	var req struct {
		FileName    string `json:"file_name"`
		FileSize    int64  `json:"file_size"`
		TotalChunks int    `json:"total_chunks"`
		ChunkSize   int    `json:"chunk_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	uploadID, err := h.svc.InitChunkUpload(req.FileName, req.FileSize, req.TotalChunks, req.ChunkSize)
	if err != nil {
		msg := err.Error()
		code := http.StatusInternalServerError
		if strings.Contains(msg, "不支持") || strings.Contains(msg, "无效") {
			code = http.StatusBadRequest
		} else if strings.Contains(msg, "大小限制") {
			code = http.StatusRequestEntityTooLarge
		}
		Error(c, code, msg)
		return
	}

	Success(c, gin.H{"upload_id": uploadID})
}

// UploadChunk handles POST /books/create/chunk.
// Each request carries one 50MB-or-smaller chunk; the last chunk triggers
// assembly and returns the finalized book.
func (h *BookHandler) UploadChunk(c *gin.Context) {
	uploadID, err := strconv.ParseUint(c.PostForm("upload_id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效: upload_id")
		return
	}
	chunkIndex, err := strconv.Atoi(c.PostForm("chunk_index"))
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效: chunk_index")
		return
	}
	totalChunks, err := strconv.Atoi(c.PostForm("total_chunks"))
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效: total_chunks")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效: 缺少文件数据")
		return
	}

	// Enforce a ~55MB cap per chunk (50MB data + multipart overhead).
	const maxChunkSize = 55 << 20
	if file.Size > maxChunkSize {
		Error(c, http.StatusRequestEntityTooLarge, "分片超过大小限制")
		return
	}

	f, err := file.Open()
	if err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	defer f.Close()

	book, completed, err := h.svc.ReceiveChunk(uint(uploadID), chunkIndex, totalChunks, f)
	if err != nil {
		msg := err.Error()
		code := http.StatusInternalServerError
		if strings.Contains(msg, "不存在") || strings.Contains(msg, "已过期") {
			code = http.StatusNotFound
		} else if strings.Contains(msg, "无效") || strings.Contains(msg, "校验失败") {
			code = http.StatusBadRequest
		} else if strings.Contains(msg, "已存在") {
			code = http.StatusConflict
		}
		Error(c, code, msg)
		return
	}

	if completed {
		SuccessWithStatus(c, http.StatusCreated, gin.H{"completed": true, "book": book})
	} else {
		Success(c, gin.H{"chunk_index": chunkIndex, "received": true})
	}
}

// Update handles POST /books/update?id=
func (h *BookHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// Only allow specific fields
	allowed := map[string]bool{
		"title":       true,
		"author":      true,
		"publisher":   true,
		"read_status": true,
	}

	fields := make(map[string]interface{})
	for k, v := range body {
		if !allowed[k] {
			continue
		}
		// Validate read_status
		if k == "read_status" {
			s, ok := v.(string)
			if !ok || (s != "unread" && s != "reading" && s != "finished") {
				Error(c, http.StatusBadRequest, "校验失败：read_status 仅接受 unread/reading/finished")
				return
			}
		}
		fields[k] = v
	}

	if len(fields) == 0 {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.svc.UpdateBook(uint(id), fields); err != nil {
		Error(c, http.StatusNotFound, "书籍不存在")
		return
	}

	Success(c, nil)
}

// Reprocess handles POST /books/reprocess?id=
func (h *BookHandler) Reprocess(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.svc.Reprocess(uint(id)); err != nil {
		msg := err.Error()
		code := http.StatusBadRequest
		if strings.Contains(msg, "不存在") {
			code = http.StatusNotFound
		}
		Error(c, code, msg)
		return
	}

	Success(c, nil)
}

// Delete handles POST /books/delete?id=
func (h *BookHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		Error(c, http.StatusNotFound, "书籍不存在")
		return
	}

	Success(c, nil)
}

// BatchDelete handles POST /books/batch_delete
func (h *BookHandler) BatchDelete(c *gin.Context) {
	var body struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	deleted, err := h.svc.BatchDelete(body.IDs)
	if err != nil {
		Error(c, http.StatusNotFound, "书籍不存在")
		return
	}

	Success(c, gin.H{"deleted": deleted})
}

// Read handles GET /books/read?id=
// Streams the book file content for the online reader.
// Supports HTTP Range requests for EPUB chapter loading and PDF partial reads.
func (h *BookHandler) Read(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	reader, book, err := h.svc.OpenBook(uint(id))
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "不可访问") {
			Error(c, http.StatusForbidden, msg)
		} else {
			Error(c, http.StatusNotFound, "书籍不存在")
		}
		return
	}
	defer reader.Close()

	// Set content type based on file type
	if book.FileType == "pdf" {
		c.Header("Content-Type", "application/pdf")
	} else {
		c.Header("Content-Type", "application/epub+zip")
	}

	c.Header("Accept-Ranges", "bytes")
	// http.ServeContent handles Range headers automatically using Seek.
	// It streams the file without loading it entirely into memory.
	http.ServeContent(c.Writer, c.Request, book.BookFile, time.Time{}, reader)
}

// Download handles GET /books/download?id=
// Triggers browser download (Content-Disposition: attachment).
func (h *BookHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	reader, book, err := h.svc.OpenBook(uint(id))
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "不可访问") {
			Error(c, http.StatusForbidden, msg)
		} else {
			Error(c, http.StatusNotFound, "书籍不存在")
		}
		return
	}
	defer reader.Close()

	ext := ".epub"
	if book.FileType == "pdf" {
		ext = ".pdf"
	}
	filename := book.Title + ext
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/octet-stream")
	// No Range support for downloads — return full file
	c.DataFromReader(http.StatusOK, book.FileSize, "application/octet-stream", reader, nil)
}

// Cover handles GET /books/cover?id=
func (h *BookHandler) Cover(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	reader, book, err := h.svc.OpenCover(uint(id))
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "不存在") {
			Error(c, http.StatusNotFound, msg)
		} else {
			Error(c, http.StatusNotFound, "书籍不存在")
		}
		return
	}
	defer reader.Close()

	contentType := "image/jpeg"
	if strings.HasSuffix(book.CoverFile, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(book.CoverFile, ".gif") {
		contentType = "image/gif"
	}

	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}
