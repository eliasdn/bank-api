package middleware

import (
	"bank-api/internal/errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockLogger for testing
type MockLogger struct {
	LastMessage string
	LastFields  []interface{}
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.LastMessage = msg
	m.LastFields = fields
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.LastMessage = msg
	m.LastFields = fields
}

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("HandleError with AppError", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			err := errors.New("test_error", "Test error message", http.StatusBadRequest)
			eh.HandleError(c, err)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "test_error")
		assert.Contains(t, w.Body.String(), "Test error message")
		assert.Contains(t, mockLogger.LastMessage, "Request error")
	})

	t.Run("HandleError with ValidationError", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			err := errors.NewValidationErrorWithField("username", "invalid username format")
			eh.HandleError(c, err)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation_error")
		assert.Contains(t, w.Body.String(), "invalid username format")
	})

	t.Run("HandleError with generic error", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			err := assert.AnError
			eh.HandleError(c, err)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal_error")
		assert.Contains(t, w.Body.String(), "An unexpected error occurred")
	})

	t.Run("ErrorHandlerMiddleware", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.Use(eh.ErrorHandlerMiddleware())
		router.GET("/test", func(c *gin.Context) {
			err := errors.New("middleware_error", "Middleware error", http.StatusConflict)
			c.Error(err)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "middleware_error")
	})

	t.Run("RecoveryMiddleware", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.Use(eh.RecoveryMiddleware())
		router.GET("/test", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "panic_recovered")
		assert.Contains(t, mockLogger.LastMessage, "Request error")
	})

	t.Run("ValidationError helper", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			eh.ValidationError(c, "email", "invalid email format")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "validation_error")
		assert.Contains(t, w.Body.String(), "invalid email format")
	})

	t.Run("NotFoundError helper", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			eh.NotFoundError(c, "user")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "not_found")
		assert.Contains(t, w.Body.String(), "user not found")
	})

	t.Run("UnauthorizedError helper", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			eh.UnauthorizedError(c, "invalid credentials")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
		assert.Contains(t, w.Body.String(), "invalid credentials")
	})

	t.Run("ForbiddenError helper", func(t *testing.T) {
		mockLogger := &MockLogger{}
		eh := NewErrorHandler(mockLogger)

		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			eh.ForbiddenError(c, "insufficient permissions")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "forbidden")
		assert.Contains(t, w.Body.String(), "insufficient permissions")
	})
}
