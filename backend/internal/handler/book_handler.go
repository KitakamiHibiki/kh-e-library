package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/config"
	"github.com/kitakami-hibiki/e-library/internal/epub"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/service"
)

type BookHandler struct {
	svc *service.BookService
}

func NewBookHandler(svc *service.BookService) *BookHandler {
	return &BookHandler{svc: svc}
}

func (h *BookHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	books, total, err := h.svc.List(page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  books,
		"total": total,
		"page":  page,
	})
}

func (h *BookHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	book, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": book})
}

func (h *BookHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer f.Close()

	book, err := h.svc.Upload(file.Filename, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// parse EPUB metadata and extract cover
	go func() {
		data, err := os.ReadFile(book.FilePath)
		if err != nil {
			return
		}
		meta, parseErr := epub.Parse(bytes.NewReader(data), int64(len(data)))
		if parseErr == nil {
			book.Title = meta.Title
			if len(meta.Authors) > 0 {
				book.Author = meta.Authors[0]
			}
			book.Publisher = meta.Publisher
			book.ISBN = meta.Identifier
			book.Language = meta.Language
			book.Description = meta.Description
			book.Pages = len(meta.Spine)
			h.svc.Update(book)
		}
		coverData, ext, coverErr := epub.ReadCover(bytes.NewReader(data), int64(len(data)))
		if coverErr == nil {
			coversDir := filepath.Join(filepath.Dir(config.DatabasePath()), "covers")
			os.MkdirAll(coversDir, 0755)
			book.CoverPath = filepath.Join(coversDir, fmt.Sprintf("%d%s", book.ID, ext))
			os.WriteFile(book.CoverPath, coverData, 0644)
			h.svc.Update(book)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{"data": book})
}

func (h *BookHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var book model.Book
	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	book.ID = uint(id)
	if err := h.svc.Update(&book); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": book})
}

func (h *BookHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *BookHandler) Read(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	book, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	reader, err := h.svc.OpenBook(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	defer reader.Close()
	c.DataFromReader(http.StatusOK, book.FileSize, "application/epub+zip", reader, nil)
}

func (h *BookHandler) Cover(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	book, err := h.svc.GetByID(uint(id))
	if err != nil || book.CoverPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "cover not found"})
		return
	}
	c.File(book.CoverPath)
}
