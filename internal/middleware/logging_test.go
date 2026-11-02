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
	lastLog RequestLog
}

func (l *testCaptureLogger) Error(msg string, fields ...interface{}) {
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "request" {
			if rl, ok := fields[i+1].(RequestLog); ok {
				l.lastLog = rl
			}
		}
	}
}

func (l *testCaptureLogger) Info(msg string, fields ...interface{}) {
	for i := 0; i < len(fields); i += 2 {
		if fields[i] == "request" {
			if rl, ok := fields[i+1].(RequestLog); ok {
				l.lastLog = rl
			}
		}
	}
}

func TestLoggingMiddleware_UserAgentSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		incomingUA    string
		expectedLogUA string
	}{
		{
			name:          "standard user-agent",
			incomingUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			expectedLogUA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		},
		{
			name:          "user-agent with CRLF injection",
			incomingUA:    "Mozilla/5.0\r\nInjected: malicious",
			expectedLogUA: "Mozilla/5.0  Injected: malicious",
		},
		{
			name:          "user-agent with null byte",
			incomingUA:    "TestClient\x00Injected",
			expectedLogUA: "TestClient Injected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			captureLogger := &testCaptureLogger{}
			reqLogger := NewRequestLogger(captureLogger)
			r.Use(reqLogger.LoggingMiddleware())

			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tt.incomingUA)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedLogUA, captureLogger.lastLog.UserAgent)
		})
	}
}
