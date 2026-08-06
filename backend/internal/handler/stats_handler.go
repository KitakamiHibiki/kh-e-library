package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/service"
)

// StatsHandler handles stats API requests.
type StatsHandler struct {
	svc *service.BookService
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(svc *service.BookService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// Overview handles GET /stats/overview
func (h *StatsHandler) Overview(c *gin.Context) {
	stats, err := h.svc.GetStatsOverview()
	if err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	Success(c, stats)
}
