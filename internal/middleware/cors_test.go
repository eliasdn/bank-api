package middleware

import (
	"bank-api/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_ProductionRejectsWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.LoadTestConfig()
	cfg.Environment = "production"
	cfg.CORS.AllowedOrigins = []string{"*"}

	router := gin.New()
	router.Use(CORSMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Preflight OPTIONS from arbitrary origin should be rejected in production
	req, err := http.NewRequest(http.MethodOptions, "/test", nil)
	assert.NoError(t, err)
	req.Header.Set("Origin", "https://malicious.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_DevelopmentAllowsWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.LoadTestConfig()
	cfg.Environment = "development"
	cfg.CORS.AllowedOrigins = []string{"*"}

	router := gin.New()
	router.Use(CORSMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	assert.NoError(t, err)
	req.Header.Set("Origin", "https://example.com")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
