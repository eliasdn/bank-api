package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware returns a Gin middleware that attaches standard HTTP security headers
// to guard against common web vulnerabilities like clickjacking, MIME-sniffing, XSS, and information leakage.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}
