package model

// UpdateTaskState is the lifecycle state of the async update task.
type UpdateTaskState string

const (
	// UpdateIdle means no update task is running or has run.
	UpdateIdle UpdateTaskState = "idle"
	// UpdateDownloading means the package is being downloaded.
	UpdateDownloading UpdateTaskState = "downloading"
	// UpdateInstalling means the package is being staged and the updater launched.
	UpdateInstalling UpdateTaskState = "installing"
	// UpdateCompleted means the updater was launched and the server will restart.
	UpdateCompleted UpdateTaskState = "completed"
	// UpdateFailed means the task errored out.
	UpdateFailed UpdateTaskState = "failed"
)

// UpdateTask is the pollable status of the async download+install task.
type UpdateTask struct {
	State    UpdateTaskState `json:"state"`
	Message  string          `json:"message"`
	Progress int             `json:"progress"` // 0-100 during download
}

// UpdateCheckResult is the response payload for check-update.
type UpdateCheckResult struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	ReleaseNotes   string `json:"release_notes"`
	DownloadURL    string `json:"download_url"`
	FileName       string `json:"file_name"`
	FileSize       int64  `json:"file_size"`
	PublishedAt    string `json:"published_at"`
	ReleaseURL     string `json:"release_url"`
}

// SystemStatus is the payload for GET /system/status.
type SystemStatus struct {
	Version    string `json:"version"`
	Platform   string `json:"platform"`
	Executable string `json:"executable"`
}

// GitHubRelease mirrors the subset of the GitHub Releases API "latest release"
// response that we consume. Only tagged releases (non-draft, non-prerelease
// for the "stable" channel) are considered.
type GitHubRelease struct {
	TagName     string            `json:"tag_name"`
	Body        string            `json:"body"`
	PublishedAt string            `json:"published_at"`
	Prerelease  bool              `json:"prerelease"`
	Assets      []GitHubAsset     `json:"assets"`
	HTMLURL     string            `json:"html_url"`
}

// GitHubAsset mirrors a single release asset entry.
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}
