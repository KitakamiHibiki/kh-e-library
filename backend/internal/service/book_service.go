package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kitakami-hibiki/e-library/internal/epub"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/repository"
	"github.com/kitakami-hibiki/e-library/internal/storage"
)

// BookService implements the business logic for book operations.
type BookService struct {
	bookRepo    *repository.BookRepository
	tagRepo     *repository.TagRepository
	settingRepo *repository.SettingRepository
	factory     *storage.Factory
	wg          sync.WaitGroup
}

// NewBookService creates a new BookService.
func NewBookService(
	bookRepo *repository.BookRepository,
	tagRepo *repository.TagRepository,
	settingRepo *repository.SettingRepository,
	factory *storage.Factory,
) *BookService {
	return &BookService{
		bookRepo:    bookRepo,
		tagRepo:     tagRepo,
		settingRepo: settingRepo,
		factory:     factory,
	}
}

// --- Book CRUD ---

// List returns a paginated, filtered, sorted list of books.
func (s *BookService) List(page, pageSize int, keyword, tag, bookStatus, sortField, sortOrder string) ([]model.Book, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if sortField == "" {
		sortField = s.settingRepo.GetWithDefault(model.SettingSortField)
	}
	if sortOrder == "" {
		sortOrder = s.settingRepo.GetWithDefault(model.SettingSortOrder)
	}
	return s.bookRepo.List(page, pageSize, keyword, tag, bookStatus, sortField, sortOrder)
}

// GetByID returns a single book by ID.
func (s *BookService) GetByID(id uint) (*model.Book, error) {
	return s.bookRepo.GetByID(id)
}

// UpdateBook updates specific fields of a book.
func (s *BookService) UpdateBook(id uint, fields map[string]interface{}) error {
	return s.bookRepo.UpdateFields(id, fields)
}

// Upload validates, saves, and creates a book record, then launches async processing.
func (s *BookService) Upload(filename string, reader io.Reader) (*model.Book, error) {
	// 1. Validate file extension (case-insensitive)
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".epub" && ext != ".pdf" {
		return nil, fmt.Errorf("不支持的文件类型: %s, 仅支持 EPUB 和 PDF", ext)
	}
	fileType := strings.TrimPrefix(ext, ".")

	// Read all data for validation, hashing, and saving
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败")
	}

	// 2. Compute SHA-256 and check for duplicates
	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])
	existing, _ := s.bookRepo.GetByHash(fileHash)
	if existing != nil {
		return nil, fmt.Errorf("文件已存在")
	}

	// 3. Basic structure validation
	if fileType == "epub" {
		if err := epub.Validate(bytes.NewReader(data), int64(len(data))); err != nil {
			return nil, fmt.Errorf("校验失败: 无效的 EPUB 文件 (%v)", err)
		}
	} else if fileType == "pdf" {
		if len(data) < 5 || !bytes.HasPrefix(data, []byte("%PDF-")) {
			return nil, fmt.Errorf("校验失败: 无效的 PDF 文件")
		}
	}

	// 4. Create database record
	driver := s.factory.Get("local")
	book := &model.Book{
		Title:      strings.TrimSuffix(filename, filepath.Ext(filename)),
		FileType:    fileType,
		FileSize:    int64(len(data)),
		FileHash:    fileHash,
		StorageKey:  driver.Name(),
		ReadStatus:  "unread",
		BookStatus:  "processing",
	}
	if err := s.bookRepo.Create(book); err != nil {
		return nil, fmt.Errorf("创建记录失败: %w", err)
	}
	log.Printf("Upload: created book %d (%s, %s, %d bytes)", book.ID, book.Title, fileType, book.FileSize)

	// 5. Save file to storage
	bookFileName := "book" + ext
	savedName, size, err := driver.Save(book.ID, bookFileName, bytes.NewReader(data))
	if err != nil {
		// Rollback: delete the DB record
		s.bookRepo.Delete(book.ID)
		return nil, fmt.Errorf("保存文件失败")
	}

	// 6. Update file info in DB
	s.bookRepo.UpdateFields(book.ID, map[string]interface{}{
		"file":      savedName,
		"file_size": size,
	})
	book.BookFile = savedName
	book.FileSize = size

	// 7. Launch async processing
	s.ProcessBookAsync(book.ID)

	return book, nil
}

