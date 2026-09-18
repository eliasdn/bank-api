package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/internal/config"
	"bank-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterUser_ValidationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupTestDB()
	cfg := config.LoadTestConfig()

	handler := &Handler{
		DB:     database,
		Config: cfg,
	}

	router := gin.Default()
	router.POST("/register", handler.RegisterUser)

	// Registration with weak password should be rejected with 400 Bad Request
	reqBody, _ := json.Marshal(RegisterRequest{
		Username: "validuser",
		Email:    "valid@example.com",
		Password: "weakpassword", // missing uppercase, number, special char
		FullName: "Valid User",
	})
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupTestDB()
	cfg := config.LoadTestConfig()

	handler := &Handler{
		DB:     database,
		Config: cfg,
	}

	router := gin.Default()
	router.POST("/register", handler.RegisterUser)

	reqBody, _ := json.Marshal(RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "ValidPassword123!",
		FullName: "New User",
	})
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var user models.User
	err := database.Where("username = ?", "newuser").First(&user).Error
	assert.NoError(t, err)
	assert.Equal(t, "new@example.com", user.Email)
}
