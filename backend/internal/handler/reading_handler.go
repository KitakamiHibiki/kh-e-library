package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/service"
)

// ReadingHandler handles reading progress API requests.
type ReadingHandler struct {
	svc *service.BookService
}

// NewReadingHandler creates a new ReadingHandler.
func NewReadingHandler(svc *service.BookService) *ReadingHandler {
	return &ReadingHandler{svc: svc}
}

// GetProgress handles GET /books/progress?id=
func (h *ReadingHandler) GetProgress(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	progress, err := h.svc.GetProgress(uint(id))
	if err != nil {
		Success(c, nil)
		return
	}

	Success(c, progress)
}

// SaveProgress handles POST /books/progress/save
func (h *ReadingHandler) SaveProgress(c *gin.Context) {
	var p model.ReadingProgress
	if err := c.ShouldBindJSON(&p); err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// Validate required fields
	if p.BookID == 0 {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// Validate progress range (0.0 ~ 1.0)
	if p.Progress < 0 || p.Progress > 1 {
		Error(c, http.StatusBadRequest, "校验失败：progress 仅接受 0.0~1.0")
		return
	}

	if err := h.svc.SaveProgress(&p); err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	Success(c, p)
}
