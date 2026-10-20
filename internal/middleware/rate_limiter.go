package middleware

import (
	"bank-api/internal/config"
	"bank-api/internal/validation"

	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	rateLimitIPExceededJSON     = []byte(`{"error":{"code":"rate_limit_exceeded","message":"Too many requests from this IP address","retry_after":"60"}}`)
	rateLimitUserExceededJSON   = []byte(`{"error":{"code":"rate_limit_exceeded","message":"Too many requests for this user account","retry_after":"3600"}}`)
	strictRateLimitExceededJSON = []byte(`{"error":{"code":"rate_limit_exceeded","message":"Too many requests to this endpoint from this IP address","retry_after":"60"}}`)
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	config *config.AppConfig
	// IP-based limiters
	ipLimiters sync.Map
	// Strict IP-based limiters (e.g. auth endpoints)
	authIpLimiters sync.Map
	// User-based limiters
	userLimiters sync.Map
	// Cleanup ticker
	cleanupTicker *time.Ticker
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *config.AppConfig) *RateLimiter {
	rl := &RateLimiter{
		config: config,
	}

	// Start cleanup routine only if not in test mode
	if gin.Mode() != gin.TestMode {
		rl.cleanupTicker = time.NewTicker(1 * time.Hour)
		go rl.cleanupExpiredLimiters()
	}

	return rl
}

// LimitByIP creates or gets a rate limiter for the given IP
func (rl *RateLimiter) LimitByIP(ip string) *rate.Limiter {
	if val, ok := rl.ipLimiters.Load(ip); ok {
		return val.(*rate.Limiter)
	}

	// 100 requests per minute per IP
	limiter := rate.NewLimiter(rate.Every(time.Minute), 100)
	actual, _ := rl.ipLimiters.LoadOrStore(ip, limiter)
	return actual.(*rate.Limiter)
}

// StrictLimitByIP creates or gets a strict rate limiter for the given IP
func (rl *RateLimiter) StrictLimitByIP(ip string) *rate.Limiter {
	if val, ok := rl.authIpLimiters.Load(ip); ok {
		return val.(*rate.Limiter)
	}

	// 5 requests per minute per IP for sensitive endpoints
	limiter := rate.NewLimiter(rate.Every(time.Minute), 5)
	actual, _ := rl.authIpLimiters.LoadOrStore(ip, limiter)
	return actual.(*rate.Limiter)
}

// normalizeUserID normalizes different user ID types (uint, float64, string, etc.) to a canonical string key
// to prevent rate limit bypasses caused by context key type mismatches across middlewares.
func normalizeUserID(userID interface{}) string {
	switch v := userID.(type) {
	case uint:
		return validation.FormatUint(uint64(v))
	case uint64:
		return validation.FormatUint(v)
	case int:
		return validation.FormatInt(int64(v))
	case int64:
		return validation.FormatInt(v)
	case float64:
		return validation.FormatUint(uint64(v))
	case string:
		return v
	default:
		return "unknown"
	}
}

// LimitByUser creates or gets a rate limiter for the given user ID
func (rl *RateLimiter) LimitByUser(userID interface{}) *rate.Limiter {
	key := normalizeUserID(userID)
	if val, ok := rl.userLimiters.Load(key); ok {
		return val.(*rate.Limiter)
	}

	// 1000 requests per hour per authenticated user
	limiter := rate.NewLimiter(rate.Every(time.Hour), 1000)
	actual, _ := rl.userLimiters.LoadOrStore(key, limiter)
	return actual.(*rate.Limiter)
}

// cleanupExpiredLimiters removes expired rate limiters
func (rl *RateLimiter) cleanupExpiredLimiters() {
	for range rl.cleanupTicker.C {
		// Identify and remove IP limiters
		rl.ipLimiters.Range(func(key, value interface{}) bool {
			limiter := value.(*rate.Limiter)
			if limiter.Tokens() == 100 { // Full bucket indicates no recent usage
				rl.ipLimiters.Delete(key)
			}
			return true
		})

		// Identify and remove strict IP limiters
		rl.authIpLimiters.Range(func(key, value interface{}) bool {
			limiter := value.(*rate.Limiter)
			if limiter.Tokens() == 5 {
				rl.authIpLimiters.Delete(key)
			}
			return true
		})

		// Identify and remove user limiters
		rl.userLimiters.Range(func(key, value interface{}) bool {
			limiter := value.(*rate.Limiter)
			if limiter.Tokens() == 1000 { // Full bucket indicates no recent usage
				rl.userLimiters.Delete(key)
			}
			return true
		})
	}
}

// RateLimit returns a gin middleware for rate limiting
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := GetClientIP(c)

		// Check IP-based rate limiting
		ipLimiter := rl.LimitByIP(clientIP)
		if !ipLimiter.Allow() {
			c.Header("Retry-After", "60")
			// Optimization: Use pre-allocated JSON response to avoid reflection and map allocations during rate limit rejection hot-path
			c.Data(http.StatusTooManyRequests, "application/json", rateLimitIPExceededJSON)
			c.Abort()
			return
		}

		// Check user-based rate limiting if authenticated
		if userID, exists := c.Get("userID"); exists {
			userLimiter := rl.LimitByUser(userID)
			if !userLimiter.Allow() {
				c.Header("Retry-After", "3600")
				// Optimization: Use pre-allocated JSON response to avoid reflection and map allocations during rate limit rejection hot-path
				c.Data(http.StatusTooManyRequests, "application/json", rateLimitUserExceededJSON)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// StrictRateLimit returns a gin middleware for strict rate limiting (e.g. auth endpoints)
func (rl *RateLimiter) StrictRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := GetClientIP(c)

		// Check strict IP-based rate limiting
		ipLimiter := rl.StrictLimitByIP(clientIP)
		if !ipLimiter.Allow() {
			c.Header("Retry-After", "60")
			// Optimization: Use pre-allocated JSON response to avoid reflection and map allocations during strict rate limit rejection hot-path
			c.Data(http.StatusTooManyRequests, "application/json", strictRateLimitExceededJSON)
			c.Abort()
			return
		}

		c.Next()
	}
}

// StopCleanup stops the cleanup routine
func (rl *RateLimiter) StopCleanup() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// RateLimitInfo contains specific rate limit details
type RateLimitInfo struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

// RateLimitStatusResponse represents the current rate limit status
type RateLimitStatusResponse struct {
	IP        string         `json:"ip"`
	IPLimit   RateLimitInfo  `json:"ip_limit"`
	UserID    interface{}    `json:"user_id,omitempty"`
	UserLimit *RateLimitInfo `json:"user_limit,omitempty"`
}

// GetRateLimitStatus returns the current rate limit status for debugging
func (rl *RateLimiter) GetRateLimitStatus(c *gin.Context) RateLimitStatusResponse {
	clientIP := GetClientIP(c)
	ipLimiter := rl.LimitByIP(clientIP)

	status := RateLimitStatusResponse{
		IP: clientIP,
		IPLimit: RateLimitInfo{
			Limit:     100,
			Remaining: int(ipLimiter.Tokens()),
			Reset:     time.Now().Add(time.Minute).Unix(),
		},
	}

	if userID, exists := c.Get("userID"); exists {
		userLimiter := rl.LimitByUser(userID)
		status.UserID = userID
		status.UserLimit = &RateLimitInfo{
			Limit:     1000,
			Remaining: int(userLimiter.Tokens()),
			Reset:     time.Now().Add(time.Hour).Unix(),
		}
	}

	return status
}
