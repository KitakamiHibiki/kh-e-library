package handler

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/service"
)

// UpdateHandler handles software update-related API requests.
// Every endpoint requires an explicit user action — there is no automatic update.
type UpdateHandler struct {
	svc      *service.UpdateService
	version  string
	shutdown func()
}

// NewUpdateHandler creates a new UpdateHandler.
// shutdown is invoked after a successful install to trigger the graceful
// shutdown in main.go (which then lets the updater script take over).
func NewUpdateHandler(svc *service.UpdateService, version string, shutdown func()) *UpdateHandler {
	return &UpdateHandler{svc: svc, version: version, shutdown: shutdown}
}

// Status handles GET /system/status — returns current version and platform info.
// This is a pure read of the running binary, no network access.
func (h *UpdateHandler) Status(c *gin.Context) {
	exe, _ := os.Executable()
	Success(c, model.SystemStatus{
		Version:    h.version,
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
		Executable: exe,
	})
}

// Check handles GET /system/check-update?repo=owner/repo
// The optional repo query param overrides the configured update.github_repo.
func (h *UpdateHandler) Check(c *gin.Context) {
	githubRepo := c.Query("repo")
	if githubRepo == "" {
		githubRepo = h.svc.GetSettingGithubRepo()
	}
	if githubRepo == "" {
		Error(c, http.StatusBadRequest, "未配置 GitHub 仓库，请在设置中填写 update.github_repo")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	result, err := h.svc.CheckGitHubRelease(ctx, githubRepo, h.version)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	Success(c, result)
}

// Download handles POST /system/download-update with body {"download_url": "..."}.
// The backend proxies the download so the whole package lands in data/update/.
func (h *UpdateHandler) Download(c *gin.Context) {
	var body struct {
		DownloadURL string `json:"download_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.DownloadURL == "" {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	// Downloads of a large binary can take a while; use the request context.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	filePath, size, err := h.svc.Download(ctx, body.DownloadURL)
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, gin.H{
		"status":    "downloaded",
		"file_path": filePath,
		"file_size": size,
	})
}

// Install handles POST /system/install-update.
// It stages the downloaded package next to the running exe, launches the
// updater script detached, responds to the client, then triggers a graceful
// shutdown so the updater can swap the binary and restart.
func (h *UpdateHandler) Install(c *gin.Context) {
	newPath, err := h.svc.Install()
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, gin.H{
		"status":    "installing",
		"new_file":  newPath,
	})

	// Let the HTTP response flush, then shut down gracefully.
	if h.shutdown != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			h.shutdown()
		}()
	}
}
