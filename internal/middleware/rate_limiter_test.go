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

		// Exhaust rate limiter tokens (limit is 100)
		for i := 0; i < 100; i++ {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		// Exceeded limit request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "60", w.Header().Get("Retry-After"))
	})

	t.Run("Strict rate limiting with Retry-After header", func(t *testing.T) {
		rl := NewRateLimiter(&config.AppConfig{})
		defer rl.StopCleanup()

		router := gin.New()
		router.Use(rl.StrictRateLimit())
		router.POST("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"

		// Limit is 5
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 6th request should be rate limited and set Retry-After header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "60", w.Header().Get("Retry-After"))
	})

	t.Run("User-based rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(&config.AppConfig{})
		defer rl.StopCleanup()

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(123))
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

	t.Run("Rate limit status endpoint protection", func(t *testing.T) {
		cfg := &config.AppConfig{}
		cfg.JWT.Secret = "test-secret"
		rl := NewRateLimiter(cfg)
		defer rl.StopCleanup()

		authMiddleware := NewAuthMiddleware(cfg)

		router := gin.New()
		router.GET("/api/v1/rate-limit-status", authMiddleware.Authenticate(), func(c *gin.Context) {
			status := rl.GetRateLimitStatus(c)
			c.JSON(http.StatusOK, status)
		})

		// 1. Unauthenticated request without token should fail with 401
		unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/rate-limit-status", nil)
		unauthReq.RemoteAddr = "192.168.1.1:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, unauthReq)
		assert.Equal(t, http.StatusUnauthorized, w1.Code)

		// 2. Authenticated request with simulated auth context should succeed
		routerAuth := gin.New()
		routerAuth.GET("/api/v1/rate-limit-status", func(c *gin.Context) {
			c.Set("userID", uint(123))
			c.Next()
		}, func(c *gin.Context) {
			status := rl.GetRateLimitStatus(c)
			c.JSON(http.StatusOK, status)
		})

		authReq := httptest.NewRequest(http.MethodGet, "/api/v1/rate-limit-status", nil)
		authReq.RemoteAddr = "192.168.1.1:12345"
		w2 := httptest.NewRecorder()
		routerAuth.ServeHTTP(w2, authReq)

		assert.Equal(t, http.StatusOK, w2.Code)
		assert.True(t, w2.Body.Len() > 0)
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
				actualIP = GetClientIP(c)
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
