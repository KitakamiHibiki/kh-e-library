package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/repository"
)

type SettingHandler struct {
	repo       *repository.SettingRepository
	onSettings func(map[string]string) // called when settings are saved
}

func NewSettingHandler(repo *repository.SettingRepository, onSettings func(map[string]string)) *SettingHandler {
	return &SettingHandler{repo: repo, onSettings: onSettings}
}

func (h *SettingHandler) ListAll(c *gin.Context) {
	m, err := h.repo.GetMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

func (h *SettingHandler) Update(c *gin.Context) {
	var body struct {
		Settings map[string]string `json:"settings"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.BatchUpsert(body.Settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// trigger hot-reload if storage config was changed
	if h.onSettings != nil {
		if all, err := h.repo.GetMap(); err == nil {
			h.onSettings(all)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
