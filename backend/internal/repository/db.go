 package repository

 import (
 	"log"

 	"github.com/kitakami-hibiki/e-library/internal/model"
 	"gorm.io/driver/sqlite"
 	"gorm.io/gorm"
 	"gorm.io/gorm/logger"
 )

 var DB *gorm.DB

 func InitDB(dsn string) {
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
 	); err != nil {
 		log.Fatalf("failed to migrate database: %v", err)
 	}

 	log.Println("database initialized successfully")
 }
