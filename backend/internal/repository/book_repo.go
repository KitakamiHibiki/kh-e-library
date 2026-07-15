package repository

import (
	"github.com/kitakami-hibiki/e-library/internal/model"
)

type BookRepository struct{}

func NewBookRepository() *BookRepository {
	return &BookRepository{}
}

func (r *BookRepository) List(page, pageSize int, keyword string) ([]model.Book, int64, error) {
	var books []model.Book
	var total int64

	query := DB.Model(&model.Book{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR author LIKE ?", like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("updated_at DESC").Find(&books).Error; err != nil {
		return nil, 0, err
	}

	return books, total, nil
}

func (r *BookRepository) GetByID(id uint) (*model.Book, error) {
	var book model.Book
	if err := DB.First(&book, id).Error; err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *BookRepository) Create(book *model.Book) error {
	return DB.Create(book).Error
}

func (r *BookRepository) Update(book *model.Book) error {
	return DB.Save(book).Error
}

func (r *BookRepository) Delete(id uint) error {
	return DB.Delete(&model.Book{}, id).Error
}

func (r *BookRepository) GetProgress(bookID uint) (*model.ReadingProgress, error) {
	var p model.ReadingProgress
	if err := DB.Where("book_id = ?", bookID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *BookRepository) UpsertProgress(p *model.ReadingProgress) error {
	return DB.Where("book_id = ?", p.BookID).Assign(p).FirstOrCreate(p).Error
}

func (r *BookRepository) ListBookmarks(bookID uint) ([]model.Bookmark, error) {
	var marks []model.Bookmark
	if err := DB.Where("book_id = ?", bookID).
		Order("progress ASC").Find(&marks).Error; err != nil {
		return nil, err
	}
	return marks, nil
}

func (r *BookRepository) CreateBookmark(b *model.Bookmark) error {
	return DB.Create(b).Error
}

func (r *BookRepository) DeleteBookmark(id uint) error {
	return DB.Delete(&model.Bookmark{}, id).Error
}
