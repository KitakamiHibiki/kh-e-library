package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check handles GET /health
func (h *HealthHandler) Check(c *gin.Context) {
	// Check SQLite connection
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, Response{Code: 503, Data: nil, Msg: "数据库不可用"})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, Response{Code: 503, Data: nil, Msg: "数据库不可用"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 200, Data: gin.H{"status": "ok"}, Msg: "ok"})
}
