package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	middleware := NewAuthMiddleware(config.LoadTestConfig())

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		expectedError  string
		expectUserID   interface{}
	}{
		{
			name: "Missing Authorization header",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authorization header is required",
		},
		{
			name: "Invalid Bearer token format",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("Authorization", "InvalidToken")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Bearer token not found",
		},
		{
			name: "Invalid JWT signature",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "user123",
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString([]byte("wrong-secret"))
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name: "Expired token",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "user123",
					"exp": time.Now().Add(-time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString([]byte("test-secret"))
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name: "Successful authentication",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "user123",
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString([]byte("test-secret"))
				req.Header.Set("Authorization", "Bearer "+tokenString)
				return req
			},
			expectedStatus: http.StatusOK,
			expectUserID:   "user123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = tt.setupRequest()

			handler := middleware.Authenticate()
			handler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			} else {
				userID, exists := c.Get("userID")
				assert.True(t, exists)
				assert.Equal(t, tt.expectUserID, userID)
			}
		})
	}
}
