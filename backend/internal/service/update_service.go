package service

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kitakami-hibiki/e-library/internal/config"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/repository"
)

// UpdateService handles software update checks, downloads, and installation.
// All operations are user-initiated — there is no automatic update behavior.
type UpdateService struct {
	settingRepo *repository.SettingRepository
	httpClient  *http.Client
}

// NewUpdateService creates a new UpdateService.
func NewUpdateService(settingRepo *repository.SettingRepository) *UpdateService {
	return &UpdateService{
		settingRepo: settingRepo,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

// UpdateDir returns the directory where downloaded update packages are stored.
func (s *UpdateService) UpdateDir() string {
	return filepath.Join(config.DataDir(), "update")
}

// GetSettingGithubRepo reads the configured GitHub repo from settings.
func (s *UpdateService) GetSettingGithubRepo() string {
	return s.settingRepo.GetWithDefault(model.SettingGithubRepo)
}

// NewFileName is the name used for the staged new binary inside the exe dir.
func (s *UpdateService) NewFileName() string {
	if runtime.GOOS == "windows" {
		return "kh-e-library.new.exe"
	}
	return "kh-e-library.new"
}

// CheckGitHubRelease queries the GitHub Releases API for the latest release of
// the given repo and compares it against the current running version.
func (s *UpdateService) CheckGitHubRelease(ctx context.Context, githubRepo, currentVersion string) (*model.UpdateCheckResult, error) {
	if githubRepo == "" {
		return nil, fmt.Errorf("update.github_repo 未配置")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "kh-e-library-update-check")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接更新源: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, fmt.Errorf("未找到仓库 %s 的发布版本", githubRepo)
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, fmt.Errorf("GitHub API 访问受限（可能触发限流），请稍后再试")
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("GitHub API 返回异常状态码 %d", resp.StatusCode)
	}

	var release model.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析更新信息失败: %w", err)
	}

	result := &model.UpdateCheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}

	// Skip prerelease tags for the stable channel.
	if release.Prerelease {
		result.LatestVersion = release.TagName
		result.HasUpdate = false
		return result, nil
	}

	if compareVersions(release.TagName, currentVersion) > 0 {
		result.HasUpdate = true
		if asset := s.pickAsset(release.Assets); asset != nil {
			result.DownloadURL = asset.BrowserDownloadURL
			result.FileName = asset.Name
			result.FileSize = asset.Size
		}
	}
	return result, nil
}

// pickAsset selects the release asset best matching the current OS/arch.
func (s *UpdateService) pickAsset(assets []model.GitHubAsset) *model.GitHubAsset {
	if len(assets) == 0 {
		return nil
	}
	goos, goarch := runtime.GOOS, runtime.GOARCH

	// 1. Prefer an exact GOOS/GOARCH match (e.g. kh-e-library-v0.1.0-windows-amd64.tar.gz)
	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if strings.Contains(name, goos) && strings.Contains(name, goarch) {
			return &assets[i]
		}
	}
	// 2. Windows fallback: any .exe
	if goos == "windows" {
		for i := range assets {
			if strings.HasSuffix(strings.ToLower(assets[i].Name), ".exe") {
				return &assets[i]
			}
		}
	}
	// 3. Otherwise: the first asset
	return &assets[0]
}

// Download fetches the update package from the given URL into the update dir.
// Returns the saved file path and its size in bytes.
func (s *UpdateService) Download(ctx context.Context, url string) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "kh-e-library-update-download")

	// GitHub asset downloads redirect to a CDN; the default client follows.
	// A generous timeout is needed for large binaries.
	dlClient := &http.Client{Timeout: 30 * time.Minute}
	resp, err := dlClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("下载失败：HTTP %d", resp.StatusCode)
	}

	dir := s.UpdateDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, err
	}

	name := filepath.Base(url)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "update.pkg"
	}
	dest := filepath.Join(dir, name)

	f, err := os.Create(dest)
	if err != nil {
		return "", 0, err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(dest)
		return "", 0, fmt.Errorf("保存文件失败: %w", copyErr)
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	return dest, n, nil
}