// Reprocess re-processes a failed book.
func (s *BookService) Reprocess(id uint) error {
	book, err := s.bookRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("书籍不存在")
	}
	if book.BookStatus != "failed" {
		return fmt.Errorf("仅限重新处理失败的书籍")
	}

	// Delete existing cover file if present
	if book.CoverFile != "" {
		driver := s.factory.Get(book.StorageKey)
		if driver != nil {
			if err := driver.Delete(book.ID, book.CoverFile); err != nil {
				log.Printf("warning: failed to delete old cover for book %d: %v", id, err)
			}
		}
	}

	// Clear cover and set status to processing
	s.bookRepo.UpdateFields(id, map[string]interface{}{
		"cover_file":   "",
		"book_status": "processing",
	})

	s.ProcessBookAsync(id)
	return nil
}

// Delete removes a book, its files, and all associated data.
func (s *BookService) Delete(id uint) error {
	book, err := s.bookRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("书籍不存在")
	}

	// 1. If processing, mark as deleted atomically
	if book.BookStatus == "processing" {
		affected, _ := s.bookRepo.UpdateBookStatus(id, "processing", "deleted")
		if affected == 0 {
			// Status changed (e.g. async goroutine just completed), re-fetch
			book, _ = s.bookRepo.GetByID(id)
			if book == nil {
				return nil
			}
		}
	}

	// 2. Delete book directory via driver
	driver := s.factory.Get(book.StorageKey)
	if driver != nil {
		driver.DeleteBook(book.ID)
	}

	// 3. Delete associated data
	s.bookRepo.DeleteProgress(id)
	s.bookRepo.DeleteBookTags(id)

	// 4. Delete DB record
	s.bookRepo.Delete(id)

	return nil
}

// BatchDelete deletes multiple books by ID.
func (s *BookService) BatchDelete(ids []uint) (int, error) {
	deleted := 0
	for _, id := range ids {
		if err := s.Delete(id); err != nil {
			continue
		}
		deleted++
	}
	return deleted, nil
}

// OpenBook opens a book file for reading. Returns the reader, the book object, and any error.
func (s *BookService) OpenBook(id uint) (storage.ReadSeekCloser, *model.Book, error) {
	book, err := s.bookRepo.GetByID(id)
	if err != nil {
		return nil, nil, fmt.Errorf("书籍不存在")
	}
	if book.BookStatus != "ready" {
		return nil, nil, fmt.Errorf("书籍未处理完成，不可访问")
	}
	driver := s.factory.Get(book.StorageKey)
	if driver == nil {
		return nil, nil, fmt.Errorf("存储后端不可用")
	}
	reader, err := driver.Open(book.ID, book.BookFile)
	if err != nil {
		return nil, nil, err
	}
	return reader, book, nil
}

// OpenCover opens a cover image file. Returns the reader, the book object, and any error.
func (s *BookService) OpenCover(id uint) (storage.ReadSeekCloser, *model.Book, error) {
	book, err := s.bookRepo.GetByID(id)
	if err != nil {
		return nil, nil, fmt.Errorf("书籍不存在")
	}
	if book.CoverFile == "" {
		return nil, nil, fmt.Errorf("封面不存在")
	}
	driver := s.factory.Get(book.StorageKey)
	if driver == nil {
		return nil, nil, fmt.Errorf("存储后端不可用")
	}
	reader, err := driver.Open(book.ID, book.CoverFile)
	if err != nil {
		return nil, nil, err
	}
	return reader, book, nil
}

// --- Progress ---

// GetProgress returns the reading progress for a book.
func (s *BookService) GetProgress(bookID uint) (*model.ReadingProgress, error) {
	return s.bookRepo.GetProgress(bookID)
}

// SaveProgress saves reading progress (upsert).
func (s *BookService) SaveProgress(p *model.ReadingProgress) error {
	return s.bookRepo.UpsertProgress(p)
}

// GetStatsOverview returns reading statistics.
func (s *BookService) GetStatsOverview() (*model.StatsOverview, error) {
	totalBooks, unread, reading, finished, recentReading, err := s.bookRepo.GetStatsOverview()
	if err != nil {
		return nil, err
	}
	return &model.StatsOverview{
		TotalBooks:    totalBooks,
		UnreadCount:   unread,
		ReadingCount:  reading,
		FinishedCount: finished,
		RecentReading: recentReading,
	}, nil
}

