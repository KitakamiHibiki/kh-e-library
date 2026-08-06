package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/repository"
)

// TagHandler handles tag-related API requests.
type TagHandler struct {
	repo *repository.TagRepository
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(repo *repository.TagRepository) *TagHandler {
	return &TagHandler{repo: repo}
}

// List handles GET /tags/list
func (h *TagHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")

	tags, err := h.repo.List(keyword)
	if err != nil {
		Error(c, http.StatusInternalServerError, "服务器内部错误")
		return
	}

	Success(c, tags)
}

// Create handles POST /tags/create
func (h *TagHandler) Create(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	tag, err := h.repo.Create(body.Name)
	if err != nil {
		// Check for unique constraint violation
		Error(c, http.StatusConflict, "标签名称已存在")
		return
	}

	SuccessWithStatus(c, http.StatusCreated, tag)
}

// Update handles POST /tags/update?id=
func (h *TagHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.repo.Update(uint(id), body.Name); err != nil {
		Error(c, http.StatusConflict, "标签名称已存在")
		return
	}

	Success(c, nil)
}

// Delete handles POST /tags/delete?id=
func (h *TagHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	if err := h.repo.Delete(uint(id)); err != nil {
		Error(c, http.StatusNotFound, "标签不存在")
		return
	}

	Success(c, nil)
}
