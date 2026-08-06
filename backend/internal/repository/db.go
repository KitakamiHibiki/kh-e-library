package repository

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the SQLite database, runs AutoMigrate, and returns a *gorm.DB instance.
func InitDB(dsn string) *gorm.DB {
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dsn+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.Book{},
		&model.Tag{},
		&model.BookTag{},
		&model.ReadingProgress{},
		&model.Setting{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Startup repair: reset any books stuck in "processing" state to "failed"
	// (their async goroutines were lost when the previous process terminated)
	result := db.Model(&model.Book{}).Where("book_status = ?", "processing").Update("book_status", "failed")
	if result.RowsAffected > 0 {
		log.Printf("startup repair: reset %d books from processing to failed", result.RowsAffected)
	}

	log.Println("database initialized successfully")
	return db
}
