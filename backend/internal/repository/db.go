package repository

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kitakami-hibiki/e-library/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(dsn string) {
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := DB.AutoMigrate(
		&model.Book{},
		&model.ReadingProgress{},
		&model.Bookmark{},
		&model.Setting{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	log.Println("database initialized successfully")
}