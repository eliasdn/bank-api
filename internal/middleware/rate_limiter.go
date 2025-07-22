package middleware

import (
	"bank-api/internal/config"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	config *config.AppConfig
	// IP-based limiters
	ipLimiters map[string]*rate.Limiter
	ipMutex    sync.RWMutex
	// User-based limiters
	userLimiters map[string]*rate.Limiter
	userMutex    sync.RWMutex
	// Cleanup ticker
	cleanupTicker *time.Ticker
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *config.AppConfig) *RateLimiter {
	rl := &RateLimiter{
		config:       config,
		ipLimiters:   make(map[string]*rate.Limiter),
		userLimiters: make(map[string]*rate.Limiter),
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
	rl.ipMutex.RLock()
	limiter, exists := rl.ipLimiters[ip]
	rl.ipMutex.RUnlock()

	if !exists {
		rl.ipMutex.Lock()
		// Double-check after acquiring write lock
		limiter, exists = rl.ipLimiters[ip]
		if !exists {
			// 100 requests per minute per IP
			limiter = rate.NewLimiter(rate.Every(time.Minute), 100)
			rl.ipLimiters[ip] = limiter
		}
		rl.ipMutex.Unlock()
	}

	return limiter
}

// LimitByUser creates or gets a rate limiter for the given user ID
func (rl *RateLimiter) LimitByUser(userID string) *rate.Limiter {
	rl.userMutex.RLock()
	limiter, exists := rl.userLimiters[userID]
	rl.userMutex.RUnlock()

	if !exists {
		rl.userMutex.Lock()
		// Double-check after acquiring write lock
		limiter, exists = rl.userLimiters[userID]
		if !exists {
			// 1000 requests per hour per authenticated user
			limiter = rate.NewLimiter(rate.Every(time.Hour), 1000)
			rl.userLimiters[userID] = limiter
		}
		rl.userMutex.Unlock()
	}

	return limiter
}

// cleanupExpiredLimiters removes expired rate limiters
func (rl *RateLimiter) cleanupExpiredLimiters() {
	for range rl.cleanupTicker.C {
		// Clean IP limiters older than 24 hours
		rl.ipMutex.Lock()
		for ip, limiter := range rl.ipLimiters {
			// Simple cleanup - remove if no recent activity
			// This is a basic implementation - could be enhanced with last access time
			if limiter.Tokens() == 100 { // Full bucket indicates no recent usage
				delete(rl.ipLimiters, ip)
			}
		}
		rl.ipMutex.Unlock()

		// Clean user limiters older than 24 hours
		rl.userMutex.Lock()
		for userID, limiter := range rl.userLimiters {
			if limiter.Tokens() == 1000 { // Full bucket indicates no recent usage
				delete(rl.userLimiters, userID)
			}
		}
		rl.userMutex.Unlock()
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
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":        "rate_limit_exceeded",
					"message":     "Too many requests from this IP address",
					"retry_after": "60",
				},
			})
			c.Abort()
			return
		}

		// Check user-based rate limiting if authenticated
		if userID, exists := c.Get("user_id"); exists {
			userStr := strconv.FormatUint(uint64(userID.(uint)), 10)
			userLimiter := rl.LimitByUser(userStr)
			if !userLimiter.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":        "rate_limit_exceeded",
						"message":     "Too many requests for this user account",
						"retry_after": "3600",
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// getClientIP extracts the real client IP from the request
func (rl *RateLimiter) getClientIP(c *gin.Context) string {
	return GetClientIP(c)
}

// StopCleanup stops the cleanup routine
func (rl *RateLimiter) StopCleanup() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// GetRateLimitStatus returns the current rate limit status for debugging
func (rl *RateLimiter) GetRateLimitStatus(c *gin.Context) gin.H {
	clientIP := GetClientIP(c)
	ipLimiter := rl.LimitByIP(clientIP)

	status := gin.H{
		"ip": clientIP,
		"ip_limit": gin.H{
			"limit":     100,
			"remaining": int(ipLimiter.Tokens()),
			"reset":     time.Now().Add(time.Minute).Unix(),
		},
	}

	if userID, exists := c.Get("user_id"); exists {
		userStr := strconv.FormatUint(uint64(userID.(uint)), 10)
		userLimiter := rl.LimitByUser(userStr)
		status["user_id"] = userID
		status["user_limit"] = gin.H{
			"limit":     1000,
			"remaining": int(userLimiter.Tokens()),
			"reset":     time.Now().Add(time.Hour).Unix(),
		}
	}

	return status
}
