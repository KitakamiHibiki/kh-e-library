package main

import (
	"fmt"
	"log"

	"github.com/kitakami-hibiki/e-library/internal/config"
	"github.com/kitakami-hibiki/e-library/internal/handler"
	"github.com/kitakami-hibiki/e-library/internal/middleware"
	"github.com/kitakami-hibiki/e-library/internal/repository"
	"github.com/kitakami-hibiki/e-library/internal/service"
	"github.com/kitakami-hibiki/e-library/internal/storage"
	"github.com/kitakami-hibiki/e-library/web"
	"github.com/gin-gonic/gin"
)

var Version = "dev"

func main() {
	cfg := config.Load()
	log.Printf("kitakami_hibiki e-library v%s starting ...", Version)

	repository.InitDB(cfg.Database.Path)

	storeFactory := storage.NewFactory()
	localDriver := storage.NewLocalDriver(cfg.Storage.Local.BooksDir)
	storeFactory.Register("local", localDriver)
	if cfg.Storage.Baidu.ClientID != "" {
		baiduDriver := storage.NewBaiduDriver(
			cfg.Storage.Baidu.ClientID,
			cfg.Storage.Baidu.ClientSecret,
			cfg.Storage.Baidu.RefreshToken,
		)
		storeFactory.Register("baidu", baiduDriver)
	}
	storeDriver := storeFactory.Get(cfg.Storage.Driver)
	if storeDriver == nil {
		log.Fatalf("unknown storage driver: %s", cfg.Storage.Driver)
	}

	bookRepo := repository.NewBookRepository()
	bookSvc := service.NewBookService(bookRepo, storeDriver)
	bookHandler := handler.NewBookHandler(bookSvc)
	readingHandler := handler.NewReadingHandler(bookSvc)

	settingRepo := repository.NewSettingRepository()
	settingHandler := handler.NewSettingHandler(settingRepo)

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