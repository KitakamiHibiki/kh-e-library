package model

import "time"

type Setting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null;size:128"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Runtime setting keys
const (
	SettingTheme           = "ui.theme"
	SettingPageSize        = "ui.page_size"
	SettingSortField       = "ui.sort_field"
	SettingSortOrder       = "ui.sort_order"
	SettingFontSize        = "reader.font_size"
	SettingStorageDriver   = "storage.driver"
	SettingStorageBooksDir = "storage.local.books_dir"
	SettingBaiduClientID   = "storage.baidu.client_id"
	SettingBaiduSecret     = "storage.baidu.client_secret"
	SettingBaiduToken      = "storage.baidu.refresh_token"
)

var DefaultSettings = map[string]string{
	SettingTheme:           "light",
	SettingPageSize:        "20",
	SettingSortField:       "updated_at",
	SettingSortOrder:       "desc",
	SettingFontSize:        "16",
	SettingStorageDriver:   "local",
	SettingStorageBooksDir: "./data/books",
	SettingBaiduClientID:   "",
	SettingBaiduSecret:     "",
	SettingBaiduToken:      "",
}