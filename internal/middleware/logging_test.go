package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware_RequestIDValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		incomingHeader string
		expectSame     bool
	}{
		{
			name:           "valid custom request ID",
			incomingHeader: "req-12345-abcde",
			expectSame:     true,
		},
		{
			name:           "invalid request ID with CRLF injection",
			incomingHeader: "req-123\r\nX-Injected: malicious",
			expectSame:     false,
		},
		{
			name:           "invalid request ID with null byte",
			incomingHeader: "req-123\x00bad",
			expectSame:     false,
		},
		{
			name:           "invalid request ID exceeding 64 characters",
			incomingHeader: strings.Repeat("a", 65),
			expectSame:     false,
		},
		{
			name:           "empty request ID header",
			incomingHeader: "",
			expectSame:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			logger := NewSimpleLogger()
			reqLogger := NewRequestLogger(logger)
			r.Use(reqLogger.LoggingMiddleware())

			var capturedContextID string
			r.GET("/test", func(c *gin.Context) {
				capturedContextID = c.GetString("request_id")
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.incomingHeader != "" {
				req.Header.Set("X-Request-ID", tt.incomingHeader)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			respHeaderID := w.Header().Get("X-Request-ID")
			assert.NotEmpty(t, respHeaderID)
			assert.NotEmpty(t, capturedContextID)
			assert.Equal(t, respHeaderID, capturedContextID)

			if tt.expectSame {
				assert.Equal(t, tt.incomingHeader, respHeaderID)
			} else {
				assert.NotEqual(t, tt.incomingHeader, respHeaderID)
				assert.True(t, isValidRequestID(respHeaderID))
			}
		})
	}
}

type testCaptureLogger struct {
	lastMsg    string
	lastFields []interface{}
}

func (l *testCaptureLogger) Error(msg string, fields ...interface{}) {
	l.lastMsg = msg
	l.lastFields = fields
}

func (l *testCaptureLogger) Info(msg string, fields ...interface{}) {
	l.lastMsg = msg
	l.lastFields = fields
}

func TestLoggingMiddleware_UserAgentSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		userAgent     string
		expectedInLog string
	}{
		{
			name:          "clean user-agent",
			userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			expectedInLog: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		},
		{
			name:          "user-agent with CRLF injection",
			userAgent:     "Mozilla/5.0\r\nInjected-Header: evil\nAdmin-Log",
			expectedInLog: "Mozilla/5.0  Injected-Header: evil Admin-Log",
		},
		{
			name:          "user-agent with null bytes",
			userAgent:     "Mozilla/5.0\x00malicious",
			expectedInLog: "Mozilla/5.0 malicious",
		},
		{
			name:          "user-agent exceeding 512 characters is truncated",
			userAgent:     strings.Repeat("A", 600),
			expectedInLog: strings.Repeat("A", 512),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			captureLogger := &testCaptureLogger{}
			reqLogger := NewRequestLogger(captureLogger)
			r.Use(reqLogger.LoggingMiddleware())

			r.GET("/test-ua", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/test-ua", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.NotEmpty(t, captureLogger.lastFields)
			logEntry, ok := captureLogger.lastFields[1].(RequestLog)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedInLog, logEntry.UserAgent)
			assert.NotContains(t, logEntry.UserAgent, "\r")
			assert.NotContains(t, logEntry.UserAgent, "\n")
			assert.NotContains(t, logEntry.UserAgent, "\x00")
			assert.LessOrEqual(t, len(logEntry.UserAgent), 512)
		})
	}
}

func TestAuditLogger_UserAgentSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	captureLogger := &testCaptureLogger{}
	auditLogger := NewAuditLogger(captureLogger)

	r := gin.New()
	r.GET("/test-audit", func(c *gin.Context) {
		auditLogger.Log(c, "test_action", "account", "123", "Test Audit", nil)
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test-audit", nil)
	req.Header.Set("User-Agent", "EvilClient\r\nCRLF-Injection\x00")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, captureLogger.lastFields)
	entry, ok := captureLogger.lastFields[1].(AuditLog)
	assert.True(t, ok)
	assert.Equal(t, "EvilClient  CRLF-Injection ", entry.UserAgent)
	assert.NotContains(t, entry.UserAgent, "\r")
	assert.NotContains(t, entry.UserAgent, "\n")
	assert.NotContains(t, entry.UserAgent, "\x00")
}
