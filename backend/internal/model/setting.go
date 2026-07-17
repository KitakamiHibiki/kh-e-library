package model

import "time"

// Setting 运行时设置，key-value 存储，修改后立即生效无需重启
type Setting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null;size:128"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Predefined setting keys
const (
	SettingTheme       = "ui.theme"        // light | dark | auto
	SettingPageSize    = "ui.page_size"    // 每页书籍数
	SettingSortField   = "ui.sort_field"   // title | author | created_at
	SettingSortOrder   = "ui.sort_order"   // asc | desc
	SettingFontSize    = "reader.font_size"
)

var DefaultSettings = map[string]string{
	SettingTheme:     "light",
	SettingPageSize:  "20",
	SettingSortField: "updated_at",
	SettingSortOrder: "desc",
	SettingFontSize:  "16",
}