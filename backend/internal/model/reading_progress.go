package model

import "time"

type ReadingProgress struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BookID      uint      `json:"book_id" gorm:"not null;index;uniqueIndex"`
	Book        Book      `json:"-" gorm:"foreignKey:BookID"`
	Progress    float64   `json:"progress"`        // 0.0 ~ 1.0
	CFI         string    `json:"cfi,omitempty"`   // EPUB CFI position
	ChapterHref string    `json:"chapter_href,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Bookmark struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BookID      uint      `json:"book_id" gorm:"not null;index"`
	Book        Book      `json:"-" gorm:"foreignKey:BookID"`
	CFI         string    `json:"cfi"`
	ChapterHref string    `json:"chapter_href,omitempty"`
	ChapterName string    `json:"chapter_name,omitempty"`
	Text        string    `json:"text,omitempty" gorm:"type:text"`
	Note        string    `json:"note,omitempty" gorm:"type:text"`
	Progress    float64   `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
