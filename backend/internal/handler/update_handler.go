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

// StartUpdate handles POST /system/start-update with body {"download_url": "..."}.
// It kicks off an async download+install task and returns immediately; the
// client polls GET /system/update-status to track progress.
func (h *UpdateHandler) StartUpdate(c *gin.Context) {
	var body struct {
		DownloadURL string `json:"download_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.DownloadURL == "" {
		Error(c, http.StatusBadRequest, "请求参数无效")
		return
	}

	h.svc.StartUpdate(body.DownloadURL)

	// After the async task finishes installing, trigger the graceful shutdown
	// so the updater script can swap the binary and restart. A failed task must
	// NOT restart the server — the user can retry from the UI.
	if h.shutdown != nil {
		go func() {
			for {
				status := h.svc.GetUpdateStatus()
				if status.State == model.UpdateCompleted || status.State == model.UpdateFailed {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			if h.svc.GetUpdateStatus().State == model.UpdateCompleted {
				// Let the client see the completed state before we drain.
				time.Sleep(500 * time.Millisecond)
				h.shutdown()
			}
		}()
	}

	Success(c, gin.H{"status": "started"})
}

// GetUpdateStatus handles GET /system/update-status — returns the current
// async update task snapshot for polling.
func (h *UpdateHandler) GetUpdateStatus(c *gin.Context) {
	Success(c, h.svc.GetUpdateStatus())
}