// OnSettingsChanged handles settings changes (e.g. storage driver hot-swap).
func (s *BookService) OnSettingsChanged(changed map[string]string) error {
	// For now, only storage.local.books_dir changes trigger driver re-initialization
	if _, ok := changed["storage.local.books_dir"]; ok {
		newDir := s.settingRepo.GetWithDefault(model.SettingStorageBooksDir)
		driver, err := storage.NewLocalDriver(newDir)
		if err != nil {
			return fmt.Errorf("failed to initialize local driver: %w", err)
		}
		s.factory.Register("local", driver)
		log.Printf("OnSettingsChanged: re-initialized local driver with books_dir=%s", newDir)
	}
	return nil
}

// Shutdown waits for async processing goroutines to complete or context cancellation.
func (s *BookService) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// --- Async Processing ---

// ProcessBookAsync launches an async goroutine to extract metadata and cover.
func (s *BookService) ProcessBookAsync(bookID uint) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Panic recovery
		defer func() {
			if r := recover(); r != nil {
				log.Printf("async processing panic for book %d: %v", bookID, r)
				s.bookRepo.UpdateBookStatus(bookID, "processing", "failed")
			}
		}()

		if ctx.Err() != nil {
			return
		}

		book, err := s.bookRepo.GetByID(bookID)
		if err != nil {
			log.Printf("async processing: book %d not found: %v", bookID, err)
			return
		}

		driver := s.factory.Get(book.StorageKey)
		if driver == nil {
			log.Printf("async processing: no driver for book %d (storageKey=%s)", bookID, book.StorageKey)
			s.setFailed(bookID)
			return
		}

		var updateFields map[string]interface{}

		if book.FileType == "epub" {
			updateFields = s.processEPUB(ctx, book, driver)
		} else if book.FileType == "pdf" {
			updateFields = s.processPDF(ctx, book, driver)
		}

		log.Printf("async processing: book %d updateFields=%v", bookID, updateFields)

		if updateFields == nil {
			updateFields = make(map[string]interface{})
		}

		// Apply metadata updates first
		if len(updateFields) > 0 {
			if err := s.bookRepo.UpdateFields(bookID, updateFields); err != nil {
				log.Printf("failed to update metadata for book %d: %v", bookID, err)
				s.setFailed(bookID)
				return
			}
			log.Printf("updated metadata for book %d successfully", bookID)
		}

		// Atomically set status to ready
		affected, err := s.bookRepo.UpdateBookStatus(bookID, "processing", "ready")
		if err != nil {
			log.Printf("failed to update book %d status: %v", bookID, err)
			return
		}
		if affected == 0 {
			log.Printf("book %d status no longer processing, skipping update", bookID)
			return
		}

		log.Printf("async processing completed for book %d", bookID)
	}()
}

func (s *BookService) setFailed(bookID uint) {
	s.bookRepo.UpdateBookStatus(bookID, "processing", "failed")
}

// --- EPUB Processing ---

func (s *BookService) processEPUB(ctx context.Context, book *model.Book, driver storage.Driver) map[string]interface{} {
	reader, err := driver.Open(book.ID, book.BookFile)
	if err != nil {
		log.Printf("processEPUB: failed to open book %d file %s: %v", book.ID, book.BookFile, err)
		s.setFailed(book.ID)
		return nil
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("processEPUB: failed to read book %d: %v", book.ID, err)
		s.setFailed(book.ID)
		return nil
	}
	log.Printf("processEPUB: read %d bytes for book %d", len(data), book.ID)

	meta, err := epub.Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		log.Printf("processEPUB: failed to parse book %d: %v", book.ID, err)
		s.setFailed(book.ID)
		return nil
	}

	fields := map[string]interface{}{}
	if meta.Title != "" {
		fields["title"] = meta.Title
	}
	if len(meta.Authors) > 0 {
		fields["author"] = strings.Join(meta.Authors, ", ")
	}
	if meta.Publisher != "" {
		fields["publisher"] = meta.Publisher
	}
	if meta.ISBN != "" {
		fields["isbn"] = meta.ISBN
	}
	if meta.Language != "" {
		fields["language"] = meta.Language
	}
	if meta.Description != "" {
		fields["description"] = meta.Description
	}
	if meta.Pages > 0 {
		fields["pages"] = meta.Pages
	}

	// Extract cover
	coverData, coverExt, coverErr := epub.ReadCover(bytes.NewReader(data), int64(len(data)))
	if coverErr == nil && len(coverData) > 0 {
		log.Printf("processEPUB: extracted cover for book %d, raw size=%d, ext=%s", book.ID, len(coverData), coverExt)
		// Compress if >1MB
		if len(coverData) > 1<<20 {
			compressed, err := compressCover(coverData)
			if err == nil {
				coverData = compressed
				log.Printf("processEPUB: compressed cover for book %d to %d bytes", book.ID, len(coverData))
			}
		}
		coverFileName := "cover" + coverExt
		savedName, _, err := driver.Save(book.ID, coverFileName, bytes.NewReader(coverData))
		if err == nil {
			fields["cover_file"] = savedName
			log.Printf("processEPUB: saved cover %s for book %d", savedName, book.ID)
		} else {
			log.Printf("processEPUB: failed to save cover for book %d: %v", book.ID, err)
		}
	} else {
		log.Printf("processEPUB: no cover for book %d: coverErr=%v, dataLen=%d", book.ID, coverErr, len(coverData))
	}

	// Auto-create tags from dc:subject
	if len(meta.Subjects) > 0 {
		s.autoCreateTags(book.ID, meta.Subjects)
	}

	return fields
}

