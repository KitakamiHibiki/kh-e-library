package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/config"
	"github.com/kitakami-hibiki/e-library/internal/handler"
	"github.com/kitakami-hibiki/e-library/internal/middleware"
	"github.com/kitakami-hibiki/e-library/internal/repository"
	"github.com/kitakami-hibiki/e-library/internal/service"
	"github.com/kitakami-hibiki/e-library/internal/storage"
	"github.com/kitakami-hibiki/e-library/web"
)

var Version = "dev"

func main() {
	cfg := config.Load()
	log.Printf("kitakami_hibiki e-library v%s starting ...", Version)

	repository.InitDB(config.DatabasePath())

	settingRepo := repository.NewSettingRepository()
	settings, _ := settingRepo.GetMap()
	storeDriver := initStorageFromSettings(settings)

	bookRepo := repository.NewBookRepository()
	bookSvc := service.NewBookService(bookRepo, storeDriver)

	bookHandler := handler.NewBookHandler(bookSvc)
	readingHandler := handler.NewReadingHandler(bookSvc)
	settingHandler := handler.NewSettingHandler(settingRepo, func(all map[string]string) {
		reloadStorage(bookSvc, all)
	})

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	web.RegisterRoutes(r)

	api := r.Group("/api/v1")
	{
		api.GET("/books", bookHandler.List)
		api.GET("/books/:id", bookHandler.GetByID)
		api.POST("/books", bookHandler.Upload)
		api.PUT("/books/:id", bookHandler.Update)
		api.DELETE("/books/:id", bookHandler.Delete)
		api.GET("/books/:id/read", bookHandler.Read)

		api.GET("/books/:id/progress", readingHandler.GetProgress)
		api.PUT("/books/:id/progress", readingHandler.SaveProgress)

		api.GET("/books/:id/bookmarks", readingHandler.ListBookmarks)
		api.POST("/books/:id/bookmarks", readingHandler.CreateBookmark)
		api.DELETE("/bookmarks/:id", readingHandler.DeleteBookmark)

		api.GET("/settings", settingHandler.ListAll)
		api.PUT("/settings", settingHandler.Update)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func initStorageFromSettings(s map[string]string) storage.Driver {
	driverType := s["storage.driver"]
	if driverType == "" || driverType == "local" {
		booksDir := s["storage.local.books_dir"]
		if booksDir == "" {
			booksDir = config.DefaultBooksDir()
		}
		return storage.NewLocalDriver(resolveExePath(booksDir))
	}
	return storage.NewBaiduDriver(
		s["storage.baidu.client_id"],
		s["storage.baidu.client_secret"],
		s["storage.baidu.refresh_token"],
	)
}

func reloadStorage(svc *service.BookService, s map[string]string) {
	svc.SetDriver(initStorageFromSettings(s))
	log.Printf("storage driver switched to %s", s["storage.driver"])
}

func resolveExePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	exe, err := os.Executable()
	if err != nil {
		return path
	}
	return filepath.Join(filepath.Dir(exe), path)
}