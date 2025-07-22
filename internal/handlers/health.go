package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthCheck performs a comprehensive health check
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   "1.0.0",
		Uptime:    getUptime(),
		Checks:    make(map[string]HealthCheck),
	}

	// Database health check
	dbCheck := h.checkDatabase()
	response.Checks["database"] = dbCheck

	// If any check fails, mark overall status as unhealthy
	for _, check := range response.Checks {
		if check.Status != "healthy" {
			response.Status = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
	}

	c.JSON(http.StatusOK, response)
}

// ReadinessCheck performs a readiness check for Kubernetes
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// Check if database is ready
	dbCheck := h.checkDatabase()
	if dbCheck.Status != "healthy" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not ready",
			"message": "Database not ready",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// LivenessCheck performs a liveness check for Kubernetes
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// checkDatabase checks the database connection
func (h *HealthHandler) checkDatabase() HealthCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB, err := h.db.DB()
	if err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Failed to get database connection: " + err.Error(),
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	// Check if we can execute a simple query
	var result int
	if err := h.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database query failed: " + err.Error(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Database is responsive",
		Data: map[string]interface{}{
			"driver": "sqlite",
		},
	}
}

// getUptime returns the application uptime
func getUptime() string {
	// This is a simplified version - in production, you'd track actual start time
	return "running"
}

// DetailedHealthCheck provides detailed health information
func (h *HealthHandler) DetailedHealthCheck(c *gin.Context) {
	response := struct {
		HealthResponse
		Environment string            `json:"environment"`
		BuildInfo   map[string]string `json:"build_info"`
	}{
		HealthResponse: HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().UTC(),
			Version:   "1.0.0",
			Uptime:    getUptime(),
			Checks:    make(map[string]HealthCheck),
		},
		Environment: gin.Mode(),
		BuildInfo: map[string]string{
			"go_version": "1.24.3",
			"git_commit": "unknown",
			"build_time": "unknown",
		},
	}

	// Database health check
	response.Checks["database"] = h.checkDatabase()

	// Add more detailed checks here as needed

	c.JSON(http.StatusOK, response)
}
