package model

import (
	"gorm.io/gorm"
	"time"
)

// Setting stores a runtime configuration key-value pair.
type Setting struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Key       string `json:"key" gorm:"uniqueIndex;not null;size:128"`
	Value     string `json:"value" gorm:"type:text"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime:false"`
}

// BeforeUpdate sets UpdatedAt.
func (s *Setting) BeforeUpdate(_ *gorm.DB) error {
	s.UpdatedAt = time.Now().Unix()
	return nil
}

// Predefined setting keys
const (
	SettingTheme           = "ui.theme"
	SettingPageSize        = "ui.page_size"
	SettingSortField       = "ui.sort_field"
	SettingSortOrder       = "ui.sort_order"
	SettingFontSize        = "reader.font_size"
	SettingPdfViewMode     = "reader.pdf_view_mode"
	SettingEpubViewMode    = "reader.epub_view_mode"
	SettingStorageDriver   = "storage.driver"
	SettingStorageBooksDir = "storage.local.books_dir"
	SettingGithubRepo      = "update.github_repo"
)

// DefaultSettings maps predefined keys to their default values.
var DefaultSettings = map[string]string{
	SettingTheme:           "light",
	SettingPageSize:        "20",
	SettingSortField:       "updated_at",
	SettingSortOrder:       "desc",
	SettingFontSize:        "16",
	SettingPdfViewMode:     "single",
	SettingEpubViewMode:    "single",
	SettingStorageDriver:   "local",
	SettingStorageBooksDir: "./data/books",
	// 默认检查原作者发布仓库；fork / 私有部署可在设置页改为自己的仓库。
	// 数据库中已保存的值优先于该默认值。
	SettingGithubRepo: "KitakamiHibiki/kh-e-library",
}

// NowUnix returns the current UTC time as a Unix timestamp in seconds.
func NowUnix() int64 {
	return time.Now().Unix()
}
