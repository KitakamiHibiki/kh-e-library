package repository

import (
	"errors"
	"fmt"

	"github.com/kitakami-hibiki/e-library/internal/model"
	"gorm.io/gorm"
)

// BookRepository provides data access for books and reading progress.
type BookRepository struct {
	db *gorm.DB
}

// NewBookRepository creates a new BookRepository.
func NewBookRepository(db *gorm.DB) *BookRepository {
	return &BookRepository{db: db}
}

// List returns a paginated list of books with optional filtering.
// sortField and sortOrder are validated against a whitelist.
func (r *BookRepository) List(page, pageSize int, keyword, tag, bookStatus, sortField, sortOrder string) ([]model.Book, int64, error) {
	var books []model.Book
	var total int64

	query := r.db.Model(&model.Book{})

	// Keyword filter: search title and author
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR author LIKE ?", like, like)
	}

	// Book status filter
	if bookStatus != "" {
		query = query.Where("book_status = ?", bookStatus)
	}

	// Tag filter: join through book_tags + tags
	if tag != "" {
		query = query.Joins("JOIN book_tags ON book_tags.book_id = books.id").
			Joins("JOIN tags ON book_tags.tag_id = tags.id").
			Where("tags.name = ?", tag)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting with whitelist
	sortField = sanitizeSortField(sortField)
	sortOrder = sanitizeSortOrder(sortOrder)
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&books).Error; err != nil {
		return nil, 0, err
	}

	// Batch-load tags for all books
	if len(books) > 0 {
		bookIDs := make([]uint, len(books))
		for i, b := range books {
			bookIDs[i] = b.ID
		}
		tagsMap, err := r.batchLoadTags(bookIDs)
		if err != nil {
			return nil, 0, err
		}
		for i := range books {
			books[i].Tags = tagsMap[books[i].ID]
		}
	}

	return books, total, nil
}

// GetByID returns a single book by ID with tags loaded.
func (r *BookRepository) GetByID(id uint) (*model.Book, error) {
	var book model.Book
	if err := r.db.First(&book, id).Error; err != nil {
		return nil, err
	}
	tags, err := r.loadTags(book.ID)
	if err != nil {
		return nil, err
	}
	book.Tags = tags
	return &book, nil
}

// Create inserts a new book record.
func (r *BookRepository) Create(book *model.Book) error {
	return r.db.Create(book).Error
}

