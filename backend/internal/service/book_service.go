package service

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/repository"
	"github.com/kitakami-hibiki/e-library/internal/storage"
)

type BookService struct {
	repo    *repository.BookRepository
	storage storage.Driver
}

func NewBookService(repo *repository.BookRepository, store storage.Driver) *BookService {
	return &BookService{repo: repo, storage: store}
}

func (s *BookService) List(page, pageSize int, keyword string) ([]model.Book, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize, keyword)
}

func (s *BookService) GetByID(id uint) (*model.Book, error) {
	return s.repo.GetByID(id)
}

func (s *BookService) Upload(filename string, reader io.Reader) (*model.Book, error) {
	ext := filepath.Ext(filename)
	if ext != ".epub" {
		return nil, fmt.Errorf("unsupported file type: %s, only EPUB allowed", ext)
	}

	path, size, err := s.storage.Save(filename, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	book := &model.Book{
		Title:      filename,
		FilePath:   path,
		FileSize:   size,
		StorageKey: "local",
	}

	if err := s.repo.Create(book); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	return book, nil
}

func (s *BookService) Update(book *model.Book) error {
	return s.repo.Update(book)
}

func (s *BookService) Delete(id uint) error {
	book, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if err := s.storage.Delete(book.FilePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return s.repo.Delete(id)
}

func (s *BookService) GetProgress(bookID uint) (*model.ReadingProgress, error) {
	return s.repo.GetProgress(bookID)
}

func (s *BookService) SaveProgress(p *model.ReadingProgress) error {
	return s.repo.UpsertProgress(p)
}

func (s *BookService) ListBookmarks(bookID uint) ([]model.Bookmark, error) {
	return s.repo.ListBookmarks(bookID)
}

func (s *BookService) CreateBookmark(b *model.Bookmark) error {
	return s.repo.CreateBookmark(b)
}

func (s *BookService) DeleteBookmark(id uint) error {
	return s.repo.DeleteBookmark(id)
}

func (s *BookService) OpenBook(id uint) (io.ReadCloser, error) {
	book, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return s.storage.Open(book.FilePath)
}

func (s *BookService) SetDriver(d storage.Driver) {
	s.storage = d
}