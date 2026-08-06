package model

import "gorm.io/gorm"

// BookTag is the join table for the many-to-many relationship between Book and Tag.
type BookTag struct {
	ID        uint  `json:"id" gorm:"primaryKey"`
	BookID    uint  `json:"book_id" gorm:"not null;index;uniqueIndex:idx_book_tag"`
	TagID     uint  `json:"tag_id" gorm:"not null;index;uniqueIndex:idx_book_tag"`
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime:false"`
}

// BeforeCreate sets CreatedAt to current Unix timestamp.
func (bt *BookTag) BeforeCreate(_ *gorm.DB) error {
	bt.CreatedAt = NowUnix()
	return nil
}