// UpdateFields performs a partial update using a map of fields.
// This avoids zero-value overwrites that DB.Save would cause.
// Returns gorm.ErrRecordNotFound if no row was updated.
func (r *BookRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&model.Book{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetByHash finds a book by its file hash. Returns nil if not found.
func (r *BookRepository) GetByHash(hash string) (*model.Book, error) {
	var book model.Book
	if err := r.db.Where("file_hash = ?", hash).First(&book).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &book, nil
}

// UpdateBookStatus atomically updates book_status with a condition check.
// Returns the number of rows affected (0 means the condition was not met).
func (r *BookRepository) UpdateBookStatus(id uint, oldStatus, newStatus string) (int64, error) {
	result := r.db.Model(&model.Book{}).
		Where("id = ? AND book_status = ?", id, oldStatus).
		Update("book_status", newStatus)
	return result.RowsAffected, result.Error
}

// Delete removes a book record by ID.
func (r *BookRepository) Delete(id uint) error {
	return r.db.Delete(&model.Book{}, id).Error
}

// AddTag creates a book-tag association.
func (r *BookRepository) AddTag(bookID, tagID uint) error {
	bt := model.BookTag{BookID: bookID, TagID: tagID}
	return r.db.Create(&bt).Error
}

// RemoveTag deletes a book-tag association.
func (r *BookRepository) RemoveTag(bookID, tagID uint) error {
	return r.db.Where("book_id = ? AND tag_id = ?", bookID, tagID).Delete(&model.BookTag{}).Error
}

// DeleteBookTags removes all tag associations for a book.
func (r *BookRepository) DeleteBookTags(bookID uint) error {
	return r.db.Where("book_id = ?", bookID).Delete(&model.BookTag{}).Error
}

// GetProgress returns the reading progress for a book.
// Returns nil (not an error) when no progress record exists.
func (r *BookRepository) GetProgress(bookID uint) (*model.ReadingProgress, error) {
	var p model.ReadingProgress
	if err := r.db.Where("book_id = ?", bookID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpsertProgress inserts or updates reading progress using SQLite UPSERT.
// This preserves CreatedAt on updates.
func (r *BookRepository) UpsertProgress(p *model.ReadingProgress) error {
	now := nowUnix()
	return r.db.Exec(`
		INSERT INTO reading_progresses (book_id, progress, cfi, chapter_href, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(book_id) DO UPDATE SET
			progress = EXCLUDED.progress,
			cfi = EXCLUDED.cfi,
			chapter_href = EXCLUDED.chapter_href,
			updated_at = EXCLUDED.updated_at
	`, p.BookID, p.Progress, p.CFI, p.ChapterHref, now, now).Error
}

// DeleteProgress removes the reading progress for a book.
func (r *BookRepository) DeleteProgress(bookID uint) error {
	return r.db.Where("book_id = ?", bookID).Delete(&model.ReadingProgress{}).Error
}

// GetStatsOverview returns reading statistics.
func (r *BookRepository) GetStatsOverview() (totalBooks, unread, reading, finished int64, recentReading []model.RecentReadingItem, err error) {
	if err = r.db.Model(&model.Book{}).Where("book_status = ?", "ready").Count(&totalBooks).Error; err != nil {
		return
	}
	if err = r.db.Model(&model.Book{}).Where("book_status = ? AND read_status = ?", "ready", "unread").Count(&unread).Error; err != nil {
		return
	}
	if err = r.db.Model(&model.Book{}).Where("book_status = ? AND read_status = ?", "ready", "reading").Count(&reading).Error; err != nil {
		return
	}
	if err = r.db.Model(&model.Book{}).Where("book_status = ? AND read_status = ?", "ready", "finished").Count(&finished).Error; err != nil {
		return
	}

	// Recent reading: 5 books with progress, ordered by progress.updated_at desc
	if err = r.db.Table("books").
		Select("books.id, books.title, reading_progresses.updated_at").
		Joins("JOIN reading_progresses ON reading_progresses.book_id = books.id").
		Order("reading_progresses.updated_at DESC").
		Limit(5).
		Scan(&recentReading).Error; err != nil {
		return
	}

	return
}

// --- Internal helpers ---

// loadTags loads tag names for a single book.
func (r *BookRepository) loadTags(bookID uint) ([]string, error) {
	var tags []string
	err := r.db.Table("tags").
		Select("tags.name").
		Joins("JOIN book_tags ON book_tags.tag_id = tags.id").
		Where("book_tags.book_id = ?", bookID).
		Pluck("name", &tags).Error
	return tags, err
}

// batchLoadTags loads tag names for multiple books, returned as a map.
func (r *BookRepository) batchLoadTags(bookIDs []uint) (map[uint][]string, error) {
	type row struct {
		BookID  uint
		TagName string
	}
	var rows []row
	err := r.db.Table("book_tags").
		Select("book_tags.book_id, tags.name as tag_name").
		Joins("JOIN tags ON book_tags.tag_id = tags.id").
		Where("book_tags.book_id IN ?", bookIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]string)
	for _, r := range rows {
		result[r.BookID] = append(result[r.BookID], r.TagName)
	}
	// Ensure all book IDs have an entry (even if empty)
	for _, id := range bookIDs {
		if _, ok := result[id]; !ok {
			result[id] = []string{}
		}
	}
	return result, nil
}

// sanitizeSortField validates and returns a safe sort field.
func sanitizeSortField(field string) string {
	allowed := map[string]bool{
		"updated_at": true,
		"title":      true,
		"author":     true,
		"created_at": true,
	}
	if allowed[field] {
		return field
	}
	return "updated_at"
}

// sanitizeSortOrder validates and returns a safe sort order.
func sanitizeSortOrder(order string) string {
	if order == "asc" {
		return "asc"
	}
	return "desc"
}

// nowUnix returns the current UTC time as a Unix timestamp in seconds.
func nowUnix() int64 {
	return model.NowUnix()
}
