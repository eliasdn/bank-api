package middleware

import (
	"bank-api/internal/errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorHandler provides centralized error handling for the application
type ErrorHandler struct {
	logger Logger
}

// Logger interface for logging errors
type Logger interface {
	Error(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
}

// NewErrorHandler creates a new error handler middleware
func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details,omitempty"`
	} `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
	Path      string    `json:"path"`
	Method    string    `json:"method"`
}

// HandleError handles errors and returns a standardized response
func (eh *ErrorHandler) HandleError(c *gin.Context, err error) {
	var statusCode int
	var errorCode string
	var message string
	var details map[string]interface{}

	// Determine error type and set appropriate response
	switch e := err.(type) {
	case *errors.Error:
		statusCode = e.Status
		errorCode = e.Code
		message = e.Message
		if e.Details != nil {
			details = map[string]interface{}{"details": e.Details}
		}
	case *errors.ValidationError:
		statusCode = http.StatusBadRequest
		errorCode = "validation_error"
		message = e.Message
		details = map[string]interface{}{
			"field": e.Field,
		}
	case error:
		// Default to internal server error for unknown errors
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "An unexpected error occurred"
		details = map[string]interface{}{"original_error": e.Error()}
	}

	// Log the error
	eh.logger.Error("Request error",
		"request_id", c.GetString("request_id"),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"error_code", errorCode,
		"error_message", message,
		"details", details,
		"status_code", statusCode,
	)

	// Build response
	response := ErrorResponse{
		Timestamp: time.Now().UTC(),
		RequestID: c.GetString("request_id"),
		Path:      c.Request.URL.Path,
		Method:    c.Request.Method,
	}
	response.Error.Code = errorCode
	response.Error.Message = message
	response.Error.Details = details

	c.JSON(statusCode, response)
}

// ErrorHandlerMiddleware returns a Gin middleware for error handling
func (eh *ErrorHandler) ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()

		// Check if there are any errors to handle
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()
			eh.HandleError(c, err.Err)
		}
	}
}

// RecoveryMiddleware recovers from panics and handles them gracefully
func (eh *ErrorHandler) RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		var err *errors.Error

		switch recovered := recovered.(type) {
		case string:
			err = errors.Wrap(nil, "panic_recovered", recovered, http.StatusInternalServerError)
		case error:
			err = errors.Wrap(recovered, "panic_recovered", "panic recovered", http.StatusInternalServerError)
		default:
			err = errors.New("panic_recovered", "panic recovered", http.StatusInternalServerError)
		}

		eh.logger.Error("Panic recovered",
			"request_id", c.GetString("request_id"),
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"panic", recovered,
		)

		eh.HandleError(c, err)
	})
}

// ValidationError creates a validation error response
func (eh *ErrorHandler) ValidationError(c *gin.Context, field string, message string) {
	err := errors.New("validation_error", message, http.StatusBadRequest)
	err.Details = map[string]interface{}{
		"field": field,
	}
	eh.HandleError(c, err)
}

// NotFoundError creates a not found error response
func (eh *ErrorHandler) NotFoundError(c *gin.Context, resource string) {
	err := errors.New("not_found", resource+" not found", http.StatusNotFound)
	eh.HandleError(c, err)
}

// UnauthorizedError creates an unauthorized error response
func (eh *ErrorHandler) UnauthorizedError(c *gin.Context, message string) {
	err := errors.New("unauthorized", message, http.StatusUnauthorized)
	eh.HandleError(c, err)
}

// ForbiddenError creates a forbidden error response
func (eh *ErrorHandler) ForbiddenError(c *gin.Context, message string) {
	err := errors.New("forbidden", message, http.StatusForbidden)
	eh.HandleError(c, err)
}
