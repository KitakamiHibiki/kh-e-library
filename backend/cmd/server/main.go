package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kitakami-hibiki/e-library/internal/config"
	"github.com/kitakami-hibiki/e-library/internal/handler"
	"github.com/kitakami-hibiki/e-library/internal/middleware"
	"github.com/kitakami-hibiki/e-library/internal/model"
	"github.com/kitakami-hibiki/e-library/internal/repository"
	"github.com/kitakami-hibiki/e-library/internal/service"
	"github.com/kitakami-hibiki/e-library/internal/storage"
	"github.com/kitakami-hibiki/e-library/web"
)

var Version = "dev"
var draining atomic.Bool

func main() {
	cfg := config.Load()

	// Initialize file-based logging
	initLogging()
	log.Printf("kitakami_hibiki e-library v%s starting ...", Version)

	// 1. Initialize database
	db := repository.InitDB(config.DatabasePath())

	// 2. Initialize repositories
	settingRepo := repository.NewSettingRepository(db)
	bookRepo := repository.NewBookRepository(db)
	tagRepo := repository.NewTagRepository(db)

	// 3. Initialize storage factory with local driver
	factory := storage.NewFactory()
	booksDir := settingRepo.GetWithDefault(model.SettingStorageBooksDir)
	localDriver, err := storage.NewLocalDriver(resolveExePath(booksDir))
	if err != nil {
		log.Fatalf("failed to initialize local storage: %v", err)
	}
	factory.Register("local", localDriver)

	// 4. Initialize service
	bookSvc := service.NewBookService(bookRepo, tagRepo, settingRepo, factory)

	// 5. Initialize handlers
	bookHandler := handler.NewBookHandler(bookSvc)
	readingHandler := handler.NewReadingHandler(bookSvc)
	tagHandler := handler.NewTagHandler(tagRepo)
	bookTagHandler := handler.NewBookTagHandler(bookRepo)
	settingHandler := handler.NewSettingHandler(settingRepo, func(changed map[string]string) error {
		return bookSvc.OnSettingsChanged(changed)
	})
	healthHandler := handler.NewHealthHandler(db)
	statsHandler := handler.NewStatsHandler(bookSvc)

	// 6. Setup Gin (use gin.New() to avoid duplicate logger from gin.Default())
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Return 503 during graceful drain phase
		if draining.Load() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code": http.StatusServiceUnavailable,
				"data": nil,
				"msg":  "服务器正在关闭",
			})
			return
		}
		c.Next()
	})
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.MaxBodySize())

	// 7. Health check (no /api/v1 prefix)
	r.GET("/health", healthHandler.Check)

	// 8. SPA static + fallback
	web.RegisterRoutes(r)

	// 9. API routes (all GET/POST, unique paths)
	api := r.Group("/api/v1")
	{
		// Books
		api.GET("/books/list", bookHandler.List)
		api.GET("/books/detail", bookHandler.GetByID)
		api.POST("/books/create", bookHandler.Upload)
		api.POST("/books/update", bookHandler.Update)
		api.POST("/books/reprocess", bookHandler.Reprocess)
		api.POST("/books/delete", bookHandler.Delete)
		api.POST("/books/batch_delete", bookHandler.BatchDelete)
		api.GET("/books/read", bookHandler.Read)
		api.HEAD("/books/read", bookHandler.Read)
		api.GET("/books/download", bookHandler.Download)
		api.GET("/books/cover", bookHandler.Cover)
		api.HEAD("/books/cover", bookHandler.Cover)

		// Progress
		api.GET("/books/progress", readingHandler.GetProgress)
		api.POST("/books/progress/save", readingHandler.SaveProgress)

		// Tags
		api.GET("/tags/list", tagHandler.List)
		api.POST("/tags/create", tagHandler.Create)
		api.POST("/tags/update", tagHandler.Update)
		api.POST("/tags/delete", tagHandler.Delete)

		// Book-Tag associations
		api.POST("/books/tags/add", bookTagHandler.Add)
		api.POST("/books/tags/remove", bookTagHandler.Remove)

		// Settings
		api.GET("/settings/list", settingHandler.List)
		api.POST("/settings/update", settingHandler.Update)

		// Stats
		api.GET("/stats/overview", statsHandler.Overview)
	}

	// 10. Start server with graceful shutdown
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	// Phase 1: Enter draining mode — reject new requests with 503
	draining.Store(true)
	log.Println("draining: rejecting new requests (503)")

	// Phase 2: Wait for async processing goroutines (5-minute timeout)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer shutdownCancel()
	bookSvc.Shutdown(shutdownCtx)

	// Phase 3: Shutdown HTTP server — wait for in-flight requests (10-second timeout)
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer httpCancel()
	if err := srv.Shutdown(httpCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	// Close database connection
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	log.Println("server exited")
}

// initLogging sets up multi-output logging: both console and daily log files.
// Log files are stored in the .log/ directory next to the working directory.
func initLogging() {
	logDir := ".log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("warning: failed to create log directory %s: %v", logDir, err)
		return
	}

	// Daily log file name
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("warning: failed to open log file %s: %v", logFile, err)
		return
	}

	// Write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// resolveExePath resolves a relative path based on working directory,
// falling back to the executable directory.
// No existence check — the data directory may not exist yet on first run.
func resolveExePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, path)
	}
	exe, err := os.Executable()
	if err != nil {
		return path
	}
	return filepath.Join(filepath.Dir(exe), path)
}
