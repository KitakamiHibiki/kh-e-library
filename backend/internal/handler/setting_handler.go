package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/repository"
)

type SettingHandler struct {
	repo *repository.SettingRepository
}

func NewSettingHandler(repo *repository.SettingRepository) *SettingHandler {
	return &SettingHandler{repo: repo}
}

// ListAll returns all settings merged with defaults
func (h *SettingHandler) ListAll(c *gin.Context) {
	settings, err := h.repo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	m := make(map[string]string)
	for k, v := range model.DefaultSettings {
		m[k] = v
	}
	for _, s := range settings {
		m[s.Key] = s.Value
	}

	c.JSON(http.StatusOK, gin.H{"data": m})
}

// Update batch updates settings
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
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}