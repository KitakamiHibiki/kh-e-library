package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/repository"
)

// SettingHandler handles setting-related API requests.
type SettingHandler struct {
	repo       *repository.SettingRepository
	onSettings func(changed map[string]string) error
}

// NewSettingHandler creates a new SettingHandler.
func NewSettingHandler(repo *repository.SettingRepository, onSettings func(changed map[string]string) error) *SettingHandler {
	return &SettingHandler{repo: repo, onSettings: onSettings}
}

// List handles GET /settings/list
func (h *SettingHandler) List(c *gin.Context) {
	m, err := h.repo.GetMap()
	if err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}
	Success(c, m)
}

// Update handles POST /settings/update
func (h *SettingHandler) Update(c *gin.Context) {
	var body struct {
		Settings map[string]string `json:"settings"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Settings) == 0 {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// Validate all keys and values
	if err := validateSettings(body.Settings); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Step 1: Save old values for rollback
	oldValues := make(map[string]string)
	for key := range body.Settings {
		oldValues[key] = h.repo.GetWithDefault(key)
	}

	// Step 2: Write new values to DB
	if err := h.repo.BatchUpsert(body.Settings); err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	// Step 3: Call onSettings callback
	if h.onSettings != nil {
		if err := h.onSettings(body.Settings); err != nil {
			// Step 4: Rollback on failure
			h.repo.BatchRollback(oldValues)
			Error(c, http.StatusInternalServerError, "服务器内部错误")
			return
		}
	}

	Success(c, nil)
}

// validateSettings checks that all keys are predefined and values are valid.
func validateSettings(settings map[string]string) error {
	for key, value := range settings {
		switch key {
		case model.SettingTheme:
			if value != "light" && value != "dark" && value != "auto" {
				return fmt.Errorf("校验失败：ui.theme 仅接受 light/dark/auto")
			}
		case model.SettingPageSize:
			n, err := strconv.Atoi(value)
			if err != nil || n < 5 || n > 100 {
				return fmt.Errorf("校验失败：ui.page_size 仅接受 5~100 的整数")
			}
		case model.SettingSortField:
			if value != "updated_at" && value != "title" && value != "author" && value != "created_at" {
				return fmt.Errorf("校验失败：ui.sort_field 仅接受 updated_at/title/author/created_at")
			}
		case model.SettingSortOrder:
			if value != "asc" && value != "desc" {
				return fmt.Errorf("校验失败：ui.sort_order 仅接受 asc/desc")
			}
		case model.SettingFontSize:
			n, err := strconv.Atoi(value)
			if err != nil || n < 12 || n > 32 {
				return fmt.Errorf("校验失败：reader.font_size 仅接受 12~32 的整数")
			}
		case model.SettingPdfViewMode:
			if value != "single" && value != "double" && value != "scroll" {
				return fmt.Errorf("校验失败：reader.pdf_view_mode 仅接受 single/double/scroll")
			}
		case model.SettingEpubViewMode:
			if value != "single" && value != "double" {
				return fmt.Errorf("校验失败：reader.epub_view_mode 仅接受 single/double")
			}
		case model.SettingStorageDriver:
			if value != "local" {
				return fmt.Errorf("校验失败：storage.driver 当前仅接受 local")
			}
		case model.SettingStorageBooksDir:
			if value == "" {
				return fmt.Errorf("校验失败：storage.local.books_dir 不能为空")
			}
		default:
			return fmt.Errorf("校验失败：未知设置项 %s", key)
		}
	}
	return nil
}
