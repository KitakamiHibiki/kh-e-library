package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxBodySize limits JSON request body size to 1MB per spec §5.1.
// Multipart/form-data uploads (e.g. /books/create) are exempt —
// the 200MB limit is enforced by the handler itself.
// Uses http.MaxBytesReader to enforce the limit on the actual body read,
// not just the Content-Length header (which can be omitted or spoofed).
func MaxBodySize() gin.HandlerFunc {
	const maxBytes = 1 << 20 // 1MB
	return func(c *gin.Context) {
		// Skip for multipart/form-data (file uploads)
		ct := c.Request.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/form-data") {
			c.Next()
			return
		}
		// Only apply to requests with a body (POST)
		if c.Request.Method == "POST" && c.Request.Body != nil {
			// Quick check: if Content-Length is known and exceeds limit, reject early
			if c.Request.ContentLength > maxBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
					"code": http.StatusRequestEntityTooLarge,
					"data": nil,
					"msg":  "文件超过大小限制",
				})
				return
			}
			// Wrap the body with MaxBytesReader to enforce the limit on actual reads.
			// This catches clients that omit or spoof the Content-Length header.
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
