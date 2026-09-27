package middleware

import (
	"bank-api/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a Gin middleware that handles CORS based on the app configuration
func CORSMiddleware(cfg *config.AppConfig) gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()

	// Configure Allowed Origins
	if len(cfg.CORS.AllowedOrigins) == 1 && cfg.CORS.AllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else if len(cfg.CORS.AllowedOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.CORS.AllowedOrigins
	} else {
		// Secure default if misconfigured
		corsConfig.AllowOrigins = []string{"http://localhost"}
	}

	if len(cfg.CORS.AllowedMethods) > 0 {
		corsConfig.AllowMethods = cfg.CORS.AllowedMethods
	}

	if len(cfg.CORS.AllowedHeaders) > 0 {
		corsConfig.AllowHeaders = cfg.CORS.AllowedHeaders
	}

	if len(cfg.CORS.ExposedHeaders) > 0 {
		corsConfig.ExposeHeaders = cfg.CORS.ExposedHeaders
	}

	corsConfig.AllowCredentials = cfg.CORS.AllowCredentials

	// Ensure AllowCredentials is not true if AllowAllOrigins is true (violates W3C CORS spec and causes panic)
	if corsConfig.AllowAllOrigins {
		corsConfig.AllowCredentials = false
	}

	corsConfig.MaxAge = cfg.CORS.MaxAge

	return cors.New(corsConfig)
}
