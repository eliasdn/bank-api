package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() db.DBInterface {
	// Use in-memory SQLite for testing
	gdb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	gdb.AutoMigrate(&models.User{})
	return db.NewDBAdapter(gdb)
}

func TestLoginTimingAttack(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	database := setupTestDB()
	cfg := config.LoadTestConfig()
	// Increase Bcrypt cost to make the difference more noticeable
	cfg.Security.BcryptCost = 14

	handler := &Handler{
		DB:     database,
		Config: cfg,
	}

	// Create a user
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), cfg.Security.BcryptCost)
	user := models.User{
		Username:     "validuser",
		Email:        "valid@example.com",
		PasswordHash: string(hashedPassword),
		FullName:     "Valid User",
	}
	database.Create(&user)

	router := gin.Default()
	router.POST("/login", handler.LoginUser)

	// Measure time for non-existent user
	start := time.Now()
	reqBody, _ := json.Marshal(LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	})
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	durationNonExistent := time.Since(start)

	// Measure time for existing user with wrong password
	start = time.Now()
	reqBody, _ = json.Marshal(LoginRequest{
		Username: "validuser",
		Password: "wrongpassword",
	})
	req, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	durationExisting := time.Since(start)

	t.Logf("Duration for non-existent user: %v", durationNonExistent)
	t.Logf("Duration for existing user (wrong password): %v", durationExisting)

	// In a real timing attack, the existing user check takes significantly longer
	// because of bcrypt comparison.
	// We expect durationExisting to be much larger than durationNonExistent
	// if the vulnerability exists.
	// The ratio test is more robust than absolute time difference.
	if durationExisting > durationNonExistent*5 {
		t.Log("Vulnerability CONFIRMED: Valid user check took significantly longer.")
	} else {
		t.Log("Vulnerability NOT confirmed (or noise too high).")
	}
}
