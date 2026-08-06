package model

import "gorm.io/gorm"

// Book represents an e-book in the library.
// CoverFile and BookFile store filenames only (e.g. "cover.jpg", "book.epub"),
// not absolute paths. The actual file path is constructed by the storage driver
// as {BooksDir}/{bookID}/{filename}.
type Book struct {
	ID          uint     `json:"id" gorm:"primaryKey"`
	Title       string   `json:"title" gorm:"not null;index"`
	Author      string   `json:"author" gorm:"index"`
	Publisher   string   `json:"publisher"`
	ISBN        string   `json:"isbn" gorm:"index"`
	CoverFile   string   `json:"cover"`                          // filename, e.g. "cover.jpg"; empty string = no cover
	BookFile    string   `json:"file" gorm:"column:file;not null"` // filename, e.g. "book.epub"
	FileType    string   `json:"file_type" gorm:"not null"`      // "epub" | "pdf"
	FileSize    int64    `json:"file_size"`
	FileHash    string   `json:"file_hash" gorm:"uniqueIndex"`   // SHA-256 hash for duplicate detection
	Description string   `json:"description" gorm:"type:text"`
	Language    string   `json:"language"`
	Pages       int      `json:"pages"`                          // EPUB: spine itemref count; PDF: page count
	StorageKey  string   `json:"storage_key" gorm:"default:local"`
	ReadStatus  string   `json:"read_status" gorm:"default:unread"`    // unread | reading | finished
	BookStatus  string   `json:"book_status" gorm:"default:processing"` // processing | failed | ready | deleted
	CreatedAt   int64    `json:"created_at" gorm:"autoCreateTime:false"`
	UpdatedAt   int64    `json:"updated_at" gorm:"autoUpdateTime:false"`
	Tags        []string `json:"tags" gorm:"-"` // computed field, populated via join query
}

// BeforeCreate sets CreatedAt and UpdatedAt to current Unix timestamp.
func (b *Book) BeforeCreate(_ *gorm.DB) error {
	now := NowUnix()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

// BeforeUpdate sets UpdatedAt to current Unix timestamp.
func (b *Book) BeforeUpdate(_ *gorm.DB) error {
	b.UpdatedAt = NowUnix()
	return nil
}
