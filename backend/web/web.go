package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var Dist embed.FS

// RegisterRoutes 注册前端静态文件路由，包含 SPA 回退。
func RegisterRoutes(r *gin.Engine) {
	subFS := mustSubFS(Dist, "dist")

	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}

		// 尝试精确匹配静态文件
		trimmed := strings.TrimPrefix(c.Request.URL.Path, "/")
		if _, err := subFS.Open(trimmed); err != nil {
			// SPA 回退到 index.html
			c.Request.URL.Path = "/"
		}

		http.FileServer(http.FS(subFS)).ServeHTTP(c.Writer, c.Request)
		c.Abort()
	})
}

func mustSubFS(fsys embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic("failed to get sub filesystem: " + err.Error())
	}
	return sub
}