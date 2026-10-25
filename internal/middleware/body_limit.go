package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestBodySizeLimiter returns a Gin middleware that caps incoming request body size
// to maxBytes using http.MaxBytesReader. This protects against Uncontrolled Resource
// Consumption (CWE-400) and Denial of Service (DoS) attacks via oversized payloads.
func RequestBodySizeLimiter(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request != nil && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
