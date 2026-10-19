package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets baseline HTTP security headers for API responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-XSS-Protection", "0")
		c.Header("Cache-Control", "no-store")
		// API is JSON-only; CSP is mainly for browser HTML but harmless
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}
