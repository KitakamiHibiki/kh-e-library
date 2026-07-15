package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/service"
)

type ReadingHandler struct {
	svc *service.BookService
}

func NewReadingHandler(svc *service.BookService) *ReadingHandler {
	return &ReadingHandler{svc: svc}
}

func (h *ReadingHandler) GetProgress(c *gin.Context) {
	bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	progress, err := h.svc.GetProgress(uint(bookID))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": progress})
}

func (h *ReadingHandler) SaveProgress(c *gin.Context) {
	var p model.ReadingProgress
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SaveProgress(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func (h *ReadingHandler) ListBookmarks(c *gin.Context) {
	bookID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	bookmarks, err := h.svc.ListBookmarks(uint(bookID))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []model.Bookmark{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bookmarks})
}

func (h *ReadingHandler) CreateBookmark(c *gin.Context) {
	var b model.Bookmark
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateBookmark(&b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": b})
}

func (h *ReadingHandler) DeleteBookmark(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.DeleteBookmark(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
