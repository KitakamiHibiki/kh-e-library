package model

import "gorm.io/gorm"

// ReadingProgress stores the reading progress for a single book.
// Each book has at most one progress record (BookID is unique).
type ReadingProgress struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	BookID      uint    `json:"book_id" gorm:"not null;uniqueIndex"`
	Progress    float64 `json:"progress"`         // 0.0 ~ 1.0
	CFI         string  `json:"cfi"`              // EPUB CFI position / PDF page number string
	ChapterHref string  `json:"chapter_href"`     // current chapter
	CreatedAt   int64   `json:"created_at" gorm:"autoCreateTime:false"`
	UpdatedAt   int64   `json:"updated_at" gorm:"autoUpdateTime:false"`
}

// BeforeCreate sets CreatedAt and UpdatedAt.
func (rp *ReadingProgress) BeforeCreate(_ *gorm.DB) error {
	now := NowUnix()
	rp.CreatedAt = now
	rp.UpdatedAt = now
	return nil
}

// BeforeUpdate sets UpdatedAt.
func (rp *ReadingProgress) BeforeUpdate(_ *gorm.DB) error {
	rp.UpdatedAt = NowUnix()
	return nil
}