// Install prepares and launches the updater: it resolves the downloaded package
// to a binary, stages it next to the running executable, writes the platform
// updater script, and launches it detached. The running process is expected to
// exit shortly after via the shutdown trigger.
func (s *UpdateService) Install() (string, error) {
	pkg, err := s.findDownloadedPackage()
	if err != nil {
		return "", err
	}

	binary, err := s.resolveBinary(pkg)
	if err != nil {
		return "", err
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法定位当前程序路径: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Stage the new binary next to the running executable.
	newPath := filepath.Join(exeDir, s.NewFileName())
	if err := copyFile(binary, newPath); err != nil {
		return "", fmt.Errorf("复制更新包失败: %w", err)
	}

	// Write the platform updater script and launch it detached.
	updaterPath, err := s.WriteUpdaterScript(exeDir, filepath.Base(exePath), filepath.Base(newPath))
	if err != nil {
		return "", err
	}
	if err := s.LaunchUpdater(updaterPath); err != nil {
		return "", err
	}
	return newPath, nil
}

// findDownloadedPackage returns the most recently modified update package in the update dir.
func (s *UpdateService) findDownloadedPackage() (string, error) {
	dir := s.UpdateDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("尚未下载更新包，请先执行下载")
		}
		return "", err
	}

	var best string
	var bestMod time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "kh-e-library") &&
			!strings.HasSuffix(name, ".exe") &&
			!strings.HasSuffix(name, ".tar.gz") &&
			!strings.HasSuffix(name, ".tgz") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if fi.ModTime().After(bestMod) {
			bestMod = fi.ModTime()
			best = filepath.Join(dir, name)
		}
	}
	if best == "" {
		return "", fmt.Errorf("未找到已下载的更新包，请先执行下载")
	}
	return best, nil
}

// resolveBinary returns the actual executable within an update package,
// extracting archives (tar.gz / tgz / zip) when necessary.
func (s *UpdateService) resolveBinary(pkgPath string) (string, error) {
	lower := strings.ToLower(pkgPath)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return s.extractTarArchive(pkgPath)
	}
	if strings.HasSuffix(lower, ".zip") {
		return s.extractZipArchive(pkgPath)
	}
	return pkgPath, nil
}

// extractTarArchive unpacks a tar.gz update package and returns the path of the
// first matching executable inside it.
func (s *UpdateService) extractTarArchive(pkgPath string) (string, error) {
	outDir := filepath.Join(filepath.Dir(pkgPath), "extracted")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	f, err := os.Open(pkgPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("无法解压更新包: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("解析更新包失败: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		name := filepath.Base(hdr.Name)
		if !strings.HasSuffix(name, ".exe") && name != "kh-e-library" {
			continue
		}
		dest := filepath.Join(outDir, name)
		if err := writeTarFile(dest, tr, hdr.Mode); err != nil {
			return "", err
		}
		return dest, nil
	}
	return "", fmt.Errorf("更新包中未找到可执行文件")
}

// extractZipArchive unpacks a zip update package and returns the path of the
// first matching executable inside it.
func (s *UpdateService) extractZipArchive(pkgPath string) (string, error) {
	outDir := filepath.Join(filepath.Dir(pkgPath), "extracted")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	zr, err := zip.OpenReader(pkgPath)
	if err != nil {
		return "", fmt.Errorf("无法解压更新包: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)
		if !strings.HasSuffix(name, ".exe") && name != "kh-e-library" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		dest := filepath.Join(outDir, name)
		if err := writeTarFile(dest, rc, int64(f.Mode().Perm())); err != nil {
			rc.Close()
			return "", err
		}
		rc.Close()
		return dest, nil
	}
	return "", fmt.Errorf("更新包中未找到可执行文件")
}

// copyFile copies src to dst, preserving the executable bit on unix.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	info, err := os.Stat(src)
	if err == nil {
		os.Chmod(dst, info.Mode().Perm())
	}
	return nil
}

// writeTarFile writes a single tar entry to disk.
func writeTarFile(dest string, r io.Reader, mode int64) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(mode)&0777)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		os.Remove(dest)
		return err
	}
	return out.Close()
}

// compareVersions compares semantic-ish versions ("v1.2.3"). Returns -1/0/1.
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

// versionParts parses "v1.2.3" (or "1.2.3") into [major, minor, patch] ints.
// Unparseable segments (e.g. "dev") are treated as 0.
func versionParts(v string) [3]int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	var parts [3]int
	segs := strings.Split(v, ".")
	for i := 0; i < len(segs) && i < 3; i++ {
		num := ""
		for _, r := range segs[i] {
			if r < '0' || r > '9' {
				break
			}
			num += string(r)
		}
		if n, err := strconv.Atoi(num); err == nil {
			parts[i] = n
		}
	}
	return parts
}
