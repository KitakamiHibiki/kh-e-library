 package model

 import "time"

 type Book struct {
 	ID          uint      `json:"id" gorm:"primaryKey"`
 	Title       string    `json:"title" gorm:"not null;index"`
 	Author      string    `json:"author" gorm:"index"`
 	Publisher   string    `json:"publisher,omitempty"`
 	ISBN        string    `json:"isbn,omitempty" gorm:"uniqueIndex"`
 	CoverPath   string    `json:"cover_path,omitempty"`
 	FilePath    string    `json:"file_path" gorm:"not null"`
 	FileSize    int64     `json:"file_size"`
 	Description string    `json:"description,omitempty" gorm:"type:text"`
 	Language    string    `json:"language,omitempty"`
 	Tags        string    `json:"tags,omitempty"`
 	Pages       int       `json:"pages,omitempty"`
 	StorageKey  string    `json:"storage_key" gorm:"default:local"` // "local" | "baidu"
 	CreatedAt   time.Time `json:"created_at"`
 	UpdatedAt   time.Time `json:"updated_at"`
 }
