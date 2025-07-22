package errors

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Error represents a structured application error
type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Status    int    `json:"-"`
	Err       error  `json:"-"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
	Timestamp string `json:"timestamp"`
}

// Response converts Error to a standardized response format
func (e *Error) Response(c *gin.Context) gin.H {
	reqID := ""
	if c != nil {
		if val, exists := c.Get("request_id"); exists {
			reqID = val.(string)
		}
	}

	return gin.H{
		"error": gin.H{
			"code":       e.Code,
			"message":    e.Message,
			"request_id": reqID,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"details":    e.Details,
		},
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap supports error unwrapping
func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new Error
func New(code, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// ValidationError represents a validation error with details
type ValidationError struct {
	Message string
	Field   string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		Message: fmt.Sprintf(format, args...),
	}
}

// NewValidationErrorWithField creates a new validation error with field information
func NewValidationErrorWithField(field, format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		Message: fmt.Sprintf(format, args...),
		Field:   field,
	}
}

// Common error types
var (
	ErrValidation   = New("validation_error", "Invalid request data", http.StatusBadRequest)
	ErrUnauthorized = New("unauthorized", "Authentication required", http.StatusUnauthorized)
	ErrForbidden    = New("forbidden", "Access denied", http.StatusForbidden)
	ErrNotFound     = New("not_found", "Resource not found", http.StatusNotFound)
	ErrInternal     = New("internal_error", "Internal server error", http.StatusInternalServerError)

	// Database errors
	ErrDBConnection     = New("db_connection_error", "Database connection failed", http.StatusServiceUnavailable)
	ErrDBQuery          = New("db_query_error", "Database query failed", http.StatusInternalServerError)
	ErrDBMigration      = New("db_migration_error", "Database migration failed", http.StatusInternalServerError)
	ErrDBTransaction    = New("db_transaction_error", "Database transaction failed", http.StatusInternalServerError)
	ErrDBRecordExists   = New("db_record_exists", "Record already exists", http.StatusConflict)
	ErrDBRecordLocked   = New("db_record_locked", "Record is locked", http.StatusConflict)
	ErrDBOptimisticLock = New("db_optimistic_lock", "Optimistic lock conflict", http.StatusConflict)
	ErrDBConstraint     = New("db_constraint_violation", "Database constraint violation", http.StatusBadRequest)
	ErrDBTimeout        = New("db_timeout", "Database operation timed out", http.StatusGatewayTimeout)
)
