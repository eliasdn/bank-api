package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger provides structured logging for HTTP requests
type RequestLogger struct {
	logger Logger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger Logger) *RequestLogger {
	return &RequestLogger{
		logger: logger,
	}
}

// RequestLog represents a structured log entry for HTTP requests
type RequestLog struct {
	RequestID    string        `json:"request_id"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	Query        string        `json:"query,omitempty"`
	IP           string        `json:"ip"`
	UserAgent    string        `json:"user_agent"`
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	ContentType  string        `json:"content_type,omitempty"`
	UserID       string        `json:"user_id,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// LoggingMiddleware creates a middleware that logs all HTTP requests
func (rl *RequestLogger) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate response time
		responseTime := time.Since(start)

		// Get user ID if authenticated
		var userID string
		if val, exists := c.Get("user_id"); exists {
			userID = val.(string)
		}

		// Create log entry
		logEntry := RequestLog{
			RequestID:    requestID,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Query:        c.Request.URL.RawQuery,
			IP:           GetClientIP(c),
			UserAgent:    c.Request.UserAgent(),
			StatusCode:   c.Writer.Status(),
			ResponseTime: responseTime,
			ContentType:  c.Writer.Header().Get("Content-Type"),
			UserID:       userID,
		}

		// Add error if any
		if len(c.Errors) > 0 {
			logEntry.Error = c.Errors.Last().Error()
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			rl.logger.Error("HTTP request error",
				"request", logEntry,
			)
		case c.Writer.Status() >= 400:
			rl.logger.Info("HTTP client error",
				"request", logEntry,
			)
		default:
			rl.logger.Info("HTTP request",
				"request", logEntry,
			)
		}
	}
}

// RequestContext returns a context with request information
func (rl *RequestLogger) RequestContext(c *gin.Context) context.Context {
	ctx := context.Background()

	if requestID, exists := c.Get("request_id"); exists {
		ctx = context.WithValue(ctx, "request_id", requestID)
	}

	if userID, exists := c.Get("user_id"); exists {
		ctx = context.WithValue(ctx, "user_id", userID)
	}

	return ctx
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) string {
	if val := ctx.Value("request_id"); val != nil {
		return val.(string)
	}
	return ""
}

// GetUserID extracts the user ID from context
func GetUserID(ctx context.Context) string {
	if val := ctx.Value("user_id"); val != nil {
		return val.(string)
	}
	return ""
}

// AuditLogger provides audit logging for sensitive operations
type AuditLogger struct {
	logger Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
	}
}

// AuditLog represents an audit log entry
type AuditLog struct {
	RequestID   string                 `json:"request_id"`
	UserID      string                 `json:"user_id"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	IP          string                 `json:"ip"`
	UserAgent   string                 `json:"user_agent"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Log logs an audit event
func (al *AuditLogger) Log(c *gin.Context, action, resource, resourceID, description string, metadata map[string]interface{}) {
	var userID string
	if val, exists := c.Get("user_id"); exists {
		userID = val.(string)
	}

	var requestID string
	if val, exists := c.Get("request_id"); exists {
		requestID = val.(string)
	}

	auditLog := AuditLog{
		RequestID:   requestID,
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		Timestamp:   time.Now().UTC(),
		IP:          GetClientIP(c),
		UserAgent:   c.Request.UserAgent(),
		Metadata:    metadata,
	}

	al.logger.Info("AUDIT", "audit", auditLog)
}

// LogWithContext logs an audit event with context
func (al *AuditLogger) LogWithContext(ctx context.Context, action, resource, resourceID, description string, metadata map[string]interface{}) {
	auditLog := AuditLog{
		RequestID:   GetRequestID(ctx),
		UserID:      GetUserID(ctx),
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		Timestamp:   time.Now().UTC(),
		Metadata:    metadata,
	}

	al.logger.Info("AUDIT", "audit", auditLog)
}
