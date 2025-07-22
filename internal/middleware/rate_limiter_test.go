package middleware

import (
	"bank-api/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("IP-based rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(&config.AppConfig{})
		defer rl.StopCleanup()

		router := gin.New()
		router.Use(rl.RateLimit())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Create a request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		// Test within limit
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("User-based rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(&config.AppConfig{})
		defer rl.StopCleanup()

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(123))
			c.Next()
		})
		router.Use(rl.RateLimit())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		// Test within limit
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Get client IP from headers", func(t *testing.T) {
		rl := NewRateLimiter(&config.AppConfig{})
		defer rl.StopCleanup()

		router := gin.New()
		router.Use(rl.RateLimit())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Test X-Forwarded-For header
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 203.0.113.2")
		req.RemoteAddr = "192.168.1.1:12345"

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Rate limit status endpoint", func(t *testing.T) {
		rl := NewRateLimiter(&config.AppConfig{})
		defer rl.StopCleanup()

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", uint(123))
			c.Next()
		})
		router.GET("/status", func(c *gin.Context) {
			status := rl.GetRateLimitStatus(c)
			c.JSON(http.StatusOK, status)
		})

		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, w.Body.Len() > 0)
	})
}

func TestGetClientIP(t *testing.T) {
	rl := NewRateLimiter(&config.AppConfig{})
	defer rl.StopCleanup()

	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For single IP",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.1"},
			remote:   "192.168.1.1:12345",
			expected: "203.0.113.1",
		},
		{
			name:     "X-Forwarded-For multiple IPs",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.1, 203.0.113.2"},
			remote:   "192.168.1.1:12345",
			expected: "203.0.113.1",
		},
		{
			name:     "X-Real-IP header",
			headers:  map[string]string{"X-Real-IP": "203.0.113.3"},
			remote:   "192.168.1.1:12345",
			expected: "203.0.113.3",
		},
		{
			name:     "RemoteAddr fallback",
			headers:  map[string]string{},
			remote:   "192.168.1.1:12345",
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			var actualIP string

			router.GET("/test", func(c *gin.Context) {
				actualIP = rl.getClientIP(c)
				c.JSON(http.StatusOK, gin.H{"ip": actualIP})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remote

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, actualIP)
		})
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter(&config.AppConfig{})
	defer rl.StopCleanup()

	router := gin.New()
	router.Use(rl.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}