// --- PDF Processing ---

func (s *BookService) processPDF(ctx context.Context, book *model.Book, driver storage.Driver) map[string]interface{} {
	reader, err := driver.Open(book.ID, book.BookFile)
	if err != nil {
		log.Printf("processPDF: failed to open book %d file %s: %v", book.ID, book.BookFile, err)
		s.setFailed(book.ID)
		return nil
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("processPDF: failed to read book %d: %v", book.ID, err)
		s.setFailed(book.ID)
		return nil
	}
	log.Printf("processPDF: read %d bytes for book %d", len(data), book.ID)

	fields := map[string]interface{}{}

	// Extract PDF metadata
	pdfMeta, err := extractPDFMetadata(data)
	if err == nil {
		log.Printf("processPDF: metadata for book %d: title=%q author=%q pages=%d", book.ID, pdfMeta.Title, pdfMeta.Author, pdfMeta.Pages)
		if pdfMeta.Title != "" {
			fields["title"] = pdfMeta.Title
		}
		if pdfMeta.Author != "" {
			fields["author"] = pdfMeta.Author
		}
		if pdfMeta.Description != "" {
			fields["description"] = pdfMeta.Description
		}
		if pdfMeta.Pages > 0 {
			fields["pages"] = pdfMeta.Pages
		}
	} else {
		log.Printf("processPDF: failed to extract metadata for book %d: %v", book.ID, err)
	}

	// Extract cover image from first page
	coverData, coverErr := extractPDFCover(data)
	if coverErr == nil && len(coverData) > 0 {
		log.Printf("processPDF: extracted cover for book %d, raw size=%d", book.ID, len(coverData))
		// Re-encode as JPEG
		jpegData, jpegErr := reencodeAsJPEG(coverData)
		if jpegErr == nil {
			coverData = jpegData
		} else {
			log.Printf("processPDF: failed to re-encode cover as JPEG for book %d: %v", book.ID, jpegErr)
		}

		// Compress if >1MB
		if len(coverData) > 1<<20 {
			compressed, err := compressCover(coverData)
			if err == nil {
				coverData = compressed
			}
		}

		coverFileName := "cover.jpg"
		savedName, _, err := driver.Save(book.ID, coverFileName, bytes.NewReader(coverData))
		if err == nil {
			fields["cover_file"] = savedName
			log.Printf("processPDF: saved cover %s for book %d", savedName, book.ID)
		} else {
			log.Printf("processPDF: failed to save cover for book %d: %v", book.ID, err)
		}
	} else {
		log.Printf("processPDF: no cover for book %d: coverErr=%v, dataLen=%d", book.ID, coverErr, len(coverData))
	}

	return fields
}

// --- Helpers ---

func (s *BookService) autoCreateTags(bookID uint, subjects []string) {
	for _, name := range subjects {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tag, err := s.tagRepo.GetByName(name)
		if err != nil {
			tag, err = s.tagRepo.Create(name)
			if err != nil {
				tag, _ = s.tagRepo.GetByName(name)
				if tag == nil {
					continue
				}
			}
		}
		if tag != nil {
			s.bookRepo.AddTag(bookID, tag.ID)
		}
	}
}

func compressCover(data []byte) ([]byte, error) {
	img, _, err := decodeImage(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	// Resize if needed to fit under 1MB
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	for len(data) > 1<<20 && w > 100 && h > 100 {
		w = w * 3 / 4
		h = h * 3 / 4
		img = resizeImage(img, w, h)
		var buf bytes.Buffer
		if err := encodeJPEG(&buf, img, 85); err != nil {
			return nil, err
		}
		data = buf.Bytes()
	}
	return data, nil
}
