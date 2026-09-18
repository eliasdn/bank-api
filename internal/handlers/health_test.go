package handlers

import (
	"bank-api/internal/testutils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Healthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.InitTestDB()

	handler := NewHealthHandler(db)
	router := gin.New()
	router.GET("/health", handler.HealthCheck)
	router.GET("/health/ready", handler.ReadinessCheck)
	router.GET("/health/live", handler.LivenessCheck)
	router.GET("/health/detailed", handler.DetailedHealthCheck)

	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", body["status"])
	})

	t.Run("ReadinessCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health/ready", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "ready", body["status"])
	})

	t.Run("LivenessCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health/live", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "alive", body["status"])
	})

	t.Run("DetailedHealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health/detailed", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", body["status"])
	})
}

func TestHealthHandler_Unhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.InitTestDB()
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close() // Close database connection to force ping/query failure

	handler := NewHealthHandler(db)
	router := gin.New()
	router.GET("/health", handler.HealthCheck)
	router.GET("/health/ready", handler.ReadinessCheck)

	t.Run("HealthCheck Unhealthy Does Not Leak Internal Errors", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
		bodyStr := resp.Body.String()
		assert.NotContains(t, bodyStr, "sql: database is closed")
		assert.Contains(t, bodyStr, "Database ping failed")
	})

	t.Run("ReadinessCheck Unhealthy", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health/ready", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
		var body map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "not ready", body["status"])
	})
}
