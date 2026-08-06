package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/repository"
)

// BookTagHandler handles book-tag association API requests.
type BookTagHandler struct {
	bookRepo *repository.BookRepository
}

// NewBookTagHandler creates a new BookTagHandler.
func NewBookTagHandler(bookRepo *repository.BookRepository) *BookTagHandler {
	return &BookTagHandler{bookRepo: bookRepo}
}

// Add handles POST /books/tags/add
func (h *BookTagHandler) Add(c *gin.Context) {
	var body struct {
		BookID uint `json:"book_id"`
		TagID  uint `json:"tag_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.BookID == 0 || body.TagID == 0 {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.bookRepo.AddTag(body.BookID, body.TagID); err != nil {
		Error(c, http.StatusConflict, "书籍-标签关联已存在")
		return
	}

	Success(c, nil)
}

// Remove handles POST /books/tags/remove
func (h *BookTagHandler) Remove(c *gin.Context) {
	var body struct {
		BookID uint `json:"book_id"`
		TagID  uint `json:"tag_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.BookID == 0 || body.TagID == 0 {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.bookRepo.RemoveTag(body.BookID, body.TagID); err != nil {
		Error(c, http.StatusNotFound, "书籍-标签关联不存在")
		return
	}

	Success(c, nil)
}
