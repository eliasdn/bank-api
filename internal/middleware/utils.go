package middleware

import (
	"github.com/gin-gonic/gin"
)

// GetClientIP extracts the real client IP from the request securely using Gin's ClientIP method,
// which properly respects configured trusted proxies and prevents IP spoofing via X-Forwarded-For headers.
func GetClientIP(c *gin.Context) string {
	return c.ClientIP()
}
